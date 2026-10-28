using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.IO;
using System.Net;
using System.Net.Sockets;
using System.Threading;
using Renki.Bridge;
using Renki.Protocol;
using Xunit;
using Xunit.Abstractions;

namespace Renki.Tests
{
    /// <summary>
    /// The C# bridge client against the real Go controller. RENKI_CONTROLLER_BIN
    /// names the controller binary; `just plugin test` builds it and sets it.
    /// Without it these tests say so and pass, so a bare `dotnet test` still
    /// works on a machine with no Go.
    /// </summary>
    public sealed class InteropTests : IDisposable
    {
        readonly ITestOutputHelper output;
        readonly string bin = Environment.GetEnvironmentVariable("RENKI_CONTROLLER_BIN");
        readonly List<Process> controllers = new List<Process>();
        readonly string dir = Directory.CreateTempSubdirectory("renki-interop-").FullName;

        public InteropTests(ITestOutputHelper output)
        {
            this.output = output;
        }

        sealed class TestLog : ILog
        {
            readonly ITestOutputHelper o;
            public TestLog(ITestOutputHelper o) => this.o = o;
            public void Info(string m) => Write("info", m);
            public void Warn(string m) => Write("warn", m);
            public void Debug(string m) => Write("debug", m);

            void Write(string level, string m)
            {
                try { o.WriteLine($"{level}: {m}"); } catch (InvalidOperationException) { }
            }
        }

        bool Available()
        {
            if (!string.IsNullOrEmpty(bin) && File.Exists(bin)) return true;
            output.WriteLine("RENKI_CONTROLLER_BIN is not set; skipped. Run `just plugin test`.");
            return false;
        }

        static int FreePort()
        {
            var l = new TcpListener(IPAddress.Loopback, 0);
            l.Start();
            var port = ((IPEndPoint)l.LocalEndpoint).Port;
            l.Stop();
            return port;
        }

        Process StartController(int port)
        {
            var config = Path.Combine(dir, "controller.json");
            File.WriteAllText(config,
                $"{{\"listen\":\"127.0.0.1:{port}\",\"token\":\"sekrit\",\"owners\":[\"Sothr\"],\"agentName\":\"Bjorn\"}}");
            var p = Process.Start(new ProcessStartInfo(bin, $"-config {config}")
            {
                RedirectStandardError = true,
                RedirectStandardOutput = true,
                UseShellExecute = false,
            });
            p.ErrorDataReceived += (_, e) => { if (e.Data != null) Log("controller: " + e.Data); };
            p.BeginErrorReadLine();
            controllers.Add(p);
            WaitFor(() => CanConnect(port), "controller listening");
            return p;
        }

        void Log(string line)
        {
            try { output.WriteLine(line); } catch (InvalidOperationException) { }
        }

        static bool CanConnect(int port)
        {
            try
            {
                using (var c = new TcpClient()) c.Connect(IPAddress.Loopback, port);
                return true;
            }
            catch (SocketException)
            {
                return false;
            }
        }

        static void WaitFor(Func<bool> cond, string what, double seconds = 10)
        {
            var sw = Stopwatch.StartNew();
            while (!cond())
            {
                if (sw.Elapsed.TotalSeconds > seconds) throw new TimeoutException("timed out waiting for " + what);
                Thread.Sleep(20);
            }
        }

        BridgeClient Client(int port, string token)
        {
            var c = new BridgeClient(new Uri($"ws://127.0.0.1:{port}/ws/"), new TestLog(output))
            {
                Backoff = _ => TimeSpan.FromMilliseconds(200),
            };
            c.SetHello(new Hello
            {
                Token = token,
                Plugin = "interop",
                Game = "test",
                Phase = Wire.PhaseInWorld,
                Character = "Bjorn",
                World = "Test",
                Capabilities = new List<string>(TaskNames.M1),
            });
            return c;
        }

        static T Next<T>(BridgeClient c, double seconds = 5) where T : Message
        {
            var sw = Stopwatch.StartNew();
            while (sw.Elapsed.TotalSeconds < seconds)
            {
                if (c.TryReceive(out var m))
                {
                    if (m is T t) return t;
                    continue;
                }
                Thread.Sleep(10);
            }
            throw new TimeoutException("no " + typeof(T).Name);
        }

        [Fact]
        public void HandshakeChatAndReconnect()
        {
            if (!Available()) return;
            var port = FreePort();
            var controller = StartController(port);
            using (var client = Client(port, "sekrit"))
            {
                client.Start();
                var welcome = Next<Welcome>(client);
                Assert.Equal("Bjorn", welcome.AgentName);
                Assert.Equal(new[] { "Sothr" }, welcome.Owners);
                Assert.True(client.Connected);
                Assert.Equal(1, client.ConnectionId);

                client.Send(new Chat
                {
                    From = new ChatSender { Name = "Sothr", PlayerId = "1", Owner = true },
                    Channel = Wire.ChannelNormal,
                    Text = "bjorn, follow me",
                    Pos = new double[] { 1, 2, 3 },
                    Dist = 4,
                });
                var follow = Next<TaskRequest>(client);
                Assert.Equal("follow", follow.Task);
                Assert.Equal("Sothr", ((FollowArgs)TaskArgs.Parse(follow)).Player);
                var say = Next<TaskRequest>(client);
                Assert.Equal("say", say.Task);

                // Pings are answered by the client itself, so the controller keeps
                // talking: well past one ping interval, it still has the plugin.
                Thread.Sleep(3500);
                Assert.True(client.SecondsSinceController < 3.5);

                // The controller restarts; the client finds it again and gets a new session.
                controller.Kill();
                controller.WaitForExit();
                WaitFor(() => !client.Connected, "disconnect");
                StartController(port);
                Next<Welcome>(client, 10);
                Assert.Equal(2, client.ConnectionId);
            }
        }

        [Fact]
        public void BadTokenNeverGetsWelcome()
        {
            if (!Available()) return;
            var port = FreePort();
            StartController(port);
            using (var client = Client(port, "wrong"))
            {
                client.Start();
                Thread.Sleep(1500);
                Assert.False(client.Connected);
                Assert.False(client.TryReceive(out _));
            }
        }

        [Fact]
        public void UnknownTypeIsAnsweredAndTheConnectionStays()
        {
            // The controller never sends an unknown type, so this checks the
            // client's decoder path directly.
            var e = Assert.Throws<ProtocolException>(() =>
                Codec.Decode("{\"v\":1,\"type\":\"dance\",\"id\":\"d-1\",\"seq\":1,\"t\":0}", Sender.Controller));
            Assert.Equal(DecodeFailure.UnknownType, e.Failure);
            Assert.Equal("d-1", e.FrameId);
        }

        public void Dispose()
        {
            foreach (var p in controllers)
            {
                try { if (!p.HasExited) p.Kill(); } catch { }
            }
            try { Directory.Delete(dir, true); } catch { }
        }
    }
}
