using System;
using System.Collections.Generic;
using System.Linq;
using Renki.Protocol;
using Renki.Tasks;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>Target proximity is not interaction reach. Measure alternate
    /// standing floors and sight, then let ordinary safe travel reach them.</summary>
    sealed class InteractionApproach
    {
        static readonly int GroundMask = LayerMask.GetMask("Default", "Default_small", "static_solid", "piece", "terrain");
        readonly RaycastHit[] supports = new RaycastHit[64];
        readonly Collider[] bodies = new Collider[64];
        readonly GameObject target;
        readonly Func<Player, Vector3, bool> ready;
        readonly string source, label;
        readonly string id;
        MoveToTask walk;
        float began = -1, walkAt, angle;
        int seed, walks, measured, bodyRejected, reachRejected;

        internal InteractionApproach(string source, string label, string id, GameObject target, Func<Player, Vector3, bool> ready)
        { this.source = source; this.label = label; this.id = id; this.target = target; this.ready = ready; }

        Outcome Failed(string reason)
        {
            Tracer.Emit(source, "interaction approaches exhausted", new Dictionary<string, object>
            { ["id"] = id, ["seeds"] = seed, ["walks"] = walks, ["measured"] = measured, ["bodyRejected"] = bodyRejected, ["reachRejected"] = reachRejected });
            return Outcome.Failed(Wire.ReasonUnreachable, reason);
        }

        internal Outcome Tick(double dt, out bool waiting)
        {
            waiting = true;
            var me = Player.m_localPlayer;
            if (ready(me, me.transform.position)) { Cancel(); waiting = false; return null; }
            if (began < 0)
            {
                began = Time.unscaledTime;
                var from = me.transform.position - target.transform.position;
                angle = Mathf.Atan2(from.z, from.x);
                Tracer.Emit(source, "measuring alternate interaction positions", new Dictionary<string, object>
                {
                    ["id"] = id,
                    ["pos"] = Convert.ToArray(me.transform.position),
                    ["target"] = Convert.ToArray(target.transform.position),
                    ["colliders"] = target.GetComponentsInChildren<Collider>().Take(8).Select(c => new
                    { name = c.name, trigger = c.isTrigger, enabled = c.enabled, center = Convert.ToArray(c.bounds.center), size = Convert.ToArray(c.bounds.size) }).ToArray()
                });
            }
            if (Time.unscaledTime - began > 60)
                return Failed("no usable " + label + " position reached within the approach deadline");
            if (walk != null)
            {
                var result = walk.Tick(dt);
                if (result == null && Time.unscaledTime - walkAt < 12) return null;
                Cancel();
                Tracer.Emit(source, "trying another interaction approach", new Dictionary<string, object>
                { ["id"] = id, ["attempt"] = walks, ["reason"] = result?.Detail ?? "approach work limit" });
                return null;
            }
            Driver.Stand();
            if (seed >= InteractionApproaches.Limit || walks >= 4)
                return Failed("no supported, clear " + label + " position after bounded alternate approaches");
            // One bounded seed per physics tick; support may be on either side
            // of a raised floor. Sight and body clearance reject the wrong one.
            var index = seed++;
            var at = target.transform.position + Convert.ToVector3(InteractionApproaches.Offset(index, angle));
            var hits = Physics.RaycastNonAlloc(at + Vector3.up * 4, Vector3.down, supports, 8, GroundMask, QueryTriggerInteraction.Ignore);
            if (hits == supports.Length) return null;
            foreach (var hit in supports.Take(hits).OrderBy(h => Mathf.Abs(h.point.y - me.transform.position.y)))
            {
                if (hit.normal.y < .65f || hit.collider.transform.IsChildOf(target.transform)) continue;
                var feet = hit.point + Vector3.up * .03f;
                if (feet.y <= WorldNavigation.WaterLevel(Convert.ToV3(feet)) + .05 || Vector3.Distance(feet, me.transform.position) > 12) continue;
                measured++;
                var blocked = Physics.OverlapCapsuleNonAlloc(feet + Vector3.up * .45f, feet + Vector3.up * 1.4f,
                    .4f, bodies, GroundMask, QueryTriggerInteraction.Ignore);
                if (blocked != 0) { bodyRejected++; continue; }
                if (!ready(me, feet)) { reachRejected++; continue; }
                walk = new MoveToTask(new MoveToArgs { To = new Target { Pos = Convert.ToV3(feet) }, StopDistance = .15, Dry = true, AvoidHostiles = true }, feet, null);
                walkAt = Time.unscaledTime; walks++;
                Tracer.Emit(source, "approaching verified interaction position", new Dictionary<string, object>
                { ["id"] = id, ["attempt"] = walks, ["seed"] = index, ["to"] = Convert.ToArray(feet) });
                return null;
            }
            return null;
        }

        internal void Cancel() { walk?.Cancel(); walk = null; Driver.Stand(); }
    }
}
