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
        internal static List<LogObstacle> LogObstacles() => obstacles.Snapshot();
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
        static V3? Support(V3 at, double fromY, out bool saturated, bool footing = false)
        {
            var top = new Vector3((float)at.X, (float)fromY + Step + .05f, (float)at.Z);
            var count = Physics.RaycastNonAlloc(top, Vector3.down, supportHits, Step * 2 + .1f, Mask, QueryTriggerInteraction.Ignore);
            saturated = count == supportHits.Length;
            if (saturated) return null;
            RaycastHit? support = null;
            for (var i = 0; i < count; i++)
                if (supportHits[i].normal.y >= .35f && (footing
                    ? DryTravel.PreferFooting(fromY, supportHits[i].point.y, support?.point.y)
                    : !support.HasValue || supportHits[i].point.y > support.Value.point.y)) support = supportHits[i];
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
                public double[] SupportHeights;
            }
            readonly SupportMemo support;
            readonly SupportMemo footing;
            readonly Func<V3, V3, V3?> physical;
            readonly RouteRefusals refusals;
            readonly Func<V3, V3, bool> corridor;
            readonly bool diagnostics;
            readonly bool dry;
            internal readonly bool Executing;
            long slice;
            internal Failure LastFailure { get; private set; }
            // Local search may wait for an asynchronous navmesh retry after
            // exhausting its frontier. Keep diagnostic evidence across slices.
            internal Failure LastRefusal { get; private set; }
            readonly Dictionary<string, int> failureCounts = new Dictionary<string, int>();
            readonly Dictionary<string, Failure> failureExamples = new Dictionary<string, Failure>();
            internal object Failures => new
            {
                counts = failureCounts,
                examples = failureExamples.Select(pair => new
                {
                    stage = pair.Key,
                    at = pair.Value.At.ToArray(),
                    collider = pair.Value.Collider,
                    supportHeights = pair.Value.SupportHeights
                }).ToArray()
            };
            public int BoundsRejected, PhysicsCasts, Saturated, HostileRejected;
            public int SupportHits => support.Hits + footing.Hits;
            public int SupportQueries => support.Misses + footing.Misses;
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
                footing = new SupportMemo((at, y) =>
                {
                    var result = Support(at, y, out var full, footing: true);
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
                LastRefusal = LastFailure;
                failureCounts[stage] = failureCounts.TryGetValue(stage, out var count) ? count + 1 : 1;
                failureExamples[stage] = LastFailure;
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
            internal bool SupportSafe(V3 from, V3 middle, V3 to)
            {
                if (!dry) return true;
                var start = footing.Sample(from, from.Y, Time.unscaledTime, slice);
                if (start.HasValue && DryTravel.ContinuousSupport(from, start.Value, middle, to)) return true;
                Refuse("support_step", to);
                if (LastFailure != null && start.HasValue)
                    LastFailure.SupportHeights = new[] { from.Y, start.Value.Y, middle.Y, to.Y };
                return false;
            }
        }

        static V3? Probe(V3 from, V3 desired, PlanningProbe plan)
        {
            var supported = plan.Sample(desired, from.Y);
            var middle = plan.Sample((from + desired) * .5, from.Y);
            if (!supported.HasValue) return plan.Refuse("endpoint_support", desired);
            if (!middle.HasValue) return plan.Refuse("midpoint_support", (from + desired) * .5);
            var dest = supported.Value;
            if (!plan.SupportSafe(from, middle.Value, dest)) return null;
            if (!plan.HostileClear(from, dest)) return plan.Refuse("hostile_corridor", dest);
            if (plan.RecentlyBlocked(from, dest)) return plan.Refuse("recent_live_refusal", dest);
            var at = Convert.ToVector3(dest);
            var water = ZoneSystem.instance != null ? ZoneSystem.instance.m_waterLevel : -10000;
            if (!plan.WaterSafe(dest, water) || !plan.WaterSafe(middle.Value, water)) return plan.Refuse("water", dest);
            if (!WallRoute.Clear(from, dest, Walls)) return plan.Refuse("wall_memory", dest);
            if (obstacles.Blocked(from, dest)) { plan.BoundsRejected++; return plan.Refuse("obstacle_bounds", dest); }
            if (obstacles.Logs.Blocked(from, dest)) { plan.BoundsRejected++; return plan.Refuse("log_footprint", dest); }
            var origin = Convert.ToVector3(from);
            var delta = at - origin;
            if (delta.sqrMagnitude < .0001f) return dest;
            // Clear the body's upper capsule. The support probe bounds the
            // permitted step; the lower part must be able to climb stairs.
            bool Obstacle(Collider c)
            {
                if (Doorman.MayIgnoreCollider(c.GetComponentInParent<Door>(), plan.Executing)) return false;
                if (obstacles.LeavingLog(c, from, dest)) return false;
                return true;
            }
            // Curved low log edges are not stairs. Check the lower body for
            // logs even before discovery/rebuild, and while a log is rolling.
            // Ordinary steps retain the existing upper-body clearance rule.
            plan.PhysicsCasts++;
            var lowCasts = Physics.CapsuleCastNonAlloc(origin + Vector3.up * .45f, origin + Vector3.up * 1.4f,
                Radius, delta.normalized, castHits, delta.magnitude, Mask, QueryTriggerInteraction.Ignore);
            if (lowCasts == castHits.Length) { plan.Saturated++; return plan.Refuse("log_cast_saturated", dest); }
            for (var i = 0; i < lowCasts; i++)
            {
                var c = castHits[i].collider;
                if (c.GetComponentInParent<TreeLog>() != null && !obstacles.LeavingLog(c, from, dest)) return plan.Refuse("log_body_cast", dest, c);
            }
            plan.PhysicsCasts++;
            var lowHits = Physics.OverlapCapsuleNonAlloc(at + Vector3.up * .45f, at + Vector3.up * 1.4f,
                Radius, overlaps, Mask, QueryTriggerInteraction.Ignore);
            if (lowHits == overlaps.Length) { plan.Saturated++; return plan.Refuse("log_overlap_saturated", dest); }
            for (var i = 0; i < lowHits; i++)
            {
                var c = overlaps[i];
                if (c.GetComponentInParent<TreeLog>() != null && !obstacles.LeavingLog(c, from, dest)) return plan.Refuse("log_body_overlap", dest, c);
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
