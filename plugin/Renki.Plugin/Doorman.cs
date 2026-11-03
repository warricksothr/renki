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
        const float SettleFor = (float)DoorPassage.SettleFor;

        /// <summary>How long a door the bot shut stays out of stuck recovery,
        /// so a stuck bot does not reopen the door it just shut.</summary>
        const float ShutFor = 10f;

        sealed class Opened
        {
            public Door Door;
            public DoorPassage Passage;
        }

        static readonly List<Opened> opened = new List<Opened>();
        static readonly Dictionary<Door, float> leave = new Dictionary<Door, float>();
        static readonly Dictionary<Door, float> shut = new Dictionary<Door, float>();
        static readonly Dictionary<Door, float> refused = new Dictionary<Door, float>();

        internal static void Reset()
        {
            opened.Clear(); leave.Clear(); shut.Clear(); refused.Clear();
        }

        internal static bool MayIgnoreCollider(Door door, bool executing)
        {
            if (door == null || door.m_nview == null || !door.m_nview.IsValid()) return false;
            var me = Player.m_localPlayer;
            var canOpen = me != null &&
                (!door.m_checkGuardStone || PrivateArea.CheckAccess(door.transform.position, 0f, false, false)) &&
                (door.m_keyItem == null || me.GetInventory().HaveItem(door.m_keyItem.m_itemData.m_shared.m_name));
            return DoorRules.MayIgnoreCollider(Enabled, !IsOpen(door), canOpen,
                refused.TryGetValue(door, out var until) && Time.time < until, executing);
        }

        // Opening is asynchronous. Hold at a physical obstruction while the
        // nearby request settles, rather than replacing the route each frame.
        internal static bool Waiting(Vector3 pos) => opened.Exists(o => o.Door != null &&
            o.Passage.Settling(Time.time) && Vector3.Distance(pos, o.Door.transform.position) <= DoorRules.StuckReach);

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

        /// <summary>Both explicit use and walker use register initially closed
        /// leaves. An already-open door remains its owner's responsibility.</summary>
        internal static bool Interact(Door door, Player me, bool hold, bool alt)
        {
            var wasOpen = IsOpen(door);
            var accepted = door.Interact(me, hold, alt);
            if (DoorPassage.TrackOpening(Enabled, wasOpen, accepted, hold))
            {
                var o = opened.Find(x => x.Door == door);
                if (o == null) opened.Add(o = new Opened { Door = door });
                o.Passage = new DoorPassage(Convert.ToV3(me.transform.position),
                    Convert.ToV3(door.transform.position), Convert.ToV3(door.transform.forward), Time.time);
                leave[door] = Time.time + SettleFor + .5f;
            }
            return accepted;
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
                if (System.Math.Abs(pos.y - at.y) > DoorRules.Level) continue;
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
            if (Interact(best, me, false, false))
            {
                // A little past SettleFor, so Tick judges a refusal before
                // stuck recovery, every 3 s, can ask again.
                leave[best] = Time.time + SettleFor + 0.5f;
                Tracer.Emit("door", "opening", data);
                return true;
            }
            leave[best] = Time.time + RefusedFor;
            refused[best] = Time.time + RefusedFor;
            Tracer.Emit("door", "will not open", data);
            return false;
        }

        /// <summary>Every frame in a session: notes when the bot is in the
        /// doorway of a door it opened, and shuts that door once the bot is
        /// past it, whether or not a task is still running. A door still
        /// shut once it has settled was refused (a ward, a key) or shut by
        /// someone else, and there is nothing to shut.</summary>
        public static void Tick(Player me)
        {
            // No player between a death and the respawn, or during a load:
            // keep the doors for when it is back.
            if (opened.Count == 0 || me == null || me.IsDead()) return;
            if (!Enabled)
            {
                opened.Clear();
                return;
            }
            var pos = Convert.ToV3(me.transform.position);
            for (var i = opened.Count - 1; i >= 0; i--)
            {
                var o = opened[i];
                if (o.Door == null || o.Door.m_nview == null || !o.Door.m_nview.IsValid())
                {
                    opened.RemoveAt(i);
                    continue;
                }
                var at = Convert.ToV3(o.Door.transform.position);
                var step = o.Passage.Observe(pos, at, Convert.ToV3(o.Door.transform.forward),
                    IsOpen(o.Door), !o.Door.m_canNotBeClosed, Time.time);
                if (step == DoorCleanup.Wait) continue;
                if (step == DoorCleanup.Close)
                {
                    o.Door.Interact(me, false, false);
                    o.Passage.Closing(Time.time);
                    leave[o.Door] = Time.time + SettleFor;
                    shut[o.Door] = Time.time + ShutFor;
                }
                else
                {
                    opened.RemoveAt(i);
                }
                if (step == DoorCleanup.OpenRefused)
                {
                    leave[o.Door] = Time.time + RefusedFor;
                    refused[o.Door] = Time.time + RefusedFor;
                }
                Tracer.Emit("door", step == DoorCleanup.Close ? "shutting behind" :
                    step == DoorCleanup.Closed ? "closed confirmed" :
                    step == DoorCleanup.OpenRefused ? "will not open" :
                    step == DoorCleanup.CloseRefused ? "will not close" : "cleanup out of reach", new Dictionary<string, object>
                    {
                        ["door"] = at.ToArray(),
                        ["pos"] = pos.ToArray(),
                    });
            }
        }
    }
}
