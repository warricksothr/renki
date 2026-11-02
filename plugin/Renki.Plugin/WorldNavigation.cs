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
        static readonly int TriggerMask = Mask | LayerMask.GetMask("character_trigger");
        static readonly NavigationObstacles obstacles = new NavigationObstacles(Mask);
        static readonly RaycastHit[] supportHits = new RaycastHit[64];
        static readonly RaycastHit[] ceilingHits = new RaycastHit[64];
        static readonly RaycastHit[] castHits = new RaycastHit[64];
        static readonly Collider[] overlaps = new Collider[64];
        static readonly RaycastHit[] portalCasts = new RaycastHit[64];
        static readonly Collider[] portalOverlaps = new Collider[64];
        static readonly HashSet<string> supportSlopePieces = new HashSet<string>();
        static Vector3? focus;
        static float nextPortalRefusalTrace;
        static string world;
        const float Step = .6f;
        const float Radius = .4f;

        internal static void Focus(Vector3 target) => focus = target;
        internal static List<LogObstacle> LogObstacles() => obstacles.Snapshot();
        internal static long GeometryRevision => obstacles.Revision;
        internal static double WaterLevel(V3 at) => Math.Max(
            ZoneSystem.instance != null ? ZoneSystem.instance.m_waterLevel : -10000,
            Floating.GetLiquidLevel(Convert.ToVector3(at)));
        internal static void Reset()
        {
            Walls = Array.Empty<Wall>(); obstacles.Clear(); Routes.Clear(); focus = null; world = null; nextPortalRefusalTrace = 0;
            Doorman.Reset();
            Array.Clear(supportHits, 0, supportHits.Length);
            Array.Clear(ceilingHits, 0, ceilingHits.Length);
            Array.Clear(castHits, 0, castHits.Length);
            Array.Clear(overlaps, 0, overlaps.Length);
            Array.Clear(portalCasts, 0, portalCasts.Length);
            Array.Clear(portalOverlaps, 0, portalOverlaps.Length);
            supportSlopePieces.Clear();
        }

        // A nearby raised floor can be a ceiling from the current storey.
        // This only ranks checked approach legs; it never admits movement.
        internal static bool BuildingAbove(V3 at, double targetY)
        {
            var length = Math.Min(6, targetY - at.Y + .6) - .35;
            if (length <= 0) return false;
            var count = Physics.RaycastNonAlloc(Convert.ToVector3(at) + Vector3.up * .35f,
                Vector3.up, ceilingHits, (float)length, Mask, QueryTriggerInteraction.Ignore);
            if (count == ceilingHits.Length) return true;
            for (var i = 0; i < count; i++)
                if (ceilingHits[i].collider?.GetComponentInParent<Piece>() != null) return true;
            return false;
        }

        // The final input guard also covers ordinary navmesh following and
        // direct body-task movement, which need not use a dry planning probe.
        internal static bool PortalClear(Vector3 origin, Vector3 direction)
        {
            var delta = direction.normalized * .6f;
            bool Refuse(string stage, Collider collider = null)
            {
                if (Time.unscaledTime >= nextPortalRefusalTrace)
                {
                    nextPortalRefusalTrace = Time.unscaledTime + 2;
                    Tracer.Emit("navigation", "movement trigger guard refused", new Dictionary<string, object>
                    {
                        ["stage"] = stage,
                        ["pos"] = Convert.ToArray(origin),
                        ["to"] = Convert.ToArray(origin + delta),
                        ["collider"] = collider != null ? collider.transform.root.name + "/" + collider.name : null
                    });
                }
                return false;
            }
            var casts = Physics.CapsuleCastNonAlloc(origin + Vector3.up * .45f, origin + Vector3.up * 1.4f,
                Radius, delta.normalized, portalCasts, delta.magnitude, TriggerMask, QueryTriggerInteraction.Collide);
            if (casts == portalCasts.Length) return Refuse("cast_saturated");
            for (var i = 0; i < casts; i++) if (NavigationObstacles.IsPortal(portalCasts[i].collider)) return Refuse("portal_cast", portalCasts[i].collider);
            var at = origin + delta;
            var hits = Physics.OverlapCapsuleNonAlloc(at + Vector3.up * .45f, at + Vector3.up * 1.4f,
                Radius, portalOverlaps, TriggerMask, QueryTriggerInteraction.Collide);
            if (hits == portalOverlaps.Length) return Refuse("overlap_saturated");
            for (var i = 0; i < hits; i++) if (NavigationObstacles.IsPortal(portalOverlaps[i])) return Refuse("portal_overlap", portalOverlaps[i]);
            return true;
        }
        internal static void Update(Player p)
        {
            var current = ZNet.instance != null ? ZNet.instance.GetWorldName() : "";
            if (world != current) { Reset(); world = current; }
            var pos = p.transform.position;
            obstacles.Update(pos, focus ?? pos, Time.unscaledTime);
        }

        // Start at this floor's level; never snap onto a different storey.
        static V3? Support(V3 at, double fromY, out bool saturated, out object evidence,
            bool footing = false, bool walking = false, bool diagnostics = false, bool footprint = true)
        {
            var top = new Vector3((float)at.X, (float)fromY + Step + .05f, (float)at.Z);
            var count = Physics.RaycastNonAlloc(top, Vector3.down, supportHits, Step * 2 + .1f, Mask, QueryTriggerInteraction.Ignore);
            saturated = count == supportHits.Length;
            evidence = null;
            var selection = new SupportSelection(fromY, Step, footing, walking);
            for (var i = 0; i < count; i++)
                selection.Consider(Convert.ToV3(supportHits[i].point), supportHits[i].normal.y);
            if (diagnostics && (saturated || !selection.Point.HasValue))
                evidence = new
                {
                    reason = saturated ? "saturated" : selection.Refusal,
                    at = new[] { at.X, at.Y, at.Z },
                    fromY,
                    step = Step,
                    footing,
                    walking,
                    rayTop = new[] { (double)top.x, top.y, top.z },
                    rayLength = Step * 2 + .1f,
                    hits = count,
                    normalHits = selection.NormalHits,
                    selected = selection.Selected.HasValue ? new[] { selection.Selected.Value.X, selection.Selected.Value.Y, selection.Selected.Value.Z } : null,
                    // Keep the nearest six hits, not an unbounded scene dump.
                    contacts = supportHits.Take(count).OrderBy(hit => Math.Abs(hit.point.y - fromY)).Take(6).Select(hit => new
                    {
                        pos = new[] { (double)hit.point.x, hit.point.y, hit.point.z },
                        normal = new[] { (double)hit.normal.x, hit.normal.y, hit.normal.z },
                        collider = hit.collider != null ? hit.collider.transform.root.name + "/" + hit.collider.name : null
                    }).ToArray()
                };
            if (diagnostics && walking && selection.Refusal == "slope" && supportSlopePieces.Count < 8)
                for (var i = 0; i < count && supportSlopePieces.Count < 8; i++)
                {
                    var piece = supportHits[i].collider?.GetComponentInParent<Piece>();
                    if (piece != null && supportSlopePieces.Add(piece.name))
                        Tracer.Emit("navigation", "building support slope refused", new Dictionary<string, object>
                        { ["piece"] = piece.name, ["supportQuery"] = evidence });
                }
            var player = Player.m_localPlayer;
            if (!saturated && footing && footprint && count == 0 && player != null)
            {
                // A capsule may stand on a floor edge with no floor beneath
                // its centre. Only the actual grounded body gets this probe;
                // future nodes/midpoints still require their centre support.
                var contact = FootprintSupport.Find(at, Convert.ToV3(player.transform.position), player.IsOnGround(),
                    selection.Refusal, sample =>
                    {
                        var point = Support(sample, fromY, out var full, out _, footing: true, walking: true, footprint: false);
                        return (point, full);
                    }, out var queries, out var fullFootprint);
                saturated |= fullFootprint;
                if (diagnostics && queries > 0)
                    evidence = new
                    {
                        reason = contact.HasValue ? "footprint_contact" : fullFootprint ? "saturated" : "no_footprint_contact",
                        centre = evidence,
                        radius = FootprintSupport.Radius,
                        queries,
                        selected = contact?.ToArray()
                    };
                return saturated ? null : contact;
            }
            return saturated ? null : selection.Point;
        }

        internal sealed class PlanningProbe
        {
            internal sealed class Failure
            {
                public string Stage, Collider;
                public Collider Solid;
                public V3 At;
                public V3? ColliderCenter;
                public double[] SupportHeights, WaterLevels;
                public double? StepDistance;
                public bool? BankClear;
                public object SupportQuery;
            }
            readonly SupportMemo support;
            readonly SupportMemo footing;
            readonly Func<V3, V3, V3?> physical;
            readonly RouteRefusals refusals;
            readonly Func<V3, V3, bool> corridor;
            readonly bool diagnostics;
            readonly bool dry;
            readonly bool bankExit;
            readonly V3? diagnosticGoal;
            bool footprintTraced;
            internal readonly bool Executing;
            long slice;
            internal Failure LastFailure { get; private set; }
            // Local search may wait for an asynchronous navmesh retry after
            // exhausting its frontier. Keep diagnostic evidence across slices.
            internal Failure LastRefusal { get; private set; }
            readonly Dictionary<string, int> failureCounts = new Dictionary<string, int>();
            readonly Dictionary<string, Failure> failureExamples = new Dictionary<string, Failure>();
            readonly List<Failure> bankFailures = new List<Failure>();
            internal object Failures => new
            {
                counts = new Dictionary<string, int>(failureCounts),
                bank = bankExit ? new { executing = Executing, dry, core = typeof(BankExit).Assembly.Location, coreMvid = typeof(BankExit).Assembly.ManifestModule.ModuleVersionId.ToString() } : null,
                examples = (bankExit ? (IEnumerable<Failure>)bankFailures : diagnosticGoal.HasValue
                    ? failureExamples.Values.OrderBy(f => (f.At - diagnosticGoal.Value).Length) : failureExamples.Values).Select(failure => new
                    {
                        stage = failure.Stage,
                        at = failure.At.ToArray(),
                        collider = failure.Collider,
                        supportHeights = failure.SupportHeights,
                        waterLevels = failure.WaterLevels,
                        supportQuery = failure.SupportQuery,
                        stepDistance = failure.StepDistance,
                        bankClear = failure.BankClear
                    }).ToArray()
            };
            public int BoundsRejected, PhysicsCasts, Saturated, HostileRejected;
            public int SupportHits => support.Hits + footing.Hits;
            public int SupportQueries => support.Misses + footing.Misses;
            public PlanningProbe(bool diagnostics = false, bool dry = false, bool executing = false, RouteRefusals refusals = null, Func<V3, V3, bool> corridor = null, bool bankExit = false, V3? diagnosticGoal = null)
            {
                physical = (from, desired) => WorldNavigation.Probe(from, desired, this);
                this.refusals = refusals; this.corridor = corridor;
                this.diagnostics = diagnostics;
                this.dry = dry;
                this.bankExit = bankExit;
                this.diagnosticGoal = diagnosticGoal;
                Executing = executing;
                support = new SupportMemo((at, y) =>
                {
                    var result = Support(at, y, out var full, out var evidence, walking: dry && !bankExit, diagnostics: diagnostics);
                    if (full) Saturated++;
                    return (result, evidence);
                });
                footing = new SupportMemo((at, y) =>
                {
                    var result = Support(at, y, out var full, out var evidence, footing: true, diagnostics: diagnostics);
                    if (full) Saturated++;
                    if (diagnostics && result.HasValue && evidence != null && !footprintTraced)
                    {
                        footprintTraced = true;
                        Tracer.Emit("navigation", "using measured footprint support", new Dictionary<string, object>
                        { ["at"] = at.ToArray(), ["supportQuery"] = evidence });
                    }
                    return (result, evidence);
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
                // An unrelated final probe can hide the obstruction beside
                // a raised entrance. Retain the nearest example per stage
                // for goal-scoped searches; bank sweeps remain chronological.
                if (!diagnosticGoal.HasValue || !failureExamples.TryGetValue(stage, out var prior) ||
                    (at - diagnosticGoal.Value).Length < (prior.At - diagnosticGoal.Value).Length)
                    failureExamples[stage] = LastFailure;
                // Preserve the first bounded bank sweep, including its uphill
                // bearings. The final example alone can conceal why those failed.
                if (bankExit && bankFailures.Count < 16) bankFailures.Add(LastFailure);
                return null;
            }
            internal V3? RefuseSupport(string stage, V3 at, bool actualFooting = false)
            {
                Refuse(stage, at);
                if (LastFailure != null) LastFailure.SupportQuery = actualFooting ? footing.LastEvidence : support.LastEvidence;
                return null;
            }
            internal bool HostileClear(V3 from, V3 to)
            {
                if (corridor == null || corridor(from, to)) return true;
                HostileRejected++; return false;
            }
            internal bool RecentlyBlocked(V3 from, V3 to) => refusals != null && refusals.Blocks(from, to, Time.unscaledTime);
            internal V3? Sample(V3 at, double y) => support.Sample(at, y, Time.unscaledTime, slice);
            internal V3? Footing(V3 at) => footing.Sample(at, at.Y, Time.unscaledTime, slice);
            internal bool WaterSafe(V3 at, double water) => dry
                ? DryTravel.Clear(at, WaterLevel(at), DryTravel.TravelMargin) : at.Y >= water - .5;
            internal bool WaterSafe(V3 from, V3 middle, V3 to, double water)
            {
                if (!bankExit && WaterSafe(to, water) && WaterSafe(middle, water)) return true;
                var start = footing.Sample(from, from.Y, Time.unscaledTime, slice);
                var levels = new[] { WaterLevel(from), WaterLevel(middle), WaterLevel(to) };
                var bankClear = bankExit && start.HasValue &&
                    BankExit.Clear(from, start.Value, middle, to, levels[0], levels[1], levels[2]);
                var clear = Executing && dry && bankClear;
                if (clear) return true;
                Refuse("water", to);
                if (LastFailure != null)
                {
                    LastFailure.SupportHeights = start.HasValue ? new[] { from.Y, start.Value.Y, middle.Y, to.Y } : new[] { from.Y, middle.Y, to.Y };
                    LastFailure.WaterLevels = levels;
                    if (bankExit)
                    {
                        LastFailure.StepDistance = V3.FlatDistance(from, to);
                        LastFailure.BankClear = bankClear;
                    }
                }
                return false;
            }
            internal bool SupportSafe(V3 from, V3 middle, V3 to)
            {
                if (!dry) return true;
                var start = footing.Sample(from, from.Y, Time.unscaledTime, slice);
                if (start.HasValue && DryTravel.ContinuousSupport(from, start.Value, middle, to)) return true;
                if (!start.HasValue) RefuseSupport("support_step", to, actualFooting: true);
                else Refuse("support_step", to);
                if (LastFailure != null && start.HasValue)
                    LastFailure.SupportHeights = new[] { from.Y, start.Value.Y, middle.Y, to.Y };
                return false;
            }
        }

        static V3? Probe(V3 from, V3 desired, PlanningProbe plan)
        {
            var supported = plan.Sample(desired, from.Y);
            if (!supported.HasValue) return plan.RefuseSupport("endpoint_support", desired);
            var middle = plan.Sample((from + desired) * .5, from.Y);
            if (!middle.HasValue) return plan.RefuseSupport("midpoint_support", (from + desired) * .5);
            var dest = supported.Value;
            if (!plan.SupportSafe(from, middle.Value, dest)) return null;
            if (!plan.HostileClear(from, dest)) return plan.Refuse("hostile_corridor", dest);
            if (plan.RecentlyBlocked(from, dest)) return plan.Refuse("recent_live_refusal", dest);
            var at = Convert.ToVector3(dest);
            var water = ZoneSystem.instance != null ? ZoneSystem.instance.m_waterLevel : -10000;
            if (!plan.WaterSafe(from, middle.Value, dest, water)) return null;
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
                Radius, delta.normalized, castHits, delta.magnitude, TriggerMask, QueryTriggerInteraction.Collide);
            if (lowCasts == castHits.Length) { plan.Saturated++; return plan.Refuse("log_cast_saturated", dest); }
            for (var i = 0; i < lowCasts; i++)
            {
                var c = castHits[i].collider;
                // Portal triggers transport the player without an interact
                // action. Ordinary walking must avoid them even before the
                // geometry cache has discovered or rebuilt the trigger.
                if (NavigationObstacles.IsPortal(c)) return plan.Refuse("portal_trigger_cast", dest, c);
                if (!c.isTrigger && c.GetComponentInParent<TreeLog>() != null && !obstacles.LeavingLog(c, from, dest)) return plan.Refuse("log_body_cast", dest, c);
            }
            plan.PhysicsCasts++;
            var lowHits = Physics.OverlapCapsuleNonAlloc(at + Vector3.up * .45f, at + Vector3.up * 1.4f,
                Radius, overlaps, TriggerMask, QueryTriggerInteraction.Collide);
            if (lowHits == overlaps.Length) { plan.Saturated++; return plan.Refuse("log_overlap_saturated", dest); }
            for (var i = 0; i < lowHits; i++)
            {
                var c = overlaps[i];
                if (NavigationObstacles.IsPortal(c)) return plan.Refuse("portal_trigger_overlap", dest, c);
                if (!c.isTrigger && c.GetComponentInParent<TreeLog>() != null && !obstacles.LeavingLog(c, from, dest)) return plan.Refuse("log_body_overlap", dest, c);
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
