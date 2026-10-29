using System.Collections.Generic;
using Renki.Movement;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>
    /// Doors for the walker: opens a closed one ahead, or beside the bot
    /// when it is stuck, and shuts the ones it opened once it is past them.
    /// It uses Door.Interact, as a player does, so wards and keys still
    /// decide. The rules on distance and direction are DoorRules.
    /// </summary>
    sealed class Doorman
    {
        /// <summary>Safety.AllowDoors; off, the walker leaves doors alone.</summary>
        internal static bool Enabled = true;

        /// <summary>How long a door that refused is left alone.</summary>
        const float RefusedFor = 10f;

        /// <summary>How long a door the bot just opened or shut is left
        /// alone. Interact asks the door's owner to toggle it, and its state
        /// takes a moment to come back: asking again before then toggles it
        /// a second time.</summary>
        const float SettleFor = 1.5f;

        readonly List<Piece> near = new List<Piece>();
        readonly List<Door> opened = new List<Door>();
        readonly Dictionary<Door, float> leave = new Dictionary<Door, float>();
        float nextLook;

        /// <summary>Each step while walking: shuts doors left behind, and a few
        /// times a second opens a closed door ahead.</summary>
        public void Walk(Player me, Vector3 pos, Vector3 heading)
        {
            if (!Enabled || me == null) return;
            ShutBehind(me, pos);
            if (Time.time < nextLook) return;
            nextLook = Time.time + 0.25f;
            var at = Convert.ToV3(pos);
            var dir = Convert.ToV3(heading);
            Open(me, pos, (float)DoorRules.Reach, d => DoorRules.Ahead(at, dir, Convert.ToV3(d)), "ahead");
        }

        /// <summary>When stuck: opens the nearest closed door in any direction.
        /// True when it opened one, which is the recovery for this round.</summary>
        public bool Stuck(Player me, Vector3 pos) =>
            Enabled && me != null && Open(me, pos, (float)DoorRules.StuckReach, _ => true, "stuck");

        static bool IsOpen(Door d)
        {
            var zdo = d.m_nview != null ? d.m_nview.GetZDO() : null;
            return zdo != null && zdo.GetInt(ZDOVars.s_state, 0) != 0;
        }

        bool Open(Player me, Vector3 pos, float radius, System.Func<Vector3, bool> wanted, string why)
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
                if (!wanted(at)) continue;
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
            // The door itself turns the bot away for a ward or a missing key.
            if (best.Interact(me, false, false))
            {
                if (!opened.Contains(best)) opened.Add(best);
                leave[best] = Time.time + SettleFor;
                Tracer.Emit("door", "opening", data);
                return true;
            }
            leave[best] = Time.time + RefusedFor;
            Tracer.Emit("door", "will not open", data);
            return false;
        }

        void ShutBehind(Player me, Vector3 pos)
        {
            for (var i = opened.Count - 1; i >= 0; i--)
            {
                var door = opened[i];
                if (door == null)
                {
                    opened.RemoveAt(i);
                    continue;
                }
                if (!DoorRules.Past(Convert.ToV3(pos), Convert.ToV3(door.transform.position))) continue;
                if (leave.TryGetValue(door, out var settling) && Time.time < settling) continue;
                opened.RemoveAt(i);
                if (!IsOpen(door) || door.m_canNotBeClosed) continue;
                door.Interact(me, false, false);
                leave[door] = Time.time + SettleFor;
                Tracer.Emit("door", "shutting behind", new Dictionary<string, object>
                {
                    ["door"] = Convert.ToArray(door.transform.position),
                    ["pos"] = Convert.ToArray(pos),
                });
            }
        }
    }
}
