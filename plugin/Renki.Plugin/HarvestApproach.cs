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

        internal static bool Contact(Collider[] colliders, Vector3 resource, Vector3 from, float height, out Vector3 contact)
        {
            contact = default;
            var best = float.PositiveInfinity;
            var eye = new Vector3(from.x, resource.y + height, from.z);
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
            return !float.IsPositiveInfinity(best);
        }

        internal static bool GroundGoal(GameObject target, Vector3 resource, Vector3 contact, Vector3 from, float standOff, out Vector3 goal)
        {
            var desired = HarvestReach.StandAt(Convert.ToV3(resource), Convert.ToV3(contact), Convert.ToV3(from), standOff);
            goal = Convert.ToVector3(desired);
            // Anchor the scan at the resource's level, never at the starting
            // player floor. Ignore the resource so its top cannot be support.
            var top = new Vector3(goal.x, resource.y + 2, goal.z);
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
