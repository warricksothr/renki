using System.Collections.Generic;
using System.Linq;
using Renki.Protocol;
using Renki.Tasks;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>Station proximity is not interaction reach. Measure alternate
    /// standing floors and sight, then let ordinary safe travel reach them.</summary>
    sealed class CookingApproach
    {
        static readonly int GroundMask = LayerMask.GetMask("Default", "Default_small", "static_solid", "piece", "terrain");
        readonly RaycastHit[] supports = new RaycastHit[64];
        readonly Collider[] bodies = new Collider[64];
        readonly CookingStation station;
        readonly CookingFire fire;
        readonly string id;
        MoveToTask walk;
        float began = -1, walkAt, angle;
        int seed, walks, measured, bodyRejected, reachRejected;

        internal CookingApproach(string id, CookingStation station, CookingFire fire)
        { this.id = id; this.station = station; this.fire = fire; }

        bool Ready(Player me, Vector3 feet)
        {
            var reach = CookingTask.InteractionRange(me);
            var fuel = fire.PreparationTarget();
            return UseReach.From(me, station.gameObject, reach, feet) &&
                (fuel == null || UseReach.From(me, fuel, reach, feet)) && fire.SafeFooting(feet) &&
                EffectArea.IsPointInsideArea(feet + Vector3.up * .5f, EffectArea.Type.Burning, .4f) == null;
        }

        Outcome Failed(string reason)
        {
            Tracer.Emit("cooking", "interaction approaches exhausted", new Dictionary<string, object>
            { ["id"] = id, ["seeds"] = seed, ["walks"] = walks, ["measured"] = measured, ["bodyRejected"] = bodyRejected, ["reachRejected"] = reachRejected });
            return Outcome.Failed(Wire.ReasonUnreachable, reason);
        }

        internal Outcome Tick(double dt, out bool waiting)
        {
            waiting = true;
            var me = Player.m_localPlayer;
            if (Ready(me, me.transform.position)) { Cancel(); waiting = false; return null; }
            if (began < 0)
            {
                began = Time.unscaledTime;
                var from = me.transform.position - station.transform.position;
                angle = Mathf.Atan2(from.z, from.x);
                Tracer.Emit("cooking", "measuring alternate interaction positions", new Dictionary<string, object>
                {
                    ["id"] = id,
                    ["range"] = CookingTask.InteractionRange(me),
                    ["pos"] = Convert.ToArray(me.transform.position),
                    ["station"] = Convert.ToArray(station.transform.position),
                    ["colliders"] = station.GetComponentsInChildren<Collider>().Take(8).Select(c => new
                    { name = c.name, trigger = c.isTrigger, enabled = c.enabled, center = Convert.ToArray(c.bounds.center), size = Convert.ToArray(c.bounds.size) }).ToArray()
                });
            }
            if (Time.unscaledTime - began > 60)
                return Failed("no usable cooking position reached within the approach deadline");
            if (walk != null)
            {
                var result = walk.Tick(dt);
                if (result == null && Time.unscaledTime - walkAt < 12) return null;
                Cancel();
                Tracer.Emit("cooking", "trying another interaction approach", new Dictionary<string, object>
                { ["id"] = id, ["attempt"] = walks, ["reason"] = result?.Detail ?? "approach work limit" });
                return null;
            }
            Driver.Stand();
            if (seed >= 48 || walks >= 4)
                return Failed("no supported, clear cooking position after bounded alternate approaches");
            // One bounded seed per physics tick; support may be on either side
            // of a raised floor. Sight and body clearance reject the wrong one.
            var index = seed++;
            var spoke = index % 12;
            var turn = spoke % 2 == 0 ? spoke / 2 : -(spoke + 1) / 2;
            var bearing = angle + turn * Mathf.PI / 6;
            var radius = 1.25f + index / 12;
            var at = station.transform.position + new Vector3(Mathf.Cos(bearing), 0, Mathf.Sin(bearing)) * radius;
            var hits = Physics.RaycastNonAlloc(at + Vector3.up * 4, Vector3.down, supports, 8, GroundMask, QueryTriggerInteraction.Ignore);
            if (hits == supports.Length) return null;
            foreach (var hit in supports.Take(hits).OrderBy(h => Mathf.Abs(h.point.y - me.transform.position.y)))
            {
                if (hit.normal.y < .65f || hit.collider.transform.IsChildOf(station.transform)) continue;
                var feet = hit.point + Vector3.up * .03f;
                if (feet.y <= WorldNavigation.WaterLevel(Convert.ToV3(feet)) + .05 || Vector3.Distance(feet, me.transform.position) > 12) continue;
                measured++;
                var blocked = Physics.OverlapCapsuleNonAlloc(feet + Vector3.up * .45f, feet + Vector3.up * 1.4f,
                    .4f, bodies, GroundMask, QueryTriggerInteraction.Ignore);
                if (blocked != 0) { bodyRejected++; continue; }
                if (!Ready(me, feet)) { reachRejected++; continue; }
                walk = new MoveToTask(new MoveToArgs { To = new Target { Pos = Convert.ToV3(feet) }, StopDistance = .15, Dry = true, AvoidHostiles = true }, feet, null);
                walkAt = Time.unscaledTime; walks++;
                Tracer.Emit("cooking", "approaching verified interaction position", new Dictionary<string, object>
                { ["id"] = id, ["attempt"] = walks, ["seed"] = index, ["to"] = Convert.ToArray(feet) });
                return null;
            }
            return null;
        }

        internal void Cancel() { walk?.Cancel(); walk = null; Driver.Stand(); }
    }
}
