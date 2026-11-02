using System;
using Newtonsoft.Json.Linq;

namespace Renki.Protocol
{
    /// <summary>The task names this plugin knows (7.2).</summary>
    public static class TaskNames
    {
        public const string Stop = "stop", MoveTo = "move_to", Follow = "follow", LookAt = "look_at",
            Say = "say", Emote = "emote", Pickup = "pickup", Drop = "drop", Interact = "interact",
            Flee = "flee", Equip = "equip", Unequip = "unequip", Deposit = "deposit", Withdraw = "withdraw";

        /// <summary>The slots unequip empties: "hands" is both.</summary>
        public static readonly string[] Slots =
            { "right", "left", "hands", "head", "chest", "legs", "shoulder", "utility", "trinket", "ammo" };

        public static readonly string[] M1 = { Stop, MoveTo, Follow, LookAt, Say, Emote };

        /// <summary>Every task the plugin carries out, M1 and what M2 has added
        /// so far. Safety still decides which of them it offers.</summary>
        public const string Harvest = "harvest", UseItem = "use_item", Attack = "attack";
        public static readonly string[] Supported = { Stop, MoveTo, Follow, LookAt, Say, Emote, Pickup, Drop, Interact, Flee, Equip, Unequip, Deposit, Withdraw, Harvest, UseItem, Attack };

        /// <summary>Instant tasks run alongside the body task (7.1).</summary>
        public static bool IsInstant(string task) =>
            task == LookAt || task == Say || task == Emote || task == Drop || task == Equip ||
            task == Unequip || task == UseItem;
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
        public bool Dry;
        public bool AvoidHostiles;
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

    public sealed class AttackArgs
    {
        public string Entity;
        public string Style = "melee";
        public bool Kite;
        public bool Defensive;
        public V3? At;
        public double? SearchRadius;
        public int? SearchSector;
        public double? SearchOffset;
        public V3? SearchCenter;
        public V3? SearchAnchor;
        public double? SearchLimit;
    }

    public sealed class PickupArgs
    {
        /// <summary>The drop, by entity id from perception.</summary>
        public string Entity;
    }

    public sealed class HarvestArgs
    {
        public string Entity;
        public bool Collect = true;
    }

    public sealed class InteractArgs
    {
        /// <summary>What to use, by entity id from perception.</summary>
        public string Entity;

        /// <summary>The alternative use, as with the modifier key held.</summary>
        public bool Alt;

        /// <summary>Hold the use key, as some things need.</summary>
        public bool Hold;

        /// <summary>Beds only: claim/set spawn if needed, then lie down.</summary>
        public bool Sleep;

        /// <summary>Stations only: repair eligible carried gear through the game UI.</summary>
        public bool Repair;
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

    public sealed class EquipArgs
    {
        /// <summary>The item, by prefab as the inventory names it (6.3).</summary>
        public string Item;
    }

    public sealed class UseItemArgs
    {
        public string Item;
    }

    public sealed class UnequipArgs
    {
        /// <summary>One of TaskNames.Slots.</summary>
        public string Slot;
    }

    public sealed class DropArgs
    {
        /// <summary>The item, by prefab as the inventory names it (6.3).</summary>
        public string Prefab;
        public int Count;
    }

    public sealed class TransferItem
    {
        public string Prefab;
        /// <summary>-1 means all; otherwise a positive count.</summary>
        public int Count;
    }

    public sealed class TransferArgs
    {
        public string Container;
        public TransferItem[] Items;
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
                case TaskNames.Deposit:
                case TaskNames.Withdraw:
                    {
                        var container = Str(a, "container");
                        if (string.IsNullOrEmpty(container)) throw new BadArgsException("`container` is required");
                        if (!(a["items"] is JArray items) || items.Count == 0 || items.Count > 256)
                            throw new BadArgsException("1–256 items are required");
                        var seen = new System.Collections.Generic.HashSet<string>();
                        var parsed = new System.Collections.Generic.List<TransferItem>();
                        foreach (var entry in items)
                        {
                            if (!(entry is JObject obj)) throw new BadArgsException("each item must be an object");
                            var prefab = Str(obj, "prefab");
                            if (string.IsNullOrEmpty(prefab) || !seen.Add(prefab)) throw new BadArgsException("unique item prefabs are required");
                            var token = obj["count"];
                            int count;
                            if (token?.Type == JTokenType.String && (string)token == "all") count = -1;
                            else if (token?.Type == JTokenType.Integer && (long)token > 0 && (long)token <= int.MaxValue) count = (int)token;
                            else throw new BadArgsException("count must be a positive integer or all");
                            parsed.Add(new TransferItem { Prefab = prefab, Count = count });
                        }
                        return new TransferArgs { Container = container, Items = parsed.ToArray() };
                    }
                case TaskNames.MoveTo:
                    if (Bool(a, "avoidHostiles", false) && !Bool(a, "dry", false))
                        throw new BadArgsException("avoidHostiles requires dry travel");
                    return new MoveToArgs
                    {
                        To = Target.Parse(a["to"], "to"),
                        StopDistance = Number(a, "stopDistance", 1.5),
                        Run = Bool(a, "run", false),
                        Dry = Bool(a, "dry", false),
                        AvoidHostiles = Bool(a, "avoidHostiles", false),
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
                case TaskNames.Attack:
                    {
                        var entity = Str(a, "entity");
                        var kite = Bool(a, "kite", false);
                        var defensive = Bool(a, "defensive", false);
                        if (defensive && (kite || string.IsNullOrWhiteSpace(entity))) throw new BadArgsException("defensive attack requires an explicit entity without kite discovery");
                        double? searchRadius = a["searchRadius"] == null ? (double?)null : Number(a, "searchRadius", 60);
                        if (searchRadius.HasValue && (!kite || double.IsNaN(searchRadius.Value) || double.IsInfinity(searchRadius.Value) || searchRadius < 12 || searchRadius > 80))
                            throw new BadArgsException("`searchRadius` requires kite=true and a radius between 12 and 80 m");
                        if (string.IsNullOrWhiteSpace(entity) && !kite) throw new BadArgsException("`entity` is required");
                        V3? at = null;
                        if (a["at"] != null)
                        {
                            if (!kite) throw new BadArgsException("`at` requires kite=true");
                            at = Target.Parse(a["at"], "at").Pos;
                            if (!at.HasValue) throw new BadArgsException("`at` requires an [x,y,z] position");
                            foreach (var coordinate in new[] { at.Value.X, at.Value.Y, at.Value.Z })
                                if (double.IsNaN(coordinate) || double.IsInfinity(coordinate)) throw new BadArgsException("`at` requires finite coordinates");
                        }
                        var style = Str(a, "style") ?? "melee";
                        int? sector = null;
                        if (a["searchSector"] != null)
                        {
                            var value = a["searchSector"];
                            if (value.Type != JTokenType.Integer || value.Value<double>() < 0 || value.Value<double>() > 7 ||
                                !kite || !string.IsNullOrWhiteSpace(entity) || at.HasValue)
                                throw new BadArgsException("`searchSector` requires untargeted kite discovery and an integer from 0 to 7");
                            sector = value.Value<int>();
                        }
                        if (at.HasValue && searchRadius > 12) throw new BadArgsException("a fixed fighting area limits `searchRadius` to 12 m");
                        double? offset = a["searchOffset"] == null || a["searchOffset"].Type == JTokenType.Null ? (double?)null : Number(a, "searchOffset", 0);
                        if (offset.HasValue && (!sector.HasValue || double.IsNaN(offset.Value) || double.IsInfinity(offset.Value) || offset < 0 || offset > 60 ||
                            (searchRadius ?? 60) + offset > 120))
                            throw new BadArgsException("`searchOffset` requires a search sector, 0–60 m offset and at most 120 m total reach");
                        if (style == "") style = "melee";
                        if (style != "melee" && style != "auto") throw new BadArgsException("this build supports melee or auto attack style");
                        V3? searchCenter = null;
                        if (a["searchCenter"] != null)
                        {
                            searchCenter = Target.Parse(a["searchCenter"], "searchCenter").Pos;
                            if (!searchCenter.HasValue || !kite || !string.IsNullOrWhiteSpace(entity) || at.HasValue || sector.HasValue || offset.HasValue)
                                throw new BadArgsException("`searchCenter` requires untargeted kite discovery without sector/offset");
                            foreach (var coordinate in new[] { searchCenter.Value.X, searchCenter.Value.Y, searchCenter.Value.Z })
                                if (double.IsNaN(coordinate) || double.IsInfinity(coordinate)) throw new BadArgsException("`searchCenter` requires finite coordinates");
                            if (searchRadius > 24) throw new BadArgsException("area discovery limits local `searchRadius` to 24 m");
                        }
                        V3? searchAnchor = null;
                        double? searchLimit = null;
                        if (a["searchAnchor"] != null || a["searchLimit"] != null)
                        {
                            if (!searchCenter.HasValue || a["searchAnchor"] == null || a["searchLimit"] == null)
                                throw new BadArgsException("regional bounds require searchCenter, searchAnchor and searchLimit");
                            searchAnchor = Target.Parse(a["searchAnchor"], "searchAnchor").Pos;
                            searchLimit = Number(a, "searchLimit", 0);
                            if (!searchAnchor.HasValue || double.IsNaN(searchLimit.Value) || double.IsInfinity(searchLimit.Value) || searchLimit <= 0 || searchLimit > 500)
                                throw new BadArgsException("regional search bounds must be finite and positive, at most 500 m");
                            foreach (var coordinate in new[] { searchAnchor.Value.X, searchAnchor.Value.Y, searchAnchor.Value.Z })
                                if (double.IsNaN(coordinate) || double.IsInfinity(coordinate)) throw new BadArgsException("searchAnchor requires finite coordinates");
                            if ((searchCenter.Value - searchAnchor.Value).Length + (searchRadius ?? 24) > searchLimit + 1e-6)
                                throw new BadArgsException("search area exceeds native search bounds");
                        }
                        return new AttackArgs { Entity = entity, Style = style, Kite = kite, Defensive = defensive, At = at, SearchRadius = searchRadius, SearchSector = sector, SearchOffset = offset, SearchCenter = searchCenter, SearchAnchor = searchAnchor, SearchLimit = searchLimit };
                    }
                case TaskNames.Harvest:
                    {
                        var entity = Str(a, "entity");
                        if (string.IsNullOrEmpty(entity)) throw new BadArgsException("`entity` is required");
                        return new HarvestArgs { Entity = entity, Collect = Bool(a, "collect", true) };
                    }
                case TaskNames.Interact:
                    {
                        var entity = Str(a, "entity");
                        if (string.IsNullOrEmpty(entity)) throw new BadArgsException("`entity` is required");
                        var args = new InteractArgs { Entity = entity, Alt = Bool(a, "alt", false), Hold = Bool(a, "hold", false), Sleep = Bool(a, "sleep", false), Repair = Bool(a, "repair", false) };
                        if (args.Sleep && (args.Alt || args.Hold)) throw new BadArgsException("`sleep` cannot be combined with `alt` or `hold`");
                        if (args.Repair && (args.Sleep || args.Alt || args.Hold)) throw new BadArgsException("`repair` cannot combine with sleep, alt or hold");
                        return args;
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
                case TaskNames.Equip:
                    {
                        var item = Str(a, "item");
                        if (string.IsNullOrEmpty(item)) throw new BadArgsException("`item` is required");
                        return new EquipArgs { Item = item };
                    }
                case TaskNames.UseItem:
                    {
                        var item = Str(a, "item");
                        if (string.IsNullOrEmpty(item)) throw new BadArgsException("`item` is required");
                        return new UseItemArgs { Item = item };
                    }
                case TaskNames.Unequip:
                    {
                        var slot = Str(a, "slot");
                        if (string.IsNullOrEmpty(slot)) throw new BadArgsException("`slot` is required");
                        if (Array.IndexOf(TaskNames.Slots, slot) < 0)
                            throw new BadArgsException($"unknown slot \"{slot}\"; want one of {string.Join(", ", TaskNames.Slots)}");
                        return new UnequipArgs { Slot = slot };
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
