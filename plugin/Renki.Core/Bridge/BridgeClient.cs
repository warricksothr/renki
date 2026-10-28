using System;
using System.Collections.Concurrent;
using System.Diagnostics;
using System.IO;
using System.Net.WebSockets;
using System.Text;
using System.Threading;
using System.Threading.Tasks;
using Renki.Protocol;

namespace Renki.Bridge
{
    /// <summary>Where the bridge writes its log, so Core does not depend on BepInEx.</summary>
    public interface ILog
    {
        void Info(string message);
        void Warn(string message);
        void Debug(string message);
    }

    /// <summary>
    /// The plugin's end of the connection (docs/BRIDGE-PROTOCOL.md sections 2
    /// and 5). The socket lives on a background task and never touches the
    /// game. Inbound messages wait in a queue that the game's main thread
    /// drains with TryReceive; outbound messages are queued by Send and
    /// written by the background task. Pings are answered here, so a busy
    /// frame cannot make the controller think the plugin is gone.
    /// </summary>
    public sealed class BridgeClient : IDisposable
    {
        public static readonly TimeSpan MaxBackoff = TimeSpan.FromSeconds(30);

        readonly Uri url;
        readonly ILog log;
        readonly ConcurrentQueue<Message> inbound = new ConcurrentQueue<Message>();
        readonly ConcurrentQueue<Message> outbound = new ConcurrentQueue<Message>();
        readonly SemaphoreSlim outboundReady = new SemaphoreSlim(0);
        readonly CancellationTokenSource stop = new CancellationTokenSource();
        readonly Stopwatch clock = Stopwatch.StartNew();

        Hello helloTemplate = new Hello();
        Task loop;
        long lastReceivedTicks = -1;
        long gameTimeBits;
        long seq;
        int connectionId;
        volatile bool welcomed;

        /// <summary>Called with each wait before a reconnect, for tests.</summary>
        public Func<int, TimeSpan> Backoff = attempt =>
            TimeSpan.FromSeconds(Math.Min(MaxBackoff.TotalSeconds, Math.Pow(2, attempt)));

        public BridgeClient(Uri url, ILog log)
        {
            this.url = url;
            this.log = log;
        }

        /// <summary>True between a welcome and the end of that connection.</summary>
        public bool Connected => welcomed;

        /// <summary>Counts welcomes, so the main thread can tell a new session from the old one.</summary>
        public int ConnectionId => Volatile.Read(ref connectionId);

        /// <summary>Seconds since anything arrived from the controller; infinite before the first message.</summary>
        public double SecondsSinceController
        {
            get
            {
                var last = Interlocked.Read(ref lastReceivedTicks);
                return last < 0 ? double.PositiveInfinity : (clock.ElapsedTicks - last) / (double)Stopwatch.Frequency;
            }
        }

        /// <summary>The hello sent on every (re)connect. The main thread keeps it current.</summary>
        public void SetHello(Hello hello) => Volatile.Write(ref helloTemplate, hello);

        /// <summary>The newest synced game time, stamped as <c>t</c> on anything sent without one.</summary>
        public double GameTime
        {
            get => BitConverter.Int64BitsToDouble(Interlocked.Read(ref gameTimeBits));
            set => Interlocked.Exchange(ref gameTimeBits, BitConverter.DoubleToInt64Bits(value));
        }

        public void Start()
        {
            if (loop == null) loop = Task.Run(() => RunAsync(stop.Token));
        }

        public bool TryReceive(out Message message) => inbound.TryDequeue(out message);

        /// <summary>Queues m for the current connection. Dropped when there is none,
        /// because nothing from an old session should reach a new one.</summary>
        public void Send(Message m)
        {
            if (!welcomed) return;
            Enqueue(m);
        }

        void Enqueue(Message m)
        {
            if (m.T == 0) m.T = GameTime;
            outbound.Enqueue(m);
            outboundReady.Release();
        }

        async Task RunAsync(CancellationToken ct)
        {
            var attempt = 0;
            while (!ct.IsCancellationRequested)
            {
                try
                {
                    if (await SessionAsync(ct).ConfigureAwait(false)) attempt = 0;
                }
                catch (OperationCanceledException) when (ct.IsCancellationRequested)
                {
                    return;
                }
                catch (Exception e)
                {
                    log.Debug("bridge: " + e.Message);
                }
                welcomed = false;
                var wait = Backoff(attempt++);
                log.Info($"bridge: disconnected; retrying in {wait.TotalSeconds:0} s");
                try
                {
                    await Task.Delay(wait, ct).ConfigureAwait(false);
                }
                catch (OperationCanceledException)
                {
                    return;
                }
            }
        }

        /// <summary>One connection, from dial to close. Returns whether it got as far as a welcome.</summary>
        async Task<bool> SessionAsync(CancellationToken ct)
        {
            using (var ws = new ClientWebSocket())
            using (var session = CancellationTokenSource.CreateLinkedTokenSource(ct))
            {
                using (var dial = CancellationTokenSource.CreateLinkedTokenSource(ct))
                {
                    dial.CancelAfter(TimeSpan.FromSeconds(10));
                    await ws.ConnectAsync(url, dial.Token).ConfigureAwait(false);
                }
                log.Info("bridge: connected to " + url);
                while (outbound.TryDequeue(out _)) { }
                Interlocked.Exchange(ref seq, 0);
                var hello = Volatile.Read(ref helloTemplate);
                hello.T = GameTime;
                await WriteAsync(ws, hello, session.Token).ConfigureAwait(false);

                var writer = WriteLoopAsync(ws, session.Token);
                var gotWelcome = false;
                try
                {
                    while (ws.State == WebSocketState.Open)
                    {
                        var frame = await ReadFrameAsync(ws, session.Token).ConfigureAwait(false);
                        if (frame == null) break;
                        Interlocked.Exchange(ref lastReceivedTicks, clock.ElapsedTicks);
                        Message msg;
                        try
                        {
                            msg = Codec.Decode(frame, Sender.Controller);
                        }
                        catch (ProtocolException e)
                        {
                            log.Warn("bridge: bad message from controller: " + e.Message);
                            Enqueue(new Error { Re = e.FrameId, Code = Wire.BadMessage, Detail = e.Message });
                            continue;
                        }
                        switch (msg)
                        {
                            case Ping ping:
                                Enqueue(new Pong { Re = ping.Id });
                                continue;
                            case Welcome _:
                                gotWelcome = true;
                                Interlocked.Increment(ref connectionId);
                                welcomed = true;
                                break;
                        }
                        inbound.Enqueue(msg);
                    }
                    if (ws.State == WebSocketState.CloseReceived)
                    {
                        log.Info($"bridge: controller closed ({(int?)ws.CloseStatus} {ws.CloseStatusDescription})");
                        await ws.CloseOutputAsync(WebSocketCloseStatus.NormalClosure, "", CancellationToken.None)
                            .ConfigureAwait(false);
                    }
                }
                finally
                {
                    welcomed = false;
                    session.Cancel();
                    try { await writer.ConfigureAwait(false); } catch { }
                }
                return gotWelcome;
            }
        }

        async Task WriteLoopAsync(ClientWebSocket ws, CancellationToken ct)
        {
            while (!ct.IsCancellationRequested)
            {
                await outboundReady.WaitAsync(ct).ConfigureAwait(false);
                if (outbound.TryDequeue(out var m)) await WriteAsync(ws, m, ct).ConfigureAwait(false);
            }
        }

        async Task WriteAsync(ClientWebSocket ws, Message m, CancellationToken ct)
        {
            m.Seq = (ulong)Interlocked.Increment(ref seq);
            var bytes = Encoding.UTF8.GetBytes(Codec.Encode(m));
            await ws.SendAsync(new ArraySegment<byte>(bytes), WebSocketMessageType.Text, true, ct).ConfigureAwait(false);
        }

        static async Task<string> ReadFrameAsync(ClientWebSocket ws, CancellationToken ct)
        {
            var buffer = new byte[16 * 1024];
            using (var ms = new MemoryStream())
            {
                while (true)
                {
                    var r = await ws.ReceiveAsync(new ArraySegment<byte>(buffer), ct).ConfigureAwait(false);
                    if (r.MessageType == WebSocketMessageType.Close) return null;
                    ms.Write(buffer, 0, r.Count);
                    if (ms.Length > 1 << 20) throw new InvalidDataException("frame over 1 MiB");
                    if (r.EndOfMessage) return Encoding.UTF8.GetString(ms.GetBuffer(), 0, (int)ms.Length);
                }
            }
        }

        /// <summary>Says bye on a live connection, then stops for good.</summary>
        public void Dispose()
        {
            if (welcomed)
            {
                Enqueue(new Bye { Reason = "quit" });
                Thread.Sleep(100);
            }
            stop.Cancel();
            try { loop?.Wait(TimeSpan.FromSeconds(2)); } catch { }
        }
    }
}
