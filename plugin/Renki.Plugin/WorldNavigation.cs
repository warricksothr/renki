using System;
using System.Collections.Generic;
using System.Linq;
using Renki.Movement;
using Renki.Protocol;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>Current measured walls plus local physical support/clearance.
    /// Planning never changes Valheim's global navmesh or creature pathing.</summary>
    static class WorldNavigation
    {
        internal static IReadOnlyList<Wall> Walls = Array.Empty<Wall>();
        static readonly int Mask = LayerMask.GetMask("Default", "Default_small", "static_solid", "piece", "terrain");
        const float Step = .6f;
        const float Radius = .4f;

        internal static V3? Probe(V3 from, V3 desired)
        {
            // Start at this floor's level; never snap down to a different
            // storey or up onto a roof just because x/z match.
            Vector3? Support(V3 at)
            {
                var top = new Vector3((float)at.X, (float)from.Y + Step + .05f, (float)at.Z);
                var hits = Physics.RaycastAll(top, Vector3.down, Step * 2 + .1f, Mask, QueryTriggerInteraction.Ignore);
                RaycastHit? support = null;
                foreach (var hit in hits)
                    if (hit.normal.y >= .35f && (!support.HasValue || hit.point.y > support.Value.point.y)) support = hit;
                return support.HasValue && Math.Abs(support.Value.point.y - from.Y) <= Step ? support.Value.point : (Vector3?)null;
            }
            var supported = Support(desired);
            var middle = Support((from + desired) * .5);
            if (!supported.HasValue || !middle.HasValue) return null;
            var at = supported.Value;
            var water = ZoneSystem.instance != null ? ZoneSystem.instance.m_waterLevel : -10000;
            if (at.y < water - .5f || middle.Value.y < water - .5f) return null;
            var dest = Convert.ToV3(at);
            if (!WallRoute.Clear(from, dest, Walls)) return null;
            var origin = Convert.ToVector3(from);
            var delta = at - origin;
            if (delta.sqrMagnitude < .0001f) return dest;
            // Clear the body's upper capsule. The support probe bounds the
            // permitted step; the lower part must be able to climb stairs.
            bool Obstacle(Collider c)
            {
                if (Doorman.Enabled && c.GetComponentInParent<Door>() != null &&
                    PrivateArea.CheckAccess(c.transform.position, 0f, false, false)) return false;
                return true;
            }
            foreach (var hit in Physics.CapsuleCastAll(origin + Vector3.up * 1.05f, origin + Vector3.up * 1.4f,
                         Radius, delta.normalized, delta.magnitude, Mask, QueryTriggerInteraction.Ignore))
                if (Obstacle(hit.collider)) return null;
            foreach (var c in Physics.OverlapCapsule(at + Vector3.up * 1.05f, at + Vector3.up * 1.4f,
                         Radius, Mask, QueryTriggerInteraction.Ignore))
                if (Obstacle(c)) return null;
            return dest;
        }
    }
}
