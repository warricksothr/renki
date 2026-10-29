using System;
using Newtonsoft.Json.Linq;

namespace Renki.Protocol
{
    /// <summary>The task names this plugin knows (7.2).</summary>
    public static class TaskNames
    {
        public const string Stop = "stop", MoveTo = "move_to", Follow = "follow", LookAt = "look_at",
            Say = "say", Emote = "emote", Pickup = "pickup", Drop = "drop", Interact = "interact",
            Flee = "flee";

        public static readonly string[] M1 = { Stop, MoveTo, Follow, LookAt, Say, Emote };

        /// <summary>Every task the plugin carries out, M1 and what M2 has added
        /// so far. Safety still decides which of them it offers.</summary>
        public static readonly string[] Supported = { Stop, MoveTo, Follow, LookAt, Say, Emote, Pickup, Drop, Interact, Flee };

        /// <summary>Instant tasks run alongside the body task (7.1).</summary>
        public static bool IsInstant(string task) =>
            task == LookAt || task == Say || task == Emote || task == Drop || task == "equip" ||
            task == "unequip" || task == "use_item";
    }

    /// <summary>Missing or malformed arguments; answered with bad_args.</summary>
    public sealed class BadArgsException : Exception
    {
        public BadArgsException(string message) : base(message) { }
    }

    /// <summary>A place to go or look at: a world position, or an entity id.</summary>
    public sealed class Target
    {
        public V3? Pos;
        public string Entity;

        public static Target Parse(JToken token, string field)
        {
            if (token == null || token.Type == JTokenType.Null)
                throw new BadArgsException($"`{field}` is required");
            if (token.Type == JTokenType.String && ((string)token).Length > 0)
                return new Target { Entity = (string)token };
            if (token is JArray a && a.Count == 3 && IsNumber(a[0]) && IsNumber(a[1]) && IsNumber(a[2]))
                return new Target { Pos = new V3((double)a[0], (double)a[1], (double)a[2]) };
            throw new BadArgsException($"`{field}` must be an [x,y,z] position or an entity id");
        }

        static bool IsNumber(JToken t) => t.Type == JTokenType.Integer || t.Type == JTokenType.Float;
    }

    public sealed class MoveToArgs
    {
        public Target To;
        public double StopDistance = 1.5;
        public bool Run;
    }

    /// <summary>follow's <c>run</c>: false walks, true runs, "auto" (the
    /// default) matches the leader.</summary>
    public enum RunMode { Auto, Walk, Run }

    public sealed class FollowArgs
    {
        public string Player;
        public double Distance = 3.5;
        public RunMode Run = RunMode.Auto;
    }

    public sealed class LookAtArgs
    {
        public Target To;
    }

    public sealed class SayArgs
    {
        public string Text;
        public string Channel = Wire.ChannelNormal;
        public string To;
    }

    public sealed class EmoteArgs
    {
        public string Name;
    }

    public sealed class PickupArgs
    {
        /// <summary>The drop, by entity id from perception.</summary>
        public string Entity;
    }

    public sealed class InteractArgs
    {
        /// <summary>What to use, by entity id from perception.</summary>
        public string Entity;

        /// <summary>The alternative use, as with the modifier key held.</summary>
        public bool Alt;

        /// <summary>Hold the use key, as some things need.</summary>
        public bool Hold;
    }

    public sealed class FleeArgs
    {
        /// <summary>What to get away from: an entity, or a place.</summary>
        public Target From;

        /// <summary>How far from it counts as away.</summary>
        public double Distance = 20;

        /// <summary>Where safety lies, such as the owner or home; the bot
        /// runs there unless it lies past the threat. Null runs straight away.</summary>
        public Target Toward;
    }

    public sealed class DropArgs
    {
        /// <summary>The item, by prefab as the inventory names it (6.3).</summary>
        public string Prefab;
        public int Count;
    }

    public static class TaskArgs
    {
        /// <summary>The typed arguments for a task, checked as the Go side
        /// checks them. Throws BadArgsException; returns null for a task
        /// this plugin does not know.</summary>
        public static object Parse(TaskRequest req)
        {
            var a = req.Args ?? new JObject();
            switch (req.Task)
            {
                case TaskNames.Stop:
                    return new object();
                case TaskNames.MoveTo:
                    return new MoveToArgs
                    {
                        To = Target.Parse(a["to"], "to"),
                        StopDistance = Number(a, "stopDistance", 1.5),
                        Run = Bool(a, "run", false),
                    };
                case TaskNames.Follow:
                    {
                        var player = Str(a, "player");
                        if (string.IsNullOrEmpty(player)) throw new BadArgsException("`player` is required");
                        return new FollowArgs { Player = player, Distance = Number(a, "distance", 3.5), Run = Run(a["run"]) };
                    }
                case TaskNames.LookAt:
                    return new LookAtArgs { To = Target.Parse(a["to"], "to") };
                case TaskNames.Say:
                    {
                        var text = Str(a, "text");
                        if (string.IsNullOrEmpty(text)) throw new BadArgsException("`text` is required");
                        var channel = Str(a, "channel") ?? Wire.ChannelNormal;
                        var to = Str(a, "to");
                        if (channel == "") channel = Wire.ChannelNormal;
                        if (channel != Wire.ChannelNormal && channel != Wire.ChannelShout && channel != Wire.ChannelWhisper)
                            throw new BadArgsException($"unknown channel \"{channel}\"");
                        if (channel == Wire.ChannelWhisper && string.IsNullOrEmpty(to))
                            throw new BadArgsException("a whisper needs `to`");
                        return new SayArgs { Text = text, Channel = channel, To = to };
                    }
                case TaskNames.Emote:
                    {
                        var name = Str(a, "name");
                        if (string.IsNullOrEmpty(name)) throw new BadArgsException("`name` is required");
                        return new EmoteArgs { Name = name };
                    }
                case TaskNames.Pickup:
                    {
                        var entity = Str(a, "entity");
                        if (string.IsNullOrEmpty(entity)) throw new BadArgsException("`entity` is required");
                        return new PickupArgs { Entity = entity };
                    }
                case TaskNames.Interact:
                    {
                        var entity = Str(a, "entity");
                        if (string.IsNullOrEmpty(entity)) throw new BadArgsException("`entity` is required");
                        return new InteractArgs { Entity = entity, Alt = Bool(a, "alt", false), Hold = Bool(a, "hold", false) };
                    }
                case TaskNames.Flee:
                    {
                        var toward = a["toward"];
                        var distance = Number(a, "distance", 20);
                        // Nothing is ever less than nothing away.
                        if (distance <= 0) throw new BadArgsException("`distance` must be more than 0");
                        return new FleeArgs
                        {
                            From = Target.Parse(a["from"], "from"),
                            Distance = distance,
                            Toward = toward == null || toward.Type == JTokenType.Null ? null : Target.Parse(toward, "toward"),
                        };
                    }
                case TaskNames.Drop:
                    {
                        var prefab = Str(a, "prefab");
                        if (string.IsNullOrEmpty(prefab)) throw new BadArgsException("`prefab` is required");
                        var t = a["count"];
                        if (t == null || t.Type != JTokenType.Integer || (long)t < 1)
                            throw new BadArgsException("`count` must be a whole number, at least 1");
                        return new DropArgs { Prefab = prefab, Count = (int)System.Math.Min((long)t, int.MaxValue) };
                    }
            }
            return null;
        }

        static string Str(JObject a, string key)
        {
            var t = a[key];
            if (t == null || t.Type == JTokenType.Null) return null;
            if (t.Type != JTokenType.String) throw new BadArgsException($"`{key}` must be a string");
            return (string)t;
        }

        static double Number(JObject a, string key, double fallback)
        {
            var t = a[key];
            if (t == null || t.Type == JTokenType.Null) return fallback;
            if (t.Type != JTokenType.Integer && t.Type != JTokenType.Float)
                throw new BadArgsException($"`{key}` must be a number");
            var d = (double)t;
            if (double.IsNaN(d) || d < 0) throw new BadArgsException($"`{key}` must not be negative");
            return d;
        }

        static bool Bool(JObject a, string key, bool fallback)
        {
            var t = a[key];
            if (t == null || t.Type == JTokenType.Null) return fallback;
            if (t.Type != JTokenType.Boolean) throw new BadArgsException($"`{key}` must be true or false");
            return (bool)t;
        }

        static RunMode Run(JToken t)
        {
            if (t == null || t.Type == JTokenType.Null) return RunMode.Auto;
            if (t.Type == JTokenType.Boolean) return (bool)t ? RunMode.Run : RunMode.Walk;
            if (t.Type == JTokenType.String && (string)t == "auto") return RunMode.Auto;
            throw new BadArgsException("`run` must be true, false or \"auto\"");
        }
    }
}
