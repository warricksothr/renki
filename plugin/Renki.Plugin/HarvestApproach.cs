using System;
using System.Linq;
using Renki.Tasks;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>Measure the resource itself, then find support beside it.
    /// Collider bounds only seed rays; they are never proof of attack reach.</summary>
    static class HarvestApproach
    {
        static readonly int GroundMask = LayerMask.GetMask("Default", "Default_small", "static_solid", "piece", "terrain");
        static readonly RaycastHit[] groundHits = new RaycastHit[64];
        // Same solid/character layers as the game's terrain-aware attack mask.
        static readonly int StrikeMask = LayerMask.GetMask("Default", "static_solid", "Default_small", "piece", "piece_nonsolid",
            "terrain", "character", "character_net", "character_ghost", "hitbox", "character_noenv", "vehicle");
        static readonly RaycastHit[] strikeHits = new RaycastHit[64];

        internal static bool ClearStrike(Player player, GameObject target, Vector3 contact, float height, float reach)
        {
            var feet = player.transform.position;
            if (!HarvestReach.InRange(Convert.ToV3(feet), Convert.ToV3(contact), height, reach)) return false;
            var origin = feet + Vector3.up * height;
            var count = Physics.RaycastNonAlloc(origin, (contact - origin).normalized, strikeHits, reach + .05f,
                StrikeMask, QueryTriggerInteraction.Ignore);
            if (count == strikeHits.Length) return false;
            return HarvestReach.CanStrike(Convert.ToV3(feet), Convert.ToV3(contact), height, reach,
                strikeHits.Take(count).Select(hit => new SightHit(hit.distance,
                    hit.collider.transform.IsChildOf(player.transform), hit.collider.transform.IsChildOf(target.transform))));
        }

        internal static bool Contact(Collider[] colliders, Vector3 resource, float baseHeight, Vector3 from, float height, bool nearest, out Vector3 contact, out string method)
        {
            contact = default;
            method = "horizontal_ray";
            var best = float.PositiveInfinity;
            var eye = new Vector3(from.x, baseHeight + height, from.z);
            if (nearest)
            {
                foreach (var c in colliders)
                {
                    if (c == null || !c.enabled || c.isTrigger || !c.gameObject.activeInHierarchy) continue;
                    // Unity cannot query the closest point on nonconvex mesh
                    // colliders. Do not use their loose bounds as a substitute.
                    if (!(c is BoxCollider || c is SphereCollider || c is CapsuleCollider || c is MeshCollider mesh && mesh.convex)) continue;
                    var point = c.ClosestPoint(eye);
                    var distance = (point - eye).sqrMagnitude;
                    if (distance >= best) continue;
                    best = distance; contact = point;
                }
                if (!float.IsPositiveInfinity(best)) { method = "closest_surface"; return true; }
            }
            // A full tree's canopy can move its mesh bounds centre away from
            // the narrow trunk at axe height. Probe through the resource pivot
            // first; bounds remain ray seeds, never evidence of contact.
            foreach (var c in colliders)
            {
                if (c == null || !c.enabled || c.isTrigger || !c.gameObject.activeInHierarchy) continue;
                var aim = new Vector3(resource.x, eye.y, resource.z);
                var direction = aim - eye;
                if (direction.sqrMagnitude < .0001f) direction = Vector3.forward;
                direction.Normalize();
                var span = Vector3.Distance(aim, c.bounds.center) + c.bounds.extents.magnitude + 2;
                if (!c.Raycast(new Ray(aim - direction * span, direction), out var hit, span * 2)) continue;
                var distance = (hit.point - eye).sqrMagnitude;
                if (distance >= best) continue;
                best = distance; contact = hit.point;
            }
            if (!float.IsPositiveInfinity(best)) { method = "pivot_ray"; return true; }
            foreach (var c in colliders)
            {
                if (c == null || !c.enabled || c.isTrigger || !c.gameObject.activeInHierarchy) continue;
                var bounds = c.bounds;
                var centre = bounds.center;
                centre.y = Mathf.Clamp(eye.y, bounds.min.y + .01f, bounds.max.y - .01f);
                var direction = centre - eye; direction.y = 0;
                if (direction.sqrMagnitude < .0001f) direction = Vector3.forward;
                direction.Normalize();
                var span = bounds.extents.magnitude + 2;
                // Start outside even when the bot is inside a loose mesh AABB.
                if (!c.Raycast(new Ray(centre - direction * span, direction), out var hit, span * 2)) continue;
                var distance = (hit.point - eye).sqrMagnitude;
                if (distance >= best) continue;
                best = distance; contact = hit.point;
            }
            if (!float.IsPositiveInfinity(best)) return true;
            // A tilted log can have an empty horizontal slice near its pivot
            // height. Probe through its physical centre in 3D as a fallback;
            // only Collider.Raycast contact may establish a strike point.
            eye = from + Vector3.up * height;
            method = "center_ray";
            foreach (var c in colliders)
            {
                if (c == null || !c.enabled || c.isTrigger || !c.gameObject.activeInHierarchy) continue;
                var bounds = c.bounds;
                var direction = (bounds.center - eye).normalized;
                if (direction.sqrMagnitude < .0001f) direction = Vector3.forward;
                var span = bounds.extents.magnitude + 2;
                if (!c.Raycast(new Ray(bounds.center - direction * span, direction), out var hit, span * 2)) continue;
                var distance = (hit.point - eye).sqrMagnitude;
                if (distance >= best) continue;
                best = distance; contact = hit.point;
            }
            return !float.IsPositiveInfinity(best);
        }

        internal static bool GroundGoal(GameObject target, Vector3 resource, Vector3 contact, Vector3 from, float standOff, out Vector3 goal)
        {
            var desired = HarvestReach.StandAt(Convert.ToV3(resource), Convert.ToV3(contact), Convert.ToV3(from), standOff);
            goal = Convert.ToVector3(desired);
            // Anchor at the pivot or a lower physical contact, never the
            // starting floor. Ignore the resource so its top is not support.
            var top = new Vector3(goal.x, (float)HarvestReach.GroundProbeTop(resource.y, contact.y), goal.z);
            var count = Physics.RaycastNonAlloc(top, Vector3.down, groundHits, 8, GroundMask, QueryTriggerInteraction.Ignore);
            if (count == groundHits.Length) return false;
            RaycastHit? best = null;
            for (var i = 0; i < count; i++)
            {
                var hit = groundHits[i];
                if (hit.normal.y < .35f || hit.collider.transform.IsChildOf(target.transform)) continue;
                if (!best.HasValue || hit.point.y > best.Value.point.y) best = hit;
            }
            if (!best.HasValue) return false;
            goal = best.Value.point;
            var water = ZoneSystem.instance != null ? ZoneSystem.instance.m_waterLevel : -10000;
            return goal.y >= water - .5f;
        }
    }
}
