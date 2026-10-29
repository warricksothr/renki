// The plugin's side of bridge protocol v1 (docs/BRIDGE-PROTOCOL.md). These
// types mirror internal/protocol in Go, and both are tested against the same
// fixtures in internal/protocol/testdata.
using System.Collections.Generic;
using Newtonsoft.Json;
using Newtonsoft.Json.Linq;

namespace Renki.Protocol
{
    /// <summary>Names and values the protocol defines.</summary>
    public static class Wire
    {
        public const int Version = 1;

        public const string Hello = "hello", Welcome = "welcome", State = "state", Perception = "perception",
            Inventory = "inventory", Container = "container", Layout = "layout", Terrain = "terrain", Chat = "chat", Event = "event", Task = "task", Ping = "ping", Pong = "pong",
            Config = "config", Logout = "logout", Bye = "bye", Error = "error", Trace = "trace";

        public const string PhaseMenu = "menu", PhaseConnecting = "connecting", PhaseInWorld = "in_world",
            PhaseDead = "dead", PhaseLoggingOut = "logging_out";

        public const string BadMessage = "bad_message";

        public const string ChannelNormal = "normal", ChannelShout = "shout", ChannelWhisper = "whisper",
            ChannelPing = "ping";

        public const string StatusAccepted = "accepted", StatusRejected = "rejected", StatusRunning = "running",
            StatusSucceeded = "succeeded", StatusFailed = "failed", StatusCancelled = "cancelled";

        public const string ReasonUnsupported = "unsupported", ReasonNotAllowed = "not_allowed",
            ReasonBadArgs = "bad_args", ReasonInvalidTarget = "invalid_target", ReasonTargetLost = "target_lost",
            ReasonUnreachable = "unreachable", ReasonDeepWater = "deep_water", ReasonStuck = "stuck", ReasonTimeout = "timeout",
            ReasonNoTool = "no_tool", ReasonInventoryFull = "inventory_full", ReasonTooHeavy = "too_heavy",
            ReasonEncumbered = "encumbered", ReasonMissingMaterials = "missing_materials", ReasonSuperseded = "superseded",
            ReasonStopped = "stopped", ReasonDied = "died", ReasonRefused = "refused";

        public const string EventDamaged = "damaged", EventDied = "died", EventRespawned = "respawned",
            EventItemAdded = "item_added", EventSkillRaised = "skill_raised", EventPlayerJoined = "player_joined",
            EventPlayerLeft = "player_left", EventEnteredBiome = "entered_biome", EventLowHealth = "low_health";

        public const string PerceivedPlayer = "player", PerceivedCreature = "creature", PerceivedTamed = "tamed";

        public const string ResourceTree = "tree", ResourceLog = "log", ResourceRock = "rock",
            ResourcePickable = "pickable", ResourceDrop = "drop", ResourceStump = "stump";

        public const string StructureContainer = "container", StructureStation = "station", StructureBed = "bed",
            StructurePortal = "portal", StructureDoor = "door", StructureFire = "fire", StructureSmelter = "smelter",
            StructureTombstone = "tombstone", StructureOther = "other";

        public static bool IsFinal(string status) =>
            status == StatusRejected || status == StatusSucceeded || status == StatusFailed ||
            status == StatusCancelled;
    }

    /// <summary>The envelope every message carries (section 3).</summary>
    public abstract class Message
    {
        [JsonProperty("v", Order = -10)] public int V = Wire.Version;
        [JsonProperty("type", Order = -9)] public string Type;
        [JsonProperty("seq", Order = -8)] public ulong Seq;
        [JsonProperty("t", Order = -7)] public double T;
        [JsonProperty("id", Order = -6, NullValueHandling = NullValueHandling.Ignore)] public string Id;
        [JsonProperty("re", Order = -5, NullValueHandling = NullValueHandling.Ignore)] public string Re;

        /// <summary>The value of <c>type</c> for this message.</summary>
        [JsonIgnore] public abstract string MessageType { get; }
    }

    // ---- Session (section 5) ----

    public sealed class Hello : Message
    {
        public override string MessageType => Wire.Hello;
        [JsonProperty("token")] public string Token = "";
        [JsonProperty("plugin")] public string Plugin = "";
        [JsonProperty("game")] public string Game = "";
        [JsonProperty("phase")] public string Phase = "";
        [JsonProperty("character")] public string Character = "";
        [JsonProperty("world")] public string World = "";
        [JsonProperty("capabilities")] public List<string> Capabilities = new List<string>();
    }

    public sealed class Rates
    {
        [JsonProperty("stateHz")] public double StateHz;
        [JsonProperty("perceptionHz")] public double PerceptionHz;
    }

    public sealed class PerceptionConfig
    {
        [JsonProperty("radius")] public double Radius;
        [JsonProperty("maxPerKind")] public int MaxPerKind;
    }

    public sealed class Welcome : Message
    {
        public override string MessageType => Wire.Welcome;
        [JsonProperty("agentName")] public string AgentName = "";
        [JsonProperty("owners")] public List<string> Owners = new List<string>();
        [JsonProperty("rates")] public Rates Rates = new Rates();
        [JsonProperty("perception")] public PerceptionConfig Perception = new PerceptionConfig();
        [JsonProperty("allowedTasks")] public List<string> AllowedTasks = new List<string>();
    }

    public sealed class Logout : Message { public override string MessageType => Wire.Logout; }

    public sealed class Bye : Message
    {
        public override string MessageType => Wire.Bye;
        [JsonProperty("reason")] public string Reason = "";
    }

    public sealed class Ping : Message { public override string MessageType => Wire.Ping; }

    public sealed class Pong : Message { public override string MessageType => Wire.Pong; }

    public sealed class Error : Message
    {
        public override string MessageType => Wire.Error;
        [JsonProperty("code")] public string Code = "";
        [JsonProperty("detail", NullValueHandling = NullValueHandling.Ignore)] public string Detail;
    }

    public sealed class Config : Message
    {
        public override string MessageType => Wire.Config;
        [JsonProperty("rates", NullValueHandling = NullValueHandling.Ignore)] public Rates Rates;
        [JsonProperty("perception", NullValueHandling = NullValueHandling.Ignore)] public PerceptionConfig Perception;
    }

    // ---- Plugin to controller (section 6) ----

    public sealed class GameTime
    {
        [JsonProperty("day")] public int Day;
        [JsonProperty("dayFraction")] public double DayFraction;
        [JsonProperty("night")] public bool Night;
    }

    public sealed class TaskRef
    {
        [JsonProperty("id")] public string Id = "";
        [JsonProperty("task")] public string Task = "";
        [JsonProperty("status")] public string Status = "";
    }

    public sealed class FoodStatus
    {
        [JsonProperty("active")] public List<ActiveFood> Active = new List<ActiveFood>();
        [JsonProperty("carried")] public List<CarriedFood> Carried = new List<CarriedFood>();
    }
    public sealed class ActiveFood
    {
        [JsonProperty("prefab")] public string Prefab;
        [JsonProperty("remainingS")] public double RemainingS;
    }
    public sealed class CarriedFood
    {
        [JsonProperty("prefab")] public string Prefab;
        [JsonProperty("health")] public double Health;
        [JsonProperty("regen")] public double Regen;
        [JsonProperty("canEat")] public bool CanEat;
    }

    public sealed class State : Message
    {
        public override string MessageType => Wire.State;
        [JsonProperty("phase")] public string Phase = "";
        [JsonProperty("character")] public string Character = "";
        [JsonProperty("world")] public string World = "";
        [JsonProperty("pos")] public double[] Pos = new double[3];
        [JsonProperty("yaw")] public double Yaw;
        [JsonProperty("vel")] public double[] Vel = new double[3];
        [JsonProperty("hp")] public double Hp;
        [JsonProperty("hpMax")] public double HpMax;
        [JsonProperty("food", NullValueHandling = NullValueHandling.Ignore)] public FoodStatus Food;
        [JsonProperty("stamina")] public double Stamina;
        [JsonProperty("staminaMax")] public double StaminaMax;
        [JsonProperty("eitr")] public double Eitr;
        [JsonProperty("eitrMax")] public double EitrMax;
        [JsonProperty("flags")] public List<string> Flags = new List<string>();
        [JsonProperty("biome")] public string Biome = "";
        [JsonProperty("env")] public string Env = "";
        [JsonProperty("time")] public GameTime Time = new GameTime();
        [JsonProperty("equipped")] public Dictionary<string, string> Equipped = new Dictionary<string, string>();
        [JsonProperty("effects")] public List<string> Effects = new List<string>();
        [JsonProperty("weight")] public double Weight;
        [JsonProperty("weightMax")] public double WeightMax;
        [JsonProperty("task", NullValueHandling = NullValueHandling.Ignore)] public TaskRef Task;
        [JsonProperty("location", NullValueHandling = NullValueHandling.Ignore)] public Location Location;
    }

    public sealed class PerceivedCharacter
    {
        [JsonProperty("id")] public string Id = "";
        [JsonProperty("kind")] public string Kind = "";
        [JsonProperty("name", NullValueHandling = NullValueHandling.Ignore)] public string Name;
        [JsonProperty("prefab", NullValueHandling = NullValueHandling.Ignore)] public string Prefab;
        [JsonProperty("pos")] public double[] Pos = new double[3];
        [JsonProperty("hpFrac")] public double HpFrac;
        [JsonProperty("stars", NullValueHandling = NullValueHandling.Ignore)] public int? Stars;
        [JsonProperty("hostile", NullValueHandling = NullValueHandling.Ignore)] public bool? Hostile;
        [JsonProperty("targetingMe", NullValueHandling = NullValueHandling.Ignore)] public bool? TargetingMe;
    }

    public sealed class PerceivedResource
    {
        [JsonProperty("id")] public string Id = "";
        [JsonProperty("kind")] public string Kind = "";
        [JsonProperty("prefab")] public string Prefab = "";
        [JsonProperty("pos")] public double[] Pos = new double[3];
        [JsonProperty("minTool", NullValueHandling = NullValueHandling.Ignore)] public int? MinTool;
        [JsonProperty("stack", NullValueHandling = NullValueHandling.Ignore)] public int? Stack;
        [JsonProperty("yield", NullValueHandling = NullValueHandling.Ignore)] public ResourceYield Yield;
    }

    public sealed class ResourceYield
    {
        [JsonProperty("prefab")] public string Prefab = "";
        [JsonProperty("count")] public int Count;
        [JsonProperty("stackMax")] public int StackMax;
        [JsonProperty("weight")] public double Weight;
    }

    public sealed class PerceivedStructure
    {
        [JsonProperty("id")] public string Id = "";
        [JsonProperty("kind")] public string Kind = "";
        [JsonProperty("prefab", NullValueHandling = NullValueHandling.Ignore)] public string Prefab;
        [JsonProperty("tag", NullValueHandling = NullValueHandling.Ignore)] public string Tag;
        [JsonProperty("pos")] public double[] Pos = new double[3];
        [JsonProperty("mine", NullValueHandling = NullValueHandling.Ignore)] public bool? Mine;
        /// <summary>Beds only; false explicitly means unclaimed.</summary>
        [JsonProperty("owned", NullValueHandling = NullValueHandling.Ignore)] public bool? Owned;
        /// <summary>Doors only.</summary>
        [JsonProperty("open", NullValueHandling = NullValueHandling.Ignore)] public bool? Open;
    }

    /// <summary>Section 6.2; streamed from M2.</summary>
    public sealed class Perception : Message
    {
        public override string MessageType => Wire.Perception;
        [JsonProperty("characters")] public List<PerceivedCharacter> Characters = new List<PerceivedCharacter>();
        [JsonProperty("resources")] public List<PerceivedResource> Resources = new List<PerceivedResource>();
        [JsonProperty("structures")] public List<PerceivedStructure> Structures = new List<PerceivedStructure>();
    }

    /// <summary>Section 6.9: the walls near the bot, sent when they change.</summary>
    public sealed class Layout : Message
    {
        public override string MessageType => Wire.Layout;
        [JsonProperty("walls")] public List<Wall> Walls = new List<Wall>();
    }

    /// <summary>One wall: the line along its foot, from A to B, and how tall it is.</summary>
    public sealed class Wall
    {
        [JsonProperty("a")] public double[] A = new double[3];
        [JsonProperty("b")] public double[] B = new double[3];
        [JsonProperty("h")] public double H;
    }

    public sealed class InventoryItem
    {
        [JsonProperty("slot")] public int[] Slot = new int[2];
        [JsonProperty("prefab")] public string Prefab = "";
        [JsonProperty("stack")] public int Stack;
        [JsonProperty("stackMax", NullValueHandling = NullValueHandling.Ignore)] public int? StackMax;
        [JsonProperty("quality", NullValueHandling = NullValueHandling.Ignore)] public int? Quality;
        [JsonProperty("durability", NullValueHandling = NullValueHandling.Ignore)] public double? Durability;
        [JsonProperty("durabilityMax", NullValueHandling = NullValueHandling.Ignore)] public double? DurabilityMax;
        [JsonProperty("equipped", NullValueHandling = NullValueHandling.Ignore)] public bool? Equipped;
        [JsonProperty("weight")] public double Weight;
        [JsonProperty("type")] public string Type = "";
        [JsonProperty("value", NullValueHandling = NullValueHandling.Ignore)] public int? Value;
    }

    /// <summary>Section 6.3; streamed from M2.</summary>
    public sealed class Inventory : Message
    {
        public override string MessageType => Wire.Inventory;
        [JsonProperty("size")] public int[] Size = new int[2];
        [JsonProperty("items")] public List<InventoryItem> Items = new List<InventoryItem>();
    }

    public sealed class ChatSender
    {
        [JsonProperty("name")] public string Name = "";
        [JsonProperty("playerId")] public string PlayerId = "";
        [JsonProperty("owner")] public bool Owner;
    }

    public sealed class Chat : Message
    {
        public override string MessageType => Wire.Chat;
        [JsonProperty("from")] public ChatSender From = new ChatSender();
        [JsonProperty("channel")] public string Channel = "";
        [JsonProperty("text")] public string Text = "";
        [JsonProperty("pos")] public double[] Pos = new double[3];
        [JsonProperty("dist")] public double Dist;
    }

    public sealed class Event : Message
    {
        public override string MessageType => Wire.Event;
        [JsonProperty("event")] public string Name = "";
        [JsonProperty("by", NullValueHandling = NullValueHandling.Ignore)] public string By;
        [JsonProperty("amount", NullValueHandling = NullValueHandling.Ignore)] public double? Amount;
        [JsonProperty("damageType", NullValueHandling = NullValueHandling.Ignore)] public string DamageType;
        [JsonProperty("pos", NullValueHandling = NullValueHandling.Ignore)] public double[] Pos;
        [JsonProperty("prefab", NullValueHandling = NullValueHandling.Ignore)] public string Prefab;
        [JsonProperty("count", NullValueHandling = NullValueHandling.Ignore)] public int? Count;
        [JsonProperty("skill", NullValueHandling = NullValueHandling.Ignore)] public string Skill;
        [JsonProperty("level", NullValueHandling = NullValueHandling.Ignore)] public int? Level;
        [JsonProperty("name", NullValueHandling = NullValueHandling.Ignore)] public string PlayerName;
        [JsonProperty("playerId", NullValueHandling = NullValueHandling.Ignore)] public string PlayerId;
        [JsonProperty("biome", NullValueHandling = NullValueHandling.Ignore)] public string Biome;
        [JsonProperty("hpFrac", NullValueHandling = NullValueHandling.Ignore)] public double? HpFrac;
    }

    /// <summary>A decision the plugin made on its own, for the controller's
    /// audit log (6.7). Nothing acts on it.</summary>
    public sealed class Trace : Message
    {
        public override string MessageType => Wire.Trace;
        [JsonProperty("task", NullValueHandling = NullValueHandling.Ignore)] public string Task;
        [JsonProperty("source")] public string Source = "";
        [JsonProperty("msg")] public string Msg = "";
        [JsonProperty("data", NullValueHandling = NullValueHandling.Ignore)] public Dictionary<string, object> Data;
    }

    /// <summary>A status report on a task, with <c>re</c> set to its id (6.6).</summary>
    public sealed class TaskStatus : Message
    {
        public override string MessageType => Wire.Task;
        [JsonProperty("status")] public string Status = "";
        [JsonProperty("reason", NullValueHandling = NullValueHandling.Ignore)] public string Reason;
        [JsonProperty("detail", NullValueHandling = NullValueHandling.Ignore)] public string Detail;
    }

    // ---- Controller to plugin (section 7) ----

    /// <summary>A task to run (7.1). Args stays raw until TaskArgs.Parse.</summary>
    public sealed class TaskRequest : Message
    {
        public override string MessageType => Wire.Task;
        [JsonProperty("task")] public string Task = "";
        [JsonProperty("args", NullValueHandling = NullValueHandling.Ignore)] public JObject Args;
        [JsonProperty("timeoutS")] public double TimeoutS;
    }
}
