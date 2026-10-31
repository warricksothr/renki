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
        internal static readonly RouteMemory Routes = new RouteMemory();
        static readonly int Mask = LayerMask.GetMask("Default", "Default_small", "static_solid", "piece", "terrain");
        static readonly NavigationObstacles obstacles = new NavigationObstacles(Mask);
        static readonly RaycastHit[] supportHits = new RaycastHit[64];
        static readonly RaycastHit[] castHits = new RaycastHit[64];
        static readonly Collider[] overlaps = new Collider[64];
        static Vector3? focus;
        static string world;
        const float Step = .6f;
        const float Radius = .4f;

        internal static void Focus(Vector3 target) => focus = target;
        internal static double WaterLevel(V3 at) => Math.Max(
            ZoneSystem.instance != null ? ZoneSystem.instance.m_waterLevel : -10000,
            Floating.GetLiquidLevel(Convert.ToVector3(at)));
        internal static void Reset()
        {
            Walls = Array.Empty<Wall>(); obstacles.Clear(); Routes.Clear(); focus = null; world = null;
            Doorman.Reset();
            Array.Clear(supportHits, 0, supportHits.Length);
            Array.Clear(castHits, 0, castHits.Length);
            Array.Clear(overlaps, 0, overlaps.Length);
        }
        internal static void Update(Player p)
        {
            var current = ZNet.instance != null ? ZNet.instance.GetWorldName() : "";
            if (world != current) { Reset(); world = current; }
            var pos = p.transform.position;
            obstacles.Update(pos, focus ?? pos, Time.unscaledTime);
        }

        // Start at this floor's level; never snap onto a different storey.
        static V3? Support(V3 at, double fromY, out bool saturated)
        {
            var top = new Vector3((float)at.X, (float)fromY + Step + .05f, (float)at.Z);
            var count = Physics.RaycastNonAlloc(top, Vector3.down, supportHits, Step * 2 + .1f, Mask, QueryTriggerInteraction.Ignore);
            saturated = count == supportHits.Length;
            if (saturated) return null;
            RaycastHit? support = null;
            for (var i = 0; i < count; i++)
                if (supportHits[i].normal.y >= .35f && (!support.HasValue || supportHits[i].point.y > support.Value.point.y)) support = supportHits[i];
            return support.HasValue && Math.Abs(support.Value.point.y - fromY) <= Step ? Convert.ToV3(support.Value.point) : (V3?)null;
        }

        internal sealed class PlanningProbe
        {
            internal sealed class Failure
            {
                public string Stage, Collider;
                public Collider Solid;
                public V3 At;
                public V3? ColliderCenter;
            }
            readonly SupportMemo support;
            readonly Func<V3, V3, V3?> physical;
            readonly RouteRefusals refusals;
            readonly Func<V3, V3, bool> corridor;
            readonly bool diagnostics;
            readonly bool dry;
            internal readonly bool Executing;
            long slice;
            internal Failure LastFailure { get; private set; }
            public int BoundsRejected, PhysicsCasts, Saturated, HostileRejected;
            public int SupportHits => support.Hits;
            public int SupportQueries => support.Misses;
            public PlanningProbe(bool diagnostics = false, bool dry = false, bool executing = false, RouteRefusals refusals = null, Func<V3, V3, bool> corridor = null)
            {
                physical = (from, desired) => WorldNavigation.Probe(from, desired, this);
                this.refusals = refusals; this.corridor = corridor;
                this.diagnostics = diagnostics;
                this.dry = dry;
                Executing = executing;
                support = new SupportMemo((at, y) =>
                {
                    var result = Support(at, y, out var full);
                    if (full) Saturated++;
                    return result;
                });
            }
            // Support (including missing support) can change independently
            // of indexed solids. Share samples only within one planning
            // slice; resample floors and moving platforms next physics step.
            public void BeginSlice() { slice++; LastFailure = null; }
            public V3? Probe(V3 from, V3 desired)
            {
                LastFailure = null;
                return dry ? DryTravel.Probe(from, desired, physical) : physical(from, desired);
            }
            internal V3? Refuse(string stage, V3 at, Collider collider = null)
            {
                if (!diagnostics) return null;
                LastFailure = new Failure
                {
                    Stage = stage,
                    Solid = collider,
                    At = at,
                    Collider = collider != null ? collider.transform.root.name + "/" + collider.name : null,
                    ColliderCenter = collider != null ? Convert.ToV3(collider.bounds.center) : (V3?)null
                };
                return null;
            }
            internal bool HostileClear(V3 from, V3 to)
            {
                if (corridor == null || corridor(from, to)) return true;
                HostileRejected++; return false;
            }
            internal bool RecentlyBlocked(V3 from, V3 to) => refusals != null && refusals.Blocks(from, to, Time.unscaledTime);
            internal V3? Sample(V3 at, double y) => support.Sample(at, y, Time.unscaledTime, slice);
            internal bool WaterSafe(V3 at, double water) => dry
                ? DryTravel.Clear(at, WaterLevel(at), DryTravel.TravelMargin) : at.Y >= water - .5;
        }

        static V3? Probe(V3 from, V3 desired, PlanningProbe plan)
        {
            var supported = plan.Sample(desired, from.Y);
            var middle = plan.Sample((from + desired) * .5, from.Y);
            if (!supported.HasValue) return plan.Refuse("endpoint_support", desired);
            if (!middle.HasValue) return plan.Refuse("midpoint_support", (from + desired) * .5);
            var dest = supported.Value;
            if (!plan.HostileClear(from, dest)) return plan.Refuse("hostile_corridor", dest);
            if (plan.RecentlyBlocked(from, dest)) return plan.Refuse("recent_live_refusal", dest);
            var at = Convert.ToVector3(dest);
            var water = ZoneSystem.instance != null ? ZoneSystem.instance.m_waterLevel : -10000;
            if (!plan.WaterSafe(dest, water) || !plan.WaterSafe(middle.Value, water)) return plan.Refuse("water", dest);
            if (!WallRoute.Clear(from, dest, Walls)) return plan.Refuse("wall_memory", dest);
            if (obstacles.Blocked(from, dest)) { plan.BoundsRejected++; return plan.Refuse("obstacle_bounds", dest); }
            var origin = Convert.ToVector3(from);
            var delta = at - origin;
            if (delta.sqrMagnitude < .0001f) return dest;
            // Clear the body's upper capsule. The support probe bounds the
            // permitted step; the lower part must be able to climb stairs.
            bool Obstacle(Collider c)
            {
                if (Doorman.MayIgnoreCollider(c.GetComponentInParent<Door>(), plan.Executing)) return false;
                return true;
            }
            plan.PhysicsCasts++;
            var casts = Physics.CapsuleCastNonAlloc(origin + Vector3.up * 1.05f, origin + Vector3.up * 1.4f,
                Radius, delta.normalized, castHits, delta.magnitude, Mask, QueryTriggerInteraction.Ignore);
            if (casts == castHits.Length) { plan.Saturated++; return plan.Refuse("cast_saturated", dest); }
            for (var i = 0; i < casts; i++) if (Obstacle(castHits[i].collider)) return plan.Refuse("body_cast", dest, castHits[i].collider);
            plan.PhysicsCasts++;
            var hits = Physics.OverlapCapsuleNonAlloc(at + Vector3.up * 1.05f, at + Vector3.up * 1.4f,
                Radius, overlaps, Mask, QueryTriggerInteraction.Ignore);
            if (hits == overlaps.Length) { plan.Saturated++; return plan.Refuse("overlap_saturated", dest); }
            for (var i = 0; i < hits; i++) if (Obstacle(overlaps[i])) return plan.Refuse("body_overlap", dest, overlaps[i]);
            return dest;
        }
    }
}
