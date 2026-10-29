using System.Collections.Generic;
using Renki.Movement;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>
    /// Doors for the walker: opens a closed one ahead, or beside the bot
    /// when it is stuck, and shuts the ones it opened once it has gone
    /// through. It uses Door.Interact, as a player does, so wards and keys
    /// still decide. The rules on distance and direction are DoorRules.
    /// What the bot has opened is kept for the whole plugin, not per walk,
    /// so Tick shuts a door behind it after the task that opened it ends.
    /// </summary>
    sealed class Doorman
    {
        /// <summary>Safety.AllowDoors; off, the walker leaves doors alone.</summary>
        internal static bool Enabled = true;

        /// <summary>How long a door that refused is left alone.</summary>
        const float RefusedFor = 10f;

        /// <summary>How long a door the bot just opened or shut is left
        /// alone, and how long one it asked to open may stay shut before it
        /// counts as refused. Interact asks the door's owner to toggle it,
        /// and its state takes a moment to come back (longer on a slow
        /// link): asking again before then toggles it a second time.</summary>
        const float SettleFor = 3f;

        /// <summary>How long a door the bot shut stays out of stuck recovery,
        /// so a stuck bot does not reopen the door it just shut.</summary>
        const float ShutFor = 10f;

        /// <summary>A door the bot opened but never went through is shut
        /// anyway after this long, once the bot is clear of it, if it is
        /// still within AbandonReach; farther off it is left as it is.</summary>
        const float AbandonAfter = 20f;

        const double AbandonReach = 6;

        sealed class Opened
        {
            public Door Door;
            public bool Through, Refused;
            public float At;
        }

        static readonly List<Opened> opened = new List<Opened>();
        static readonly Dictionary<Door, float> leave = new Dictionary<Door, float>();
        static readonly Dictionary<Door, float> shut = new Dictionary<Door, float>();

        readonly List<Piece> near = new List<Piece>();
        float nextLook;

        /// <summary>A few times a second while walking: opens a closed door ahead.</summary>
        public void Walk(Player me, Vector3 pos, Vector3 heading)
        {
            if (!Enabled || me == null || Time.time < nextLook) return;
            nextLook = Time.time + 0.25f;
            var at = Convert.ToV3(pos);
            var dir = Convert.ToV3(heading);
            Open(me, pos, (float)DoorRules.Reach, (d, _) => DoorRules.Ahead(at, dir, Convert.ToV3(d)), "ahead");
        }

        /// <summary>When stuck: opens the nearest closed door in any direction,
        /// other than one it has just shut. True when it opened one, which is
        /// the recovery for this round.</summary>
        public bool Stuck(Player me, Vector3 pos) =>
            Enabled && me != null &&
            Open(me, pos, (float)DoorRules.StuckReach,
                (_, door) => !(shut.TryGetValue(door, out var until) && Time.time < until), "stuck");

        static bool IsOpen(Door d)
        {
            var zdo = d.m_nview != null ? d.m_nview.GetZDO() : null;
            return zdo != null && zdo.GetInt(ZDOVars.s_state, 0) != 0;
        }

        bool Open(Player me, Vector3 pos, float radius, System.Func<Vector3, Door, bool> wanted, string why)
        {
            near.Clear();
            Piece.GetAllPiecesInRadius(pos, radius, near);
            Door best = null;
            var bestDistance = float.MaxValue;
            foreach (var p in near)
            {
                if (!p.TryGetComponent<Door>(out var door) || IsOpen(door)) continue;
                if (leave.TryGetValue(door, out var until) && Time.time < until) continue;
                var at = door.transform.position;
                if (!wanted(at, door)) continue;
                var d = Vector3.Distance(pos, at);
                if (d < bestDistance) (best, bestDistance) = (door, d);
            }
            if (best == null) return false;

            var data = new Dictionary<string, object>
            {
                ["door"] = Convert.ToArray(best.transform.position),
                ["pos"] = Convert.ToArray(pos),
                ["why"] = why,
            };
            // Interact answers true for a ward or a missing key as well as for
            // a door it opened, so a refusal shows only as a door still shut
            // once it has settled: Tick catches that.
            if (best.Interact(me, false, false))
            {
                var o = opened.Find(x => x.Door == best);
                if (o == null) opened.Add(o = new Opened { Door = best });
                o.At = Time.time;
                o.Through = o.Refused = false;
                leave[best] = Time.time + SettleFor;
                Tracer.Emit("door", "opening", data);
                return true;
            }
            leave[best] = Time.time + RefusedFor;
            Tracer.Emit("door", "will not open", data);
            return false;
        }

        /// <summary>Every frame in a session: notes when the bot is in the
        /// doorway of a door it opened, and shuts that door once the bot is
        /// past it, whether or not a task is still running.</summary>
        public static void Tick(Player me)
        {
            if (opened.Count == 0) return;
            if (!Enabled || me == null)
            {
                opened.Clear();
                return;
            }
            var pos = Convert.ToV3(me.transform.position);
            for (var i = opened.Count - 1; i >= 0; i--)
            {
                var o = opened[i];
                if (o.Door == null)
                {
                    opened.RemoveAt(i);
                    continue;
                }
                var at = Convert.ToV3(o.Door.transform.position);
                if (!o.Through && !o.Refused && Time.time - o.At > SettleFor && !IsOpen(o.Door))
                {
                    // Asked to open and still shut: a ward, a key, or the owner
                    // shut it again. Leave it alone for a while, but keep it,
                    // in case it opens late and has to be shut behind.
                    o.Refused = true;
                    leave[o.Door] = Time.time + RefusedFor;
                    Tracer.Emit("door", "will not open", new Dictionary<string, object>
                    {
                        ["door"] = at.ToArray(),
                        ["pos"] = pos.ToArray(),
                    });
                    continue;
                }
                // In the doorway counts only with the door open: pressed
                // against a shut one is not through it.
                if (DoorRules.In(pos, at) && IsOpen(o.Door)) o.Through = true;
                if (!DoorRules.Past(pos, at)) continue;
                if (!o.Through)
                {
                    if (Time.time - o.At < AbandonAfter) continue;
                    if (V3.FlatDistance(pos, at) > AbandonReach)
                    {
                        // Too far off to reach for it, as a player would not.
                        opened.RemoveAt(i);
                        continue;
                    }
                }
                if (leave.TryGetValue(o.Door, out var settling) && Time.time < settling) continue;
                opened.RemoveAt(i);
                if (!IsOpen(o.Door) || o.Door.m_canNotBeClosed) continue;
                o.Door.Interact(me, false, false);
                leave[o.Door] = Time.time + SettleFor;
                shut[o.Door] = Time.time + ShutFor;
                Tracer.Emit("door", o.Through ? "shutting behind" : "shutting, never went through", new Dictionary<string, object>
                {
                    ["door"] = Convert.ToArray(o.Door.transform.position),
                    ["pos"] = pos.ToArray(),
                });
            }
        }
    }
}
