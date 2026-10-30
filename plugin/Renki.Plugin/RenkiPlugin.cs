using System;
using System.Collections.Generic;
using System.Linq;
using BepInEx;
using BepInEx.Configuration;
using HarmonyLib;
using Renki.Bridge;
using Renki.Perception;
using Renki.Protocol;
using Renki.Tasks;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>
    /// renki's body inside Valheim. It dials the controller, reports what the
    /// bot sees and hears, and carries out the tasks it is sent, within the
    /// safety switches in its own config. With no controller it does nothing
    /// and the game plays as normal.
    /// </summary>
    [BepInPlugin(Guid, Name, Version)]
    public sealed class RenkiPlugin : BaseUnityPlugin
    {
        /// <summary>The plugin's log, for code that is not the plugin itself.</summary>
        internal static BepInEx.Logging.ManualLogSource Log;

        public const string Guid = "com.warricksothr.renki", Name = "renki", Version = "0.1.0";

        /// <summary>No controller message for this long stops the bot (section 8).</summary>
        public const double DeadManSeconds = 10;

        internal static RenkiPlugin Instance;

        ConfigEntry<string> controllerUrl;
        ConfigEntry<string> token;
        readonly SafetyConfig safety = new SafetyConfig();

        BridgeClient client;
        TaskRunner runner;
        Reporter reporter;
        readonly Perceiver perceiver = new Perceiver();
        readonly TerrainReporter terrain = new TerrainReporter();
        InventoryReporter inventory;
        readonly CameraReporter cameraReporter = new CameraReporter();
        readonly CaptureBeacon captureBeacon = new CaptureBeacon();
        bool captureBeaconEnabled;
        internal CaptureProof CaptureProof => captureBeacon.Proof;
        internal bool CameraActive => runner.InSession && client.Connected && !loggingOut && Phase() == Wire.PhaseInWorld && Player.m_localPlayer != null;
        internal int CameraConnection => client.ConnectionId;
        internal string CameraTask => runner.Current?.Id;
        internal void SendScreenshot(Screenshot frame, int connection) => client.SendScreenshot(frame, connection);
        Harmony harmony;

        Welcome welcome;
        double stateHz = 5;
        double perceptionHz = 1;
        readonly Renki.Perception.ObservationSchedule stateSchedule = new Renki.Perception.ObservationSchedule();
        float nextPerception;
        float nextHello;
        bool loggingOut;
        bool? sharePosition;

        internal IReadOnlyCollection<string> Owners => welcome?.Owners ?? (IReadOnlyCollection<string>)Array.Empty<string>();

        void Awake()
        {
            Instance = this;
            controllerUrl = Config.Bind("Bridge", "ControllerUrl", "ws://127.0.0.1:8765/ws/",
                "Where the controller listens (docs/BRIDGE-PROTOCOL.md section 2).");
            token = Config.Bind("Bridge", "Token", "",
                "Sent in hello. Required when the controller listens on anything but loopback.");
            captureBeaconEnabled = Config.Bind("Streaming", "CaptureBeacon", false,
                "Draw a tiny changing in-frame nonce for verified OBS capture. Requires restart.").Value;
            terrain.Namespace = Config.Bind("Mapping", "Namespace", "local",
                "Stable server/save namespace for atlas data; use a distinct value for unrelated save copies.").Value;
            terrain.Epoch = Config.Bind("Mapping", "Epoch", "1",
                "Change after restoring, resetting or forking a save to keep incompatible maps separate.").Value;
            terrain.Source = Config.Bind("Mapping", "Source", System.Guid.NewGuid().ToString("N"),
                "Persistent observer ID for map provenance.").Value;
            safety.Enabled = Config.Bind("Safety", "Enabled", true,
                "Master switch. Off, the plugin accepts no task at all.").Value;
            safety.AllowMovement = Config.Bind("Safety", "AllowMovement", true, "move_to, follow, look_at.").Value;
            safety.AllowChat = Config.Bind("Safety", "AllowChat", true, "say and emote.").Value;
            safety.AllowGather = Config.Bind("Safety", "AllowGather", false, "harvest, pickup (M2).").Value;
            safety.AllowDrop = Config.Bind("Safety", "AllowDrop", true,
                "drop: put items down at the bot's feet, never what it has equipped. The controller drops to free a bot that carries too much to move.").Value;
            safety.AllowCombat = Config.Bind("Safety", "AllowCombat", false, "Explicit attack (M3). Flee remains available with AllowMovement.").Value;
            safety.AllowSailing = Config.Bind("Safety", "AllowSailing", false, "Steering ships (later).").Value;
            safety.AllowBossSummoning = Config.Bind("Safety", "AllowBossSummoning", false,
                "Using offering bowls (M2).").Value;
            safety.AllowRawControls = Config.Bind("Safety", "AllowRawControls", false,
                "The debug `controls` task.").Value;
            Doorman.Enabled = Config.Bind("Safety", "AllowDoors", true,
                "Open closed doors in the bot's way, and shut them behind it.").Value;

            Log = Logger;
            Tracer.Sink = t =>
            {
                t.Task = runner?.Current?.Id;
                if (client != null && client.Connected) Send(t);
            };
            runner = new TaskRunner(safety, TaskNames.Supported, new GameTaskHost(safety), Send, () => Time.unscaledTimeAsDouble);
            Driver.OnFixedStep = dt => runner.FixedTick(dt);
            reporter = new Reporter(this);
            TerrainReporter.Current = terrain;
            inventory = new InventoryReporter(this);

            harmony = new Harmony(Guid);
            harmony.PatchAll(typeof(RenkiPlugin).Assembly);

            client = new BridgeClient(new Uri(controllerUrl.Value), new BepLog(Logger));
            client.SetHello(BuildHello());
            client.Start();
            Logger.LogInfo($"{Name} {Version} loaded; dialling {controllerUrl.Value}; " +
                           $"tasks allowed by config: {string.Join(", ", TaskNames.Supported.Where(safety.Permits))}");
        }

        void Update()
        {
            var now = Time.unscaledTime;
            client.GameTime = ZNet.instance != null ? ZNet.instance.GetTimeSeconds() : 0;
            if (now >= nextHello)
            {
                client.SetHello(BuildHello());
                nextHello = now + 1f;
            }

            while (client.TryReceive(out var msg)) Handle(msg);

            // The dead-man rule: a silent controller stops everything, and
            // nothing resumes until it says welcome again.
            if (runner.InSession && client.SecondsSinceController > DeadManSeconds)
            {
                Logger.LogWarning($"no controller message for {DeadManSeconds} s; stopping");
                EndSession("dead-man");
            }

            var phase = Phase();
            captureBeacon.Update(captureBeaconEnabled && CameraActive);
            cameraReporter.Update(this, CameraActive);
            var player = Player.m_localPlayer;
            var world = ZNet.instance != null ? ZNet.instance.GetWorldName() ?? "" : "";
            // Inventory is sent on change, often as soon as the player
            // appears between periodic state ticks. Establish its world
            // first, then resend observations on phase/world/player changes.
            // Logout revokes control before the game finishes unloading. Keep
            // reporting that transition so the controller can observe menu.
            if ((runner.InSession || loggingOut) && stateSchedule.Due(now, phase, world, player,
                1 / Math.Max(0.2, stateHz), out var scopeChanged))
            {
                var state = reporter.State(phase, runner.Current);
                if (state != null) Send(state);
                if (scopeChanged && runner.InSession)
                {
                    inventory.Resend(); perceiver.Resend(); terrain.Resend();
                    nextPerception = 0;
                }
            }
            if (runner.InSession && safety.Enabled && sharePosition.HasValue && Player.m_localPlayer != null && ZNet.instance != null &&
                ZNet.instance.IsReferencePositionPublic() != sharePosition.Value)
                ZNet.instance.SetPublicReferencePosition(sharePosition.Value);
            reporter.Update(runner.InSession);
            inventory.Update(runner.InSession && (phase == Wire.PhaseInWorld || phase == Wire.PhaseDead));
            if (runner.InSession && Player.m_localPlayer != null && !Player.m_localPlayer.IsDead())
            {
                WorldNavigation.Update(Player.m_localPlayer);
            }
            if (runner.InSession) Doorman.Tick(Player.m_localPlayer);
            // Publish the current world before any terrain patch, including
            // the first update after a new welcome.
            if (runner.InSession && Player.m_localPlayer != null && !Player.m_localPlayer.IsDead())
                terrain.Update(this, Player.m_localPlayer, perceiver.Radius);
            if (runner.InSession && now >= nextPerception)
            {
                var p = Player.m_localPlayer;
                if (p != null && !p.IsDead())
                {
                    Send(perceiver.Scan(p));
                    var layout = perceiver.TakeLayout();
                    if (layout != null) Send(layout);
                }
                nextPerception = now + (float)(1 / Math.Max(0.1, perceptionHz));
            }
        }

        void OnGUI() { captureBeacon.Draw(); }

        void Handle(Message msg)
        {
            switch (msg)
            {
                case Welcome w:
                    // Reconnecting the bridge must not undo an explicit logout.
                    if (loggingOut) { stateSchedule.Reset(); break; }
                    welcome = w;
                    cameraReporter.Configure(w.Screenshot);
                    client.ClearScreenshots();
                    sharePosition = w.SharePosition;
                    stateHz = w.Rates.StateHz > 0 ? w.Rates.StateHz : 5;
                    perceptionHz = w.Rates.PerceptionHz > 0 ? w.Rates.PerceptionHz : 1;
                    Perceive(w.Perception);
                    runner.Welcome(w.AllowedTasks);
                    inventory.Resend();
                    perceiver.Resend();
                    terrain.Resend();
                    stateSchedule.Reset();
                    Driver.Active = true;
                    var name = w.AgentName != "" ? w.AgentName : "the character's name";
                    Logger.LogInfo($"session started, answering to {name}; owners {string.Join(", ", w.Owners)}; " +
                                   $"allowed {string.Join(", ", runner.Allowed)}");
                    break;
                case TaskRequest t:
                    runner.Handle(t);
                    break;
                case Config c:
                    if (c.Screenshot != null) { cameraReporter.Configure(c.Screenshot); client.ClearScreenshots(); }
                    if (c.SharePosition.HasValue) sharePosition = c.SharePosition;
                    if (c.Rates != null && c.Rates.StateHz > 0) stateHz = c.Rates.StateHz;
                    if (c.Rates != null && c.Rates.PerceptionHz > 0) perceptionHz = c.Rates.PerceptionHz;
                    Perceive(c.Perception);
                    break;
                case Logout _:
                    Logger.LogInfo("controller asked to log out");
                    EndSession("logout");
                    loggingOut = true;
                    AutoJoinPatch.Suppress = true;
                    Game.instance?.Logout(true, true);
                    break;
                case Bye b:
                    Logger.LogInfo($"controller said bye ({b.Reason})");
                    EndSession("controller left");
                    break;
                case Protocol.Error e:
                    Logger.LogWarning($"controller error {e.Code} (re {e.Re}): {e.Detail}");
                    break;
            }
        }

        /// <summary>The perception bounds from welcome or config; zero or
        /// missing keeps what there was.</summary>
        void Perceive(PerceptionConfig c)
        {
            if (c == null) return;
            if (c.Radius > 0) perceiver.Radius = c.Radius;
            if (c.MaxPerKind > 0) perceiver.MaxPerKind = c.MaxPerKind;
        }

        void EndSession(string why)
        {
            cameraReporter.Configure(null);
            client.ClearScreenshots();
            sharePosition = null;
            runner.EndSession(why);
            WorldNavigation.Reset();
            Driver.Active = false;
            Driver.Stand();
            Driver.LookAt = null;
        }

        /// <summary>Called by the reporter when the bot dies.</summary>
        internal void BotDied() => runner.Died();

        internal void Send(Message m) => client.Send(m);

        internal string Phase()
        {
            var p = Player.m_localPlayer;
            return SessionPhase.Resolve(loggingOut, p != null, ZNet.instance != null, p != null && p.IsDead());
        }

        Hello BuildHello() => new Hello
        {
            Token = token.Value,
            Plugin = Version,
            Game = global::Version.GetVersionString(),
            Phase = Phase(),
            Character = Game.instance?.GetPlayerProfile()?.GetName() ?? "",
            World = ZNet.instance != null ? ZNet.instance.GetWorldName() ?? "" : "",
            Capabilities = runner.Capabilities.ToList(),
        };

        void OnDestroy()
        {
            harmony?.UnpatchSelf();
            client?.Dispose();
        }
    }

    sealed class BepLog : ILog
    {
        readonly BepInEx.Logging.ManualLogSource log;
        public BepLog(BepInEx.Logging.ManualLogSource log) => this.log = log;
        public void Info(string message) => log.LogInfo(message);
        public void Warn(string message) => log.LogWarning(message);
        public void Debug(string message) => log.LogDebug(message);
    }
}
