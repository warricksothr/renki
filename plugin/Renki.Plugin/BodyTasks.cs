using System.Collections.Generic;
using System.Linq;
using Renki.Movement;
using Renki.Protocol;
using Renki.Tasks;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>
    /// Getting from here to a point: a path from the game's navmesh
    /// (straight line when there is none), steering along it, and the
    /// stuck recovery of docs/BRIDGE-PROTOCOL.md 7.3. MoveTo and Follow both
    /// drive one of these.
    /// </summary>
    sealed class Walker
    {
        readonly PathFollower path = new PathFollower();
        readonly StuckDetector stuck = new StuckDetector();
        readonly Doorman doors = new Doorman();
        readonly Climber climb = new Climber();
        readonly ApproachRecovery approach = new ApproachRecovery();
        readonly EntranceHints entrances = new EntranceHints();
        readonly RouteRefusals refusals = new RouteRefusals();
        readonly RouteProgress execution = new RouteProgress();
        V3? commandedLeg;
        readonly HostileRoute hostiles = new HostileRoute();
        readonly System.Func<V3, V3, bool> hostileClearance;
        bool avoidHostiles;
        readonly SprintPacing sprint = new SprintPacing();
        readonly List<Piece> entrancePieces = new List<Piece>();
        Vector3? entrance;
        Vector3 NavigationGoal => entrance ?? goal;
        double NavigationWithin => entrance.HasValue ? .25 : arrivalWithin;
        double NavigationLevel => entrance.HasValue ? .6 : Arrival.Level;
        readonly WorldNavigation.PlanningProbe dryProbe;
        readonly WorldNavigation.PlanningProbe bankProbe;
        BankExit bankExit = new BankExit();
        DryStaging dryStaging = new DryStaging();
        bool bankMoving;
        public Walker(System.Func<V3, V3, bool> boundary = null)
        {
            hostileClearance = (from, to) => (boundary == null || boundary(from, to)) && (!avoidHostiles || hostiles.Clear(from, to));
            dryProbe = new WorldNavigation.PlanningProbe(diagnostics: true, dry: true, executing: true, corridor: hostileClearance);
            bankProbe = new WorldNavigation.PlanningProbe(diagnostics: true, dry: true, executing: true, corridor: hostileClearance, bankExit: true);
        }
        bool dryGround;
        double wadeDepth;
        readonly List<Vector3> scratch = new List<Vector3>();
        readonly RouteTrail trail = new RouteTrail();
        readonly NavmeshCandidates navmeshCandidates = new NavmeshCandidates();
        RouteValidation replay;
        WorldNavigation.PlanningProbe replayProbe;
        long replayID;
        V3? replayWaypoint;
        bool replaying, replayNavmesh, skipReuse, trailStarted, reuseRoutes;
        double replayMs, replayWorstMs;
        Vector3 goal;
        double arrivalWithin = .5;
        bool hasGoal, checkedOnly;
        RouteSearch localRoute;
        bool preferGuidance;
        bool localUnderFloorExit;
        WorldNavigation.PlanningProbe localProbe;
        Outcome navigationFailure;
        string localWaterBlocked;
        float localStarted, nextWallCheck, nextNavmeshCheck, nextDryNavmeshTrace, nextDryRefusalTrace;
        double localMs, localWorstMs, localGuidanceMs;
        float strafeUntil;
        float strafeSign = 1;

        public bool StraightLine { get; private set; }

        /// <summary>Why the walker will not go on: the straight line it fell
        /// back to crosses deep water. Null while it can.</summary>
        public string Blocked { get; private set; }

        /// <summary>The deep_water failure, once the bot is on dry land. A
        /// swimming bot is not left treading water: Steer takes it ashore
        /// first, and the re-plan there decides.</summary>
        public Outcome Refused() =>
            Player.m_localPlayer?.IsSwimming() == true || dryGround && swimming ? null : navigationFailure ?? (Blocked != null
                ? Outcome.Failed(Wire.ReasonDeepWater, Blocked) : null);

        /// <summary>The terrain's height at x, z, or NaN where it is not
        /// loaded.</summary>
        static double Ground(double x, double z) =>
            ZoneSystem.instance != null && ZoneSystem.instance.GetGroundHeight(new Vector3((float)x, 0, (float)z), out var h)
                ? h : double.NaN;

        static double WaterLevel => ZoneSystem.instance != null ? ZoneSystem.instance.m_waterLevel : -10000;

        /// <summary>The navmesh path ends above or below the goal's level:
        /// under a raised floor, at the foot of a rock. The bot can stand
        /// under the goal but not reach it.</summary>
        public bool OffLevel { get; private set; }

        Vector3 end;

        /// <summary>How near the end of an off-level path counts as there.</summary>
        const double AtEnd = 1.0;

        /// <summary>How near the goal, across the ground, a path's end must
        /// be for an end off the goal's level to mean the goal is out of
        /// reach: under it, or at the foot of the rock it is on.</summary>
        const double NearGoal = 4.0;

        /// <summary>Try other approaches before failing an off-level goal.</summary>
        public Outcome Unreachable(Vector3 pos, double within)
        {
            if (localRoute != null || replay != null) return null;
            var here = Convert.ToV3(pos);
            var atEnd = OffLevel && (V3.FlatDistance(here, Convert.ToV3(goal)) <= within ||
                                    V3.FlatDistance(here, Convert.ToV3(end)) <= AtEnd);
            var step = approach.Update(here, Convert.ToV3(goal), atEnd, Time.time,
                candidate => Detour(pos, candidate));
            if (step == ApproachStep.None) return null;
            Tracer.Emit("walker", step == ApproachStep.Detour ? "off-level goal; repositioning"
                : step == ApproachStep.Retry ? "off-level goal; retrying path" : "off-level goal; approaches exhausted",
                new Dictionary<string, object>
                {
                    ["attempt"] = approach.Attempts,
                    ["pos"] = Convert.ToArray(pos),
                    ["goal"] = Convert.ToArray(goal),
                    ["detour"] = approach.Target?.ToArray(),
                });
            if (step != ApproachStep.GiveUp)
            {
                ResetStuck(pos);
                Repath(pos);
                return null;
            }
            var dy = goal.y - pos.y;
            return Outcome.Failed(Wire.ReasonUnreachable,
                $"the goal is {System.Math.Abs(dy):0.0} m {(dy > 0 ? "above" : "below")} and the navmesh does not reach it after {approach.Attempts} approach attempts");
        }

        V3? Detour(Vector3 from, V3 candidate)
        {
            scratch.Clear();
            var to = new Vector3((float)candidate.X, (float)candidate.Y, (float)candidate.Z);
            if (Pathfinding.instance == null ||
                !Pathfinding.instance.GetPath(from, to, scratch, Pathfinding.AgentType.Humanoid, true, true, false) ||
                scratch.Count == 0) return null;
            return SafeDetour(from, candidate) && WallRoute.Clear(Convert.ToV3(from), scratch.Select(Convert.ToV3).ToArray(), WorldNavigation.Walls)
                ? Convert.ToV3(scratch[scratch.Count - 1]) : (V3?)null;
        }

        bool SafeDetour(Vector3 from, V3 candidate) =>
            ApproachRecovery.SafePath(Convert.ToV3(from), candidate, scratch.Select(Convert.ToV3).ToArray(),
                (a, b) => DeepWater.Crossing(a, b, Ground, WaterLevel) != null);

        public void SetGoal(Vector3 from, Vector3 to, double within = .5, bool reuseRoutes = false, bool dryGround = false, bool avoidHostiles = false, bool wade = false)
        {
            var player = Player.m_localPlayer;
            wadeDepth = wade && player != null ? Wading.DepthLimit(player.m_swimDepth) : 0;
            if (wade && (!dryGround || !avoidHostiles || wadeDepth <= 0))
                throw new TaskRefused(Wire.ReasonNotAllowed, "checked wading needs supported hostile-avoiding travel and a known swimming threshold");
            dryProbe.WadeDepth = wadeDepth;
            if (wade) Tracer.Emit("walker", "checked wading enabled", new Dictionary<string, object>
            { ["maxDepth"] = wadeDepth, ["swimDepth"] = player.m_swimDepth, ["settlingReserve"] = Wading.Settling });
            checkedOnly = false;
            bankExit = new BankExit(); bankMoving = false;
            dryStaging = new DryStaging();
            preferGuidance = false;
            approach.Reset();
            refusals.Clear();
            execution.Reset(); commandedLeg = null;
            entrances.Reset(); entrance = null;
            goal = to;
            arrivalWithin = within;
            this.reuseRoutes = reuseRoutes;
            this.dryGround = dryGround;
            this.avoidHostiles = avoidHostiles;
            WorldNavigation.Focus(to);
            hasGoal = true;
            skipReuse = false;
            navmeshCandidates.Reset();
            navmeshCandidates.Observe(Convert.ToV3(from), Convert.ToV3(to));
            nextDryNavmeshTrace = 0;
            trailStarted = false;
            RecordGrounded(from);
            Repath(from);
        }

        // A host-validated route is the movement authority. Do not replace it
        // with another navmesh query or an unvalidated stuck-recovery strafe.
        public void SetCheckedRoute(Vector3 to, IReadOnlyList<V3> points)
        {
            checkedOnly = hasGoal = true;
            goal = to;
            navigationFailure = null; Blocked = null;
            localRoute = null; replay = null;
            climb.Stop(); strafeUntil = 0;
            path.Reach = .25; path.FinalReach = .125;
            path.Level = .6;
            path.Set(points);
        }

        public IReadOnlyList<V3> RemainingCheckedRoute(Vector3 pos) => path.Remaining(Convert.ToV3(pos));

        void RecordGrounded(Vector3 pos)
        {
            if (!reuseRoutes || Player.m_localPlayer?.IsOnGround() != true) return;
            if (!trailStarted) { trail.Reset(Convert.ToV3(pos)); trailStarted = true; }
            else trail.Record(Convert.ToV3(pos));
        }

        /// <summary>Only a completed point move makes its actual trail reusable.</summary>
        public void Arrived(Vector3 pos)
        {
            if (!trailStarted || Player.m_localPlayer?.IsOnGround() != true) return;
            trail.Record(Convert.ToV3(pos), final: true);
            if (WorldNavigation.Routes.Remember(trail) is long id)
                Tracer.Emit("walker", "recorded successful route", new Dictionary<string, object>
                {
                    ["route"] = id,
                    ["points"] = trail.Points.Count,
                    ["stored"] = WorldNavigation.Routes.Count,
                });
        }

        void Repath(Vector3 from, bool forceLocal = false)
        {
            navigationFailure = null;
            localRoute = null;
            replay = null;
            replaying = false;
            replayNavmesh = false;
            replayWaypoint = null;
            if (entrance.HasValue && Arrival.Reached(Convert.ToV3(from), Convert.ToV3(entrance.Value), .5, .6))
            {
                entrance = null; navmeshCandidates.Reset();
            }
            if (!entrance.HasValue && dryGround && Doorman.Enabled)
            {
                entrancePieces.Clear();
                Piece.GetAllPiecesInRadius(goal, 12, entrancePieces);
                var floors = new List<V3>();
                var probe = new WorldNavigation.PlanningProbe(dry: true, refusals: refusals, corridor: hostileClearance, wadeDepth: wadeDepth);
                probe.BeginSlice();
                foreach (var piece in entrancePieces)
                {
                    if (!piece.TryGetComponent<Door>(out var door) || door.m_nview == null || !door.m_nview.IsValid()) continue;
                    var at = Convert.ToV3(door.transform.position);
                    // Door pivots sit above their threshold. Sample the actual
                    // supported entrance floor, never substitute the pivot.
                    var floor = probe.Sample(at, at.Y - 1);
                    if (floor.HasValue) floors.Add(floor.Value);
                }
                if (entrances.Select(Convert.ToV3(from), Convert.ToV3(goal), floors) is V3 selected)
                {
                    entrance = Convert.ToVector3(selected); navmeshCandidates.Reset();
                    Tracer.Emit("walker", "routing raised goal through observed entrance", new Dictionary<string, object>
                    {
                        ["from"] = Convert.ToArray(from),
                        ["goal"] = Convert.ToArray(goal),
                        ["entrance"] = selected.ToArray(),
                    });
                }
            }
            path.Reach = 1;
            path.Level = dryGround ? .6 : Arrival.Level;
            path.FinalReach = NavigationWithin;
            var target = approach.Target is V3 via
                ? new Vector3((float)via.X, (float)via.Y, (float)via.Z) : NavigationGoal;
            if (reuseRoutes && !forceLocal && !skipReuse && approach.Target == null &&
                WorldNavigation.Routes.Find(Convert.ToV3(from), Convert.ToV3(target)) is RouteHint hint)
            {
                replayProbe = new WorldNavigation.PlanningProbe(diagnostics: true, dry: dryGround, refusals: refusals, corridor: hostileClearance, wadeDepth: wadeDepth);
                replay = new RouteValidation(Convert.ToV3(from), Convert.ToV3(target), hint, replayProbe.Probe, NavigationWithin, NavigationLevel);
                replayID = hint.ID;
                replayMs = replayWorstMs = 0;
                path.Set(new V3[0]);
                OffLevel = StraightLine = false;
                Blocked = null;
                return;
            }
            if (dryGround)
            {
                // The navmesh can guide us through a raised-floor entrance, but
                // can also cross an inlet. Validate all its legs before use.
                OffLevel = StraightLine = false;
                if (!forceLocal && BeginDryNavmesh(from, target)) return;
                BeginLocal(from, target);
                return;
            }
            scratch.Clear();
            var pf = Pathfinding.instance;
            StraightLine = !(pf != null &&
                             pf.GetPath(from, target, scratch, Pathfinding.AgentType.Humanoid, false, true, false) &&
                             scratch.Count > 0);
            Blocked = null;
            // Detour validation already obtained a complete path. If a tile
            // changes before this second query, wait for the leg's retry;
            // never walk an unvalidated fallback away from the real goal.
            if (approach.Target != null && (StraightLine || !SafeDetour(from, Convert.ToV3(target)) ||
                !WallRoute.Clear(Convert.ToV3(from), scratch.Select(Convert.ToV3).ToArray(), WorldNavigation.Walls)))
            {
                path.Set(new V3[0]);
                return;
            }
            if (StraightLine)
            {
                scratch.Clear();
                scratch.Add(goal);
                // The straight line is the fallback that drowned the bot:
                // it does not know about lakes.
                if (DeepWater.Crossing(Convert.ToV3(from), Convert.ToV3(goal), Ground, WaterLevel) is V3 wet)
                {
                    Blocked = $"deep water in the way at ({wet.X:0}, {wet.Z:0})";
                    Tracer.Emit("walker", "no navmesh path, and deep water on the straight line", new Dictionary<string, object>
                    {
                        ["from"] = Convert.ToArray(from),
                        ["to"] = Convert.ToArray(goal),
                        ["water"] = wet.ToArray(),
                    });
                }
            }
            path.Set(scratch.Select(Convert.ToV3));
            // The navmesh gives the way to the nearest point it can reach,
            // which may be under a floor or behind a shut door.
            end = scratch[scratch.Count - 1];
            var shortBy = Vector3.Distance(end, target);
            // Off level only when the end is near the goal across the ground:
            // a path that stops short farther off may yet go on, through a
            // door or after a re-path.
            if (approach.Target == null)
            {
                OffLevel = !StraightLine && System.Math.Abs(end.y - goal.y) > Arrival.Level &&
                           V3.FlatDistance(Convert.ToV3(end), Convert.ToV3(goal)) <= NearGoal;
                if (!OffLevel) approach.Reset();
            }
            var data = new Dictionary<string, object>
            {
                ["from"] = Convert.ToArray(from),
                ["to"] = Convert.ToArray(target),
                ["waypoints"] = scratch.Count,
            };
            if (shortBy > 1) data["end"] = Convert.ToArray(end);
            var crossesWall = !WallRoute.Clear(Convert.ToV3(from), scratch.Select(Convert.ToV3).ToArray(), WorldNavigation.Walls);
            // A complete-looking path can still cross a wall missing from
            // its navmesh. A partial path must never turn into a straight
            // drive from its endpoint to the original goal.
            var shortAcrossGround = V3.FlatDistance(Convert.ToV3(end), Convert.ToV3(target)) > 1.5;
            if (approach.Target == null && (StraightLine || crossesWall || shortAcrossGround || forceLocal))
            {
                data["why"] = crossesWall ? "known wall" : shortAcrossGround ? "partial navmesh path" : forceLocal ? "stuck" : "no navmesh path";
                data["walls"] = WorldNavigation.Walls.Count;
                if (!StraightLine && !forceLocal)
                {
                    var prefix = WallRoute.Prefix(Convert.ToV3(from), Convert.ToV3(target), scratch.Select(Convert.ToV3).ToArray(), WorldNavigation.Walls);
                    if (prefix.Count > 0)
                    {
                        path.Set(prefix);
                        end = Convert.ToVector3(prefix[prefix.Count - 1]);
                        OffLevel = System.Math.Abs(end.y - goal.y) > Arrival.Level &&
                            V3.FlatDistance(Convert.ToV3(end), Convert.ToV3(goal)) <= NearGoal;
                        data["end"] = Convert.ToArray(end);
                        data["waypoints"] = prefix.Count;
                        Tracer.Emit("walker", "following safe navmesh prefix before local search", data);
                        return;
                    }
                }
                Tracer.Emit("walker", "planning a local route", data);
                BeginLocal(from, target);
                return;
            }
            Tracer.Emit("walker", StraightLine ? "no navmesh path; straight line"
                : shortBy > 1 ? $"path stops {shortBy:0.0} m short" : "path", data);
        }

        public void ResetStuck(Vector3 pos)
        {
            stuck.Reset(Convert.ToV3(pos), Time.time);
            strafeUntil = 0;
            climb.Stop();
            climbWaiting = false;
        }

        /// <summary>Steers toward the goal for this step. Returns false when stuck recovery gave up.</summary>
        public bool Steer(Vector3 pos, bool run, bool climbObstacles = true)
        {
            if (!hasGoal) return true;
            var attempted = commandedLeg;
            commandedLeg = null;
            if (dryGround && !checkedOnly && execution.Stalled(Convert.ToV3(pos), attempted, Time.unscaledTime))
            {
                var from = Convert.ToV3(pos);
                refusals.Remember(from, attempted.Value, Time.unscaledTime, () => true);
                Tracer.Emit("walker", "remembered stalled execution approach", new Dictionary<string, object>
                { ["from"] = from.ToArray(), ["to"] = attempted.Value.ToArray(), ["count"] = refusals.Count });
                Repath(pos, true);
                Driver.Stand();
                return true;
            }
            navmeshCandidates.Observe(Convert.ToV3(pos), Convert.ToV3(NavigationGoal));
            if (checkedOnly)
            {
                Driver.Move = Toward(pos).normalized;
                Driver.Run = run;
                return true;
            }
            if (!climbObstacles) { climb.Stop(); climbWaiting = false; }
            RecordGrounded(pos);
            if (ExitBank(pos, goal)) return true;
            if (Swim(pos, ref run)) return true;
            if (replay != null) return PlanReplay(pos);
            if (localRoute != null) return PlanLocal(pos);
            if (navigationFailure != null) { Driver.Stand(); return true; }
            if (dryGround && path.AdvanceChecked(Convert.ToV3(pos), .4, (from, to) =>
            {
                dryProbe.BeginSlice();
                var supported = dryProbe.Probe(from, to);
                return supported.HasValue && (supported.Value - to).Length < .1;
            })) ResetStuck(pos);
            if (replaying && path.Next(Convert.ToV3(pos)) is V3 replayNext)
            {
                // Validation is a planning hint, not a lease on unchanged
                // geometry. Recheck the immediate short leg every step.
                // Short replay legs can turn around a doorway without
                // covering the stuck detector's 3 m circle. Consuming a
                // validated waypoint is progress on this finite path.
                if (!(replayWaypoint is V3 previous) || (replayNext - previous).Length > .01)
                {
                    ResetStuck(pos);
                    replayWaypoint = replayNext;
                }
                replayProbe.BeginSlice();
                var supported = replayProbe.Probe(Convert.ToV3(pos), replayNext);
                if (!(supported is V3 at) || (at - replayNext).Length > .65)
                {
                    if (Doorman.Waiting(pos)) { Driver.Stand(); stuck.Hold(Time.time); return true; }
                    RememberSupportRefusal(Convert.ToV3(pos), replayNext, replayProbe.LastFailure?.Stage);
                    RejectReplay(pos, "next leg changed");
                    return true;
                }
            }
            if (Time.unscaledTime >= nextWallCheck)
            {
                nextWallCheck = Time.unscaledTime + .25f;
                var next = path.Next(Convert.ToV3(pos));
                if (next is V3 above && V3.FlatDistance(Convert.ToV3(pos), above) <= path.Reach &&
                    System.Math.Abs(pos.y - above.Y) > path.Level)
                {
                    Tracer.Emit("walker", "waypoint on another floor; replanning", new Dictionary<string, object>
                    {
                        ["pos"] = Convert.ToArray(pos),
                        ["waypoint"] = above.ToArray(),
                    });
                    Repath(pos, true);
                    Driver.Stand();
                    return true;
                }
                if (next is V3 waypoint && !WallRoute.Clear(Convert.ToV3(pos), waypoint, WorldNavigation.Walls))
                {
                    Repath(pos, true);
                    Driver.Stand();
                    return true;
                }
                if (next == null && approach.Target == null && !Arrival.Reached(Convert.ToV3(pos), Convert.ToV3(goal), .5))
                {
                    Repath(pos, true);
                    Driver.Stand();
                    return true;
                }
            }
            // The detour has its own short deadline. Do not strafe or jump
            // off its validated path while waiting to retry the real goal.
            if (approach.Target != null) stuck.Hold(Time.time);
            if (climb.Active && Climb(pos)) return true;
            var recovery = stuck.Update(Convert.ToV3(pos), Time.time);
            if (replaying && recovery != Recovery.None)
            {
                RejectReplay(pos, "no progress on checked route");
                return true;
            }
            if (recovery != Recovery.None)
                Tracer.Emit("walker", recovery == Recovery.GiveUp ? "stuck; giving up"
                    : !climbObstacles && (recovery == Recovery.Jump || recovery == Recovery.Climb)
                        ? "stuck; keeping feet on support" : $"stuck; trying {recovery}",
                    new Dictionary<string, object>
                    {
                        ["attempt"] = stuck.Attempts,
                        ["pos"] = Convert.ToArray(pos),
                        ["center"] = stuck.Center.ToArray(),
                        ["goal"] = Convert.ToArray(goal),
                        ["next"] = path.Next(Convert.ToV3(pos))?.ToArray(),
                    });
            // A closed door beside a stuck bot is the likeliest wall: open it
            // in place of this round's jump, strafe or re-path.
            if (recovery != Recovery.None && recovery != Recovery.GiveUp && doors.Stuck(Player.m_localPlayer, pos))
                recovery = Recovery.None;
            switch (recovery)
            {
                case Recovery.Jump:
                    if (climbObstacles) Driver.Jump = true;
                    break;
                case Recovery.Strafe:
                    strafeUntil = Time.time + 1f;
                    strafeSign = -strafeSign;
                    break;
                case Recovery.Repath:
                    Repath(pos, true);
                    if (localRoute != null) { Driver.Stand(); return true; }
                    break;
                case Recovery.Climb:
                    if (!climbObstacles) break;
                    climb.Start(Time.time);
                    climbWaiting = false;
                    if (Climb(pos)) return true;
                    break;
                case Recovery.GiveUp:
                    return false;
            }
            var dir = Toward(pos);
            if (Time.time < strafeUntil)
                dir = Vector3.Cross(Vector3.up, dir.normalized) * strafeSign + dir.normalized * 0.3f;
            Driver.Move = dir.sqrMagnitude > 0.0001f ? dir.normalized : Vector3.zero;
            doors.Walk(Player.m_localPlayer, pos, Driver.Move);
            if (dryGround && Driver.Move.sqrMagnitude > .0001f)
            {
                dryProbe.BeginSlice();
                var from = Convert.ToV3(pos);
                var next = path.Next(from);
                var clearance = DryTravel.ClearanceDistance(from, next, Time.time < strafeUntil);
                if (!dryProbe.Probe(from, from + Convert.ToV3(Driver.Move) * clearance).HasValue)
                {
                    if (Time.unscaledTime >= nextDryRefusalTrace)
                    {
                        nextDryRefusalTrace = Time.unscaledTime + 2;
                        Tracer.Emit("walker", "dry movement corridor refused", new Dictionary<string, object>
                        {
                            ["from"] = from.ToArray(),
                            ["next"] = next?.ToArray(),
                            ["lookahead"] = clearance,
                            ["failure"] = dryProbe.LastFailure?.Stage,
                            ["supportHeights"] = dryProbe.LastFailure?.SupportHeights,
                            ["supportQuery"] = dryProbe.LastFailure?.SupportQuery,
                            ["collider"] = dryProbe.LastFailure?.Collider,
                        });
                    }
                    Driver.Stand();
                    if (Doorman.Waiting(pos)) { stuck.Hold(Time.time); return true; }
                    var failure = dryProbe.LastFailure;
                    RememberSupportRefusal(from, from + Convert.ToV3(dir.normalized) * clearance, failure?.Stage);
                    var solid = failure?.Solid;
                    if (solid != null && solid.GetComponentInParent<Door>() == null &&
                        (failure.Stage == "body_cast" || failure.Stage == "body_overlap"))
                    {
                        var bounds = solid.bounds; var matrix = solid.transform.localToWorldMatrix;
                        var checkedFrame = -1; var current = false;
                        bool Unchanged()
                        {
                            if (checkedFrame != Time.frameCount)
                            {
                                checkedFrame = Time.frameCount;
                                current = solid != null && solid.enabled && solid.gameObject.activeInHierarchy &&
                                    solid.bounds.Equals(bounds) && solid.transform.localToWorldMatrix.Equals(matrix);
                            }
                            return current;
                        }
                        refusals.Remember(from, from + Convert.ToV3(dir.normalized) * clearance, Time.unscaledTime, Unchanged);
                        Tracer.Emit("walker", "remembered refused movement approach", new Dictionary<string, object>
                        { ["from"] = from.ToArray(), ["at"] = failure.At.ToArray(), ["collider"] = failure.Collider, ["count"] = refusals.Count });
                    }
                    BeginLocal(pos, NavigationGoal);
                    return true;
                }
            }
            // A strafe walks, so it cannot carry the bot out of the stuck
            // circle on its own and pass for progress.
            var player = Player.m_localPlayer;
            Driver.Run = player != null && sprint.Run(run && Time.time >= strafeUntil, player.GetStamina(), player.GetMaxStamina());
            if (dryGround && Driver.Move.sqrMagnitude > .0001f)
                commandedLeg = path.Next(Convert.ToV3(pos)) ?? Convert.ToV3(NavigationGoal);
            return true;
        }

        void RememberSupportRefusal(V3 from, V3 to, string stage)
        {
            if (stage != "support_step") return;
            var revision = WorldNavigation.GeometryRevision;
            refusals.Remember(from, to, Time.unscaledTime, () => WorldNavigation.GeometryRevision == revision);
            Tracer.Emit("walker", "remembered refused slope approach", new Dictionary<string, object>
            { ["from"] = from.ToArray(), ["to"] = to.ToArray(), ["count"] = refusals.Count });
        }

        public bool DepartDryBank(Vector3 pos, Vector3 toward)
        {
            dryGround = avoidHostiles = true;
            return ExitBank(pos, toward);
        }

        public Outcome WadingRefusal(Vector3 pos) => wadeDepth > 0 && !WadingReady(pos) ? navigationFailure : null;

        bool WadingReady(Vector3 pos)
        {
            var p = Player.m_localPlayer;
            var from = Convert.ToV3(pos);
            dryProbe.BeginSlice();
            var footing = dryProbe.Footing(from);
            var water = WorldNavigation.WaterLevel(from);
            var depth = p != null ? System.Math.Min(wadeDepth, Wading.DepthLimit(p.m_swimDepth)) : 0;
            if (p != null && depth >= wadeDepth && Wading.Supported(from, footing, water, depth,
                p.IsOnGround(), p.IsSwimming())) return true;
            var reason = p == null ? "no_player" : p.IsSwimming() ? "swimming" : !p.IsOnGround() ? "ground_contact"
                : !footing.HasValue ? "unsupported_footing" : depth < wadeDepth ? "swim_threshold_changed"
                : !DryTravel.PreferFooting(from.Y, footing.Value.Y, null) ? "footing_offset" : "water_depth";
            navigationFailure = Outcome.Failed(reason == "swimming" || reason == "water_depth" ? Wire.ReasonDeepWater : Wire.ReasonUnreachable,
                "checked wading stopped: " + reason);
            Tracer.Emit("walker", "checked wading refused", new Dictionary<string, object>
            {
                ["pos"] = from.ToArray(),
                ["support"] = footing?.ToArray(),
                ["waterLevel"] = water,
                ["maxDepth"] = depth,
                ["grounded"] = p?.IsOnGround(),
                ["swimming"] = p?.IsSwimming()
            });
            return false;
        }

        bool ExitBank(Vector3 pos, Vector3 toward)
        {
            if (!dryGround || wadeDepth > 0) return false;
            var p = Player.m_localPlayer;
            if (p == null) return false;
            var from = Convert.ToV3(pos);
            bankProbe.BeginSlice();
            var support = bankProbe.Footing(from);
            var water = WorldNavigation.WaterLevel(from);
            if (support.HasValue && (bankMoving ? BankExit.Recovered(from, support.Value, water, p.GetLastGroundNormal().y) :
                !BankExit.NeedsDeparture(from, support.Value, water, localRoute != null || replay != null)))
            {
                if (!bankMoving) return false;
                bankMoving = false;
                Tracer.Emit("walker", "dry bank departure regained planning buffer", new Dictionary<string, object>
                {
                    ["pos"] = from.ToArray(),
                    ["waterLevel"] = water,
                    ["support"] = support.Value.ToArray(),
                    ["groundNormal"] = Convert.ToArray(p.GetLastGroundNormal())
                });
                if (hasGoal) Repath(pos);
                Driver.Stand(); return true;
            }
            if (!bankMoving && (!support.HasValue || p.InWater() || p.IsSwimming() || !p.IsOnGround())) return false;
            Driver.Stand(); stuck.Hold(Time.time);
            if (!bankMoving)
            {
                bankMoving = true;
                Tracer.Emit("walker", "checking bounded dry bank departure", new Dictionary<string, object>
                {
                    ["pos"] = from.ToArray(),
                    ["waterLevel"] = water,
                    ["support"] = support?.ToArray(),
                    ["groundNormal"] = Convert.ToArray(p.GetLastGroundNormal()),
                    ["groundCollider"] = p.GetLastGroundCollider()?.name,
                    ["planning"] = PlanningEvidence()
                });
            }
            var eligible = support.HasValue && !p.InWater() && !p.IsSwimming() && p.IsOnGround() &&
                DryTravel.Clear(from, water, BankExit.ActualMargin) && DryTravel.Clear(support.Value, water, BankExit.ActualMargin);
            var stalls = bankExit.Stalls;
            var step = eligible ? bankExit.Next(from, Convert.ToV3(toward), Time.unscaledTime, bankProbe.Probe) : null;
            if (bankExit.Stalls > stalls)
                Tracer.Emit("walker", "dry bank step stalled; checking another bearing", new Dictionary<string, object>
                {
                    ["pos"] = from.ToArray(),
                    ["stalls"] = bankExit.Stalls,
                    ["samples"] = bankExit.Samples,
                    ["groundNormal"] = Convert.ToArray(p.GetLastGroundNormal()),
                    ["groundCollider"] = p.GetLastGroundCollider()?.name
                });
            if (!eligible || bankExit.Refused)
            {
                var reason = !eligible ? "live_eligibility_lost" : bankExit.RefusalReason;
                navigationFailure = Outcome.Failed(Wire.ReasonUnreachable, "bounded dry bank departure refused: " + reason);
                Tracer.Emit("walker", "dry bank departure refused", new Dictionary<string, object>
                { ["pos"] = from.ToArray(), ["waterLevel"] = water, ["eligible"] = eligible, ["reason"] = reason, ["samples"] = bankExit.Samples, ["failures"] = bankProbe.Failures, ["planning"] = PlanningEvidence() });
                return true;
            }
            // Ordinary walking input; Run remains false. An attenuated input
            // cannot reliably overcome native uphill friction on a checked step.
            if (step.HasValue) Driver.Move = (Convert.ToVector3(step.Value) - pos).normalized;
            return true;
        }

        object PlanningEvidence() => new
        {
            local = localRoute == null ? null : new
            {
                status = localRoute.Status.ToString(),
                expanded = localRoute.Expanded,
                probes = localRoute.Probes,
                elapsedS = System.Math.Round(Time.unscaledTime - localStarted, 2),
                frontier = FrontierEvidence(),
                failures = localProbe?.Failures
            },
            replay = replay == null ? null : new
            {
                status = replay.Status.ToString(),
                samples = replay.Samples,
                checking = replay.Checking?.ToArray(),
                checkingFrom = replay.CheckingFrom.ToArray(),
                failures = replayProbe?.Failures
            }
        };

        object FrontierEvidence() => localRoute == null ? null : new
        {
            target = localRoute.Target.ToArray(),
            phase = localRoute.Phase,
            highest = localRoute.HighestExpanded?.ToArray(),
            closest = localRoute.ClosestExpanded?.ToArray(),
            gap = localRoute.ClosestExpanded.HasValue ? (double?)localRoute.ClosestGap : null,
            goalFloorExpanded = localRoute.GoalFloorExpanded
        };

        void RejectReplay(Vector3 pos, string why)
        {
            if (!replayNavmesh) { WorldNavigation.Routes.Remove(replayID); skipReuse = true; }
            Tracer.Emit("walker", replayNavmesh ? "navmesh candidate rejected" : "recorded route rejected", new Dictionary<string, object>
            {
                ["why"] = why,
                ["route"] = replayID,
                ["pos"] = Convert.ToArray(pos),
                ["next"] = path.Next(Convert.ToV3(pos))?.ToArray(),
                ["checking"] = replay?.Checking?.ToArray(),
                ["checkingFrom"] = replay?.CheckingFrom.ToArray(),
                ["supported"] = replay?.Supported?.ToArray(),
                ["validationRejection"] = replay?.Rejection,
                ["maxHintHeightOffset"] = replay?.MaxHintHeightOffset,
                ["samples"] = replay?.Samples,
                ["failure"] = replayProbe?.LastFailure?.Stage,
                ["supportHeights"] = replayProbe?.LastFailure?.SupportHeights,
                ["supportQuery"] = replayProbe?.LastFailure?.SupportQuery,
                ["collider"] = replayProbe?.LastFailure?.Collider,
                ["ms"] = System.Math.Round(replayMs, 2),
                ["worstFrameMs"] = System.Math.Round(replayWorstMs, 2),
            });
            if (replayNavmesh)
            {
                replay = null; replaying = replayNavmesh = false;
                // An asynchronous candidate must not discard the local search
                // or restart its elapsed-time bound after a failed validation.
                if (localRoute == null) BeginLocal(pos, NavigationGoal);
                else path.Set(new V3[0]);
            }
            else Repath(pos, true);
            Driver.Stand();
        }

        bool PlanReplay(Vector3 pos)
        {
            Driver.Stand();
            stuck.Hold(Time.time);
            replayProbe.BeginSlice();
            var watch = System.Diagnostics.Stopwatch.StartNew();
            for (var i = 0; i < 8 && watch.Elapsed.TotalMilliseconds < 2 && replay.Status == RouteStatus.Searching; i++)
                replay.Advance(1);
            watch.Stop();
            replayMs += watch.Elapsed.TotalMilliseconds;
            replayWorstMs = System.Math.Max(replayWorstMs, watch.Elapsed.TotalMilliseconds);
            if (replay.Status == RouteStatus.Searching) return true;
            if (replay.Status != RouteStatus.Found)
            {
                var prefix = replayNavmesh ? navmeshCandidates.CheckedPrefix(replay, Convert.ToV3(pos), Convert.ToV3(NavigationGoal)) : System.Array.Empty<V3>();
                if (prefix.Count > 0)
                {
                    FollowReplay(pos, prefix, "following validated dry prefix before refused navmesh leg");
                    return true;
                }
                RejectReplay(pos, "live route validation failed");
                return true;
            }
            FollowReplay(pos, replay.Path, replayNavmesh ? "following validated dry navmesh route" : "reusing recorded route");
            return true;
        }

        void FollowReplay(Vector3 pos, IReadOnlyList<V3> points, string message)
        {
            path.Set(points);
            path.Reach = .15;
            path.Level = .6;
            replaying = true;
            localRoute = null;
            Tracer.Emit("walker", message, new Dictionary<string, object>
            {
                ["route"] = replayID,
                ["samples"] = replay.Samples,
                ["waypoints"] = points.Count,
                ["maxHintHeightOffset"] = replay.MaxHintHeightOffset,
                ["failure"] = replayProbe.LastFailure?.Stage,
                ["end"] = points[points.Count - 1].ToArray(),
                ["ms"] = System.Math.Round(replayMs, 2),
                ["worstFrameMs"] = System.Math.Round(replayWorstMs, 2),
                ["physicsCasts"] = replayProbe.PhysicsCasts,
            });
            replay = null;
            ResetStuck(pos);
        }

        bool BeginDryNavmesh(Vector3 from, Vector3 target)
        {
            scratch.Clear();
            var found = Pathfinding.instance != null &&
                Pathfinding.instance.GetPath(from, target, scratch, Pathfinding.AgentType.Humanoid, false, true, false);
            if (Time.unscaledTime >= nextDryNavmeshTrace)
            {
                nextDryNavmeshTrace = Time.unscaledTime + 30;
                Tracer.Emit("walker", "dry navmesh query", new Dictionary<string, object>
                {
                    ["found"] = found,
                    ["waypoints"] = scratch.Count,
                    ["from"] = Convert.ToArray(from),
                    ["to"] = Convert.ToArray(target),
                    ["end"] = scratch.Count > 0 ? Convert.ToArray(scratch[scratch.Count - 1]) : null,
                });
            }
            if (!found) return false;
            var probe = new WorldNavigation.PlanningProbe(diagnostics: true, dry: true, refusals: refusals, corridor: hostileClearance, wadeDepth: wadeDepth);
            var candidate = navmeshCandidates.Try(Convert.ToV3(from), Convert.ToV3(target),
                scratch.Select(Convert.ToV3).ToArray(), probe.Probe, NavigationWithin, NavigationLevel);
            if (candidate == null) return false;
            replayProbe = probe; replay = candidate;
            replayNavmesh = true; replayID = 0;
            replayMs = replayWorstMs = 0;
            path.Set(new V3[0]);
            OffLevel = StraightLine = false; Blocked = null;
            Tracer.Emit("walker", "validating dry navmesh candidate", new Dictionary<string, object>
            {
                ["from"] = Convert.ToArray(from),
                ["to"] = Convert.ToArray(target),
                ["waypoints"] = scratch.Count,
                ["end"] = Convert.ToArray(scratch[scratch.Count - 1]),
            });
            return true;
        }

        void BeginLocal(Vector3 from, Vector3 target)
        {
            localWaterBlocked = Blocked;
            Blocked = null;
            localProbe = new WorldNavigation.PlanningProbe(diagnostics: true, dry: dryGround, refusals: refusals, corridor: hostileClearance, diagnosticGoal: Convert.ToV3(target), wadeDepth: wadeDepth);
            localUnderFloorExit = dryGround && avoidHostiles && dryStaging.Available &&
                target.y - from.y > Arrival.Level && WorldNavigation.BuildingAbove(Convert.ToV3(from), target.y);
            ThreatGuidance Guidance()
            {
                if (!dryGround || !avoidHostiles || !preferGuidance && localProbe.HostileRejected == 0) return null;
                var watch = System.Diagnostics.Stopwatch.StartNew();
                var threats = hostiles.Current();
                var result = threats != null && threats.Count > 0 ? new ThreatGuidance(Convert.ToV3(from), Convert.ToV3(target), threats, at =>
                {
                    var height = Ground(at.X, at.Z);
                    return double.IsNaN(height) || (wadeDepth > 0
                        ? Wading.Clear(new V3(at.X, height, at.Z), WorldNavigation.WaterLevel(at), wadeDepth)
                        : height >= WorldNavigation.WaterLevel(at) + DryTravel.Margin);
                }, deferred: true) : null;
                watch.Stop(); localGuidanceMs = watch.Elapsed.TotalMilliseconds;
                return result;
            }
            localGuidanceMs = 0;
            localRoute = new RouteSearch(guidance => new LocalRoute(Convert.ToV3(from), Convert.ToV3(target), localProbe.Probe, NavigationWithin,
                progressHorizon: LocalRoute.FrontierDistance + NavigationWithin, arrivalLevel: NavigationLevel,
                guidance: guidance != null ? guidance.Estimate : (System.Func<V3, double>)null,
                stagingEligible: dryGround && avoidHostiles && dryStaging.Available
                    ? (System.Func<V3, bool>)(at => DryTravel.Clear(at, WorldNavigation.WaterLevel(at), DryStaging.WaterMargin) &&
                        (target.y - from.y <= Arrival.Level || !WorldNavigation.BuildingAbove(at, target.y))) : null,
                heightAware: dryGround, underFloorExit: localUnderFloorExit), Guidance, preferGuidance);
            replay = null; replaying = replayNavmesh = false;
            localStarted = Time.unscaledTime;
            nextNavmeshCheck = localStarted + 1;
            localMs = localWorstMs = 0;
            path.Set(new V3[0]);
        }

        bool PlanLocal(Vector3 pos)
        {
            Driver.Stand();
            stuck.Hold(Time.time);
            // GetPath requests tiles asynchronously. A first query after
            // joining may fail while the usable navmesh is still building.
            // Keep the ongoing search if a retry offers no safe progress.
            if (Time.unscaledTime >= nextNavmeshCheck && Pathfinding.instance != null)
            {
                nextNavmeshCheck = Time.unscaledTime + 1;
                if (dryGround)
                {
                    if (BeginDryNavmesh(pos, NavigationGoal)) return true;
                }
                else
                {
                    scratch.Clear();
                    if (Pathfinding.instance.GetPath(pos, goal, scratch, Pathfinding.AgentType.Humanoid, false, true, false) && scratch.Count > 0)
                    {
                        var points = scratch.Select(Convert.ToV3).ToArray();
                        var from = Convert.ToV3(pos);
                        var target = Convert.ToV3(goal);
                        var complete = Arrival.Reached(points[points.Length - 1], target, 1.5) && WallRoute.Clear(from, points, WorldNavigation.Walls);
                        if (complete || WallRoute.Prefix(from, target, points, WorldNavigation.Walls).Count > 0)
                        {
                            Tracer.Emit("walker", "navmesh became usable during local search");
                            ResetStuck(pos);
                            Repath(pos);
                            return true;
                        }
                    }
                }
            }
            var watch = System.Diagnostics.Stopwatch.StartNew();
            localProbe.BeginSlice();
            // At most eight expansions and roughly 2 ms per physics step.
            for (var i = 0; i < 8 && watch.Elapsed.TotalMilliseconds < 2 && localRoute.Status == RouteStatus.Searching; i++)
            {
                var building = localRoute.BuildingGuidance;
                var before = watch.Elapsed.TotalMilliseconds;
                localRoute.Advance(1);
                if (building) localGuidanceMs += watch.Elapsed.TotalMilliseconds - before;
            }
            watch.Stop();
            localMs += watch.Elapsed.TotalMilliseconds;
            localWorstMs = System.Math.Max(localWorstMs, watch.Elapsed.TotalMilliseconds);
            var elapsed = Time.unscaledTime - localStarted;
            // At eight expansions per physics step the full 4096-node
            // bound already needs over ten seconds, before physical probes
            // consume the two-ms slice budget. Do not discard a still-live
            // entrance search after only half its permitted nodes. Frame
            // work, node count and the enclosing task timeout remain bounded.
            if (localRoute.Guidance?.Active == true) preferGuidance = true;
            IReadOnlyList<V3> stagingPath = null;
            if (dryGround && avoidHostiles && dryStaging.Available && elapsed >= DryStaging.WaitSeconds &&
                localRoute.Status == RouteStatus.Searching && localRoute.StagingPath is IReadOnlyList<V3> candidate &&
                DryTravel.Clear(candidate[candidate.Count - 1], WorldNavigation.WaterLevel(candidate[candidate.Count - 1]), DryStaging.WaterMargin) &&
                (!localUnderFloorExit || !WorldNavigation.BuildingAbove(candidate[candidate.Count - 1], localRoute.Target.Y)) &&
                dryStaging.Take(elapsed, localUnderFloorExit && WorldNavigation.BuildingAbove(Convert.ToV3(pos), localRoute.Target.Y) ||
                    !DryTravel.Clear(Convert.ToV3(pos), WorldNavigation.WaterLevel(Convert.ToV3(pos)), DryStaging.WaterMargin), candidate))
                stagingPath = candidate;
            if (stagingPath == null && localRoute.Status == RouteStatus.Searching && elapsed < 16 * localRoute.BudgetStages) return true;
            // Even an enclosed start may exhaust the local search before
            // the navmesh's asynchronous tile request has had one retry.
            if (localRoute.Status == RouteStatus.NoRoute && elapsed < 1.25f) return true;
            var found = localRoute.Status == RouteStatus.Found || stagingPath != null;
            var selectedPath = stagingPath ?? localRoute.Path;
            Tracer.Emit("walker", stagingPath != null ? "following checked dry staging leg" : found ? "local route found" : "no safe local route", new Dictionary<string, object>
            {
                ["from"] = Convert.ToArray(pos),
                ["to"] = Convert.ToArray(goal),
                ["expanded"] = localRoute.Expanded,
                ["probes"] = localRoute.Probes,
                ["supportHits"] = localProbe.SupportHits,
                ["supportQueries"] = localProbe.SupportQueries,
                ["boundsRejected"] = localProbe.BoundsRejected,
                ["hostileRejected"] = localProbe.HostileRejected,
                ["hostiles"] = !found && avoidHostiles && localProbe.HostileRejected > 0 ? hostiles.Snapshot() : null,
                ["physicsCasts"] = localProbe.PhysicsCasts,
                ["saturated"] = localProbe.Saturated,
                ["failure"] = !found ? localProbe.LastRefusal?.Stage : null,
                ["supportHeights"] = !found ? localProbe.LastRefusal?.SupportHeights : null,
                ["supportQuery"] = !found ? localProbe.LastRefusal?.SupportQuery : null,
                ["failures"] = !found ? localProbe.Failures : null,
                ["searchStatus"] = localRoute.Status.ToString(),
                ["frontier"] = FrontierEvidence(),
                ["nodeLimitReached"] = localRoute.NodeLimitReached,
                ["guidance"] = localRoute.Guidance == null ? null : new
                {
                    active = localRoute.Guidance.Active,
                    waypoints = localRoute.Guidance.Waypoints,
                    rejectedHints = localRoute.Guidance.RejectedHints,
                    hintSamples = localRoute.Guidance.HintSamples,
                    buildMs = System.Math.Round(localGuidanceMs, 2),
                    initialEstimate = localRoute.Guidance.InitialEstimate,
                    phase = localRoute.Phase,
                    complete = localRoute.Guidance.Complete,
                    buildSteps = localRoute.Guidance.BuildSteps,
                },
                ["elapsedS"] = System.Math.Round(elapsed, 2),
                ["ms"] = System.Math.Round(localMs, 2),
                ["worstFrameMs"] = System.Math.Round(localWorstMs, 2),
                ["waypoints"] = found ? selectedPath.Count : 0,
                ["progressLeg"] = localRoute.ProgressLeg || stagingPath != null,
                ["stagingLegs"] = dryStaging.Used,
                ["stagingKind"] = stagingPath != null ? localUnderFloorExit ? "under_floor_exit" : "shore" : null,
                ["end"] = found ? selectedPath[selectedPath.Count - 1].ToArray() : null,
                ["start"] = found ? selectedPath.Take(4).Select(p => p.ToArray()).ToArray() : null,
            });
            if (found)
            {
                path.Set(selectedPath);
                path.Reach = .15;
                path.Level = .6;
                OffLevel = false;
                Blocked = null;
                ResetStuck(pos);
            }
            else if (Player.m_localPlayer?.IsSwimming() == true)
                Blocked = "no safe route from the water; seeking shore";
            else if (entrance.HasValue)
            {
                entrance = null;
                Repath(pos, true);
                return true;
            }
            else navigationFailure = Outcome.Failed(localWaterBlocked != null ? Wire.ReasonDeepWater : Wire.ReasonUnreachable,
                "no supported, water-safe route around the obstacles" + (avoidHostiles ? " and known hostiles" : "") + " within the local search limit");
            localRoute = null;
            return true;
        }

        /// <summary>Stamina below this share, a swimming bot makes for shore.</summary>
        const float SwimLow = 0.3f;

        /// <summary>Seconds without getting nearer before a shore is given up.</summary>
        const float ShoreStall = 4f;

        V3? shore;
        readonly List<V3> triedShores = new List<V3>();
        double shoreBest;
        float shoreGainAt, noShoreAt = float.MinValue, dryAt;
        bool swimming;

        /// <summary>A swimming bot walks: running drains the stamina it
        /// needs to stay afloat, and none comes back in the water. With
        /// little left it makes for the nearest shore, whatever the goal, and
        /// holds the stuck count meanwhile; a shore it gets no nearer to, or
        /// reaches still swimming, it gives up for another. True while it is
        /// doing that. Out of the water it plans the way to the goal again,
        /// since the line it swam out on may cross back.</summary>
        bool Swim(Vector3 pos, ref bool run)
        {
            if (wadeDepth > 0) return false;
            var p = Player.m_localPlayer;
            var wet = p != null && p.IsSwimming();
            // A thin planning buffer is not actual immersion. While still
            // dry, let the normal planner find an uphill departure: every
            // midpoint, endpoint and live step retains TravelMargin. Seeking
            // a geometric shore instead can discard that checked route and
            // deadlock on its own first step under the same water rule.
            if (dryGround && p != null) wet |= p.InWater();
            if (p == null || !wet)
            {
                shore = null;
                if (swimming && p != null)
                {
                    swimming = false;
                    dryAt = Time.time;
                    Repath(pos);
                }
                // A moment ashore at the foot of a bank it then slides off
                // does not make the bank worth another try.
                if (Time.time - dryAt > ShoreStall) triedShores.Clear();
                return false;
            }
            swimming = true;
            run = false;
            if (dryGround) Blocked = "dry-ground travel must first get ashore";
            if (shore is V3 aim)
            {
                var d = V3.FlatDistance(Convert.ToV3(pos), aim);
                if (d < shoreBest - 0.5)
                {
                    shoreBest = d;
                    shoreGainAt = Time.time;
                }
                else if (Time.time - shoreGainAt > ShoreStall || d < 1.5)
                {
                    Tracer.Emit("walker", "swimming: cannot get ashore there; trying another", new Dictionary<string, object>
                    {
                        ["pos"] = Convert.ToArray(pos),
                        ["shore"] = aim.ToArray(),
                    });
                    triedShores.Add(aim);
                    shore = null;
                }
            }
            // Low on stamina, or refused the way on: ashore first.
            if (shore == null && Blocked == null && p.GetStamina() >= SwimLow * p.GetMaxStamina()) return false;
            if (shore == null)
            {
                // A failed search is not run again every frame.
                if (Time.time - noShoreAt < 2f) return false;
                shore = DeepWater.Shore(Convert.ToV3(pos), Ground,
                    dryGround ? WorldNavigation.WaterLevel(Convert.ToV3(pos)) : WaterLevel, tried: triedShores,
                    maxDepth: dryGround ? -DryTravel.TravelMargin : .3);
                var why = Blocked != null ? "swimming, the way on is refused" : "swimming, low on stamina";
                Tracer.Emit("walker", shore != null ? why + ": making for shore" : why + ", and no shore in reach",
                    new Dictionary<string, object>
                    {
                        ["stamina"] = System.Math.Round(p.GetStamina(), 1),
                        ["pos"] = Convert.ToArray(pos),
                        ["shore"] = shore?.ToArray(),
                    });
                if (shore == null)
                {
                    noShoreAt = Time.time;
                    return false;
                }
                shoreBest = V3.FlatDistance(Convert.ToV3(pos), shore.Value);
                shoreGainAt = Time.time;
            }
            stuck.Hold(Time.time);
            var dir = Convert.ToVector3(shore.Value) - pos;
            if (dryGround)
            {
                dryProbe.BeginSlice();
                var step = DryTravel.ShoreStep(Convert.ToV3(pos), shore.Value, dryProbe.Probe);
                if (!step.HasValue)
                {
                    if (Time.unscaledTime >= nextDryRefusalTrace)
                    {
                        nextDryRefusalTrace = Time.unscaledTime + 1;
                        Tracer.Emit("walker", "dry shore step refused", new Dictionary<string, object>
                        {
                            ["from"] = Convert.ToArray(pos),
                            ["shore"] = shore.Value.ToArray(),
                            ["failure"] = dryProbe.LastFailure?.Stage,
                            ["inWater"] = p.InWater(),
                            ["swimming"] = p.IsSwimming(),
                            ["grounded"] = p.IsOnGround(),
                            ["waterLevel"] = WorldNavigation.WaterLevel(Convert.ToV3(pos)),
                        });
                    }
                    navigationFailure = Outcome.Failed(Wire.ReasonUnreachable,
                        "no checked dry shore step: " + (dryProbe.LastFailure?.Stage ?? "unsupported"));
                    swimming = false;
                    Driver.Stand();
                    return true;
                }
                dir = Convert.ToVector3(step.Value) - pos;
            }
            dir.y = 0;
            Driver.Move = dir.sqrMagnitude > 0.0001f ? dir.normalized : Vector3.zero;
            Driver.Run = false;
            return true;
        }

        /// <summary>The flat direction to the next waypoint, or the goal.</summary>
        Vector3 Toward(Vector3 pos)
        {
            return Convert.ToVector3(path.Direction(Convert.ToV3(pos)));
        }

        /// <summary>One step of a climb: wait for stamina, then jump a few
        /// times pushing on. The stuck count holds meanwhile. False once the
        /// climb is over, and the walker steers as usual.</summary>
        bool Climb(Vector3 pos)
        {
            var p = Player.m_localPlayer;
            if (p == null) return false;
            // Out of the stuck circle: free, and walking on is what is left.
            if (V3.FlatDistance(Convert.ToV3(pos), stuck.Center) > stuck.Radius)
            {
                climb.Stop();
                climbWaiting = false;
                Tracer.Emit("walker", "climb: free", new Dictionary<string, object> { ["pos"] = Convert.ToArray(pos) });
                return false;
            }
            stuck.Hold(Time.time);
            // What a jump costs before gear and effects, which is near enough
            // to know how long to wait.
            var cost = p.m_jumpStaminaUsage * Game.m_moveStaminaRate;
            var wasWaiting = climbWaiting;
            var step = climb.Tick(Time.time, p.GetStamina(), p.GetMaxStamina(), cost, !p.IsEncumbered());
            climbWaiting = step == ClimbStep.Wait;
            if (step == ClimbStep.Done)
            {
                Tracer.Emit("walker", "climb done", new Dictionary<string, object> { ["pos"] = Convert.ToArray(pos) });
                return false;
            }
            if (step == ClimbStep.Wait)
            {
                if (!wasWaiting)
                    Tracer.Emit("walker", "climb: waiting for stamina", new Dictionary<string, object>
                    {
                        ["stamina"] = System.Math.Round(p.GetStamina(), 1),
                        ["needed"] = System.Math.Round(climb.Needed(cost, p.GetMaxStamina()), 1),
                        ["pos"] = Convert.ToArray(pos),
                    });
                // Standing still: stamina comes back faster than walking into a wall.
                Driver.Move = Vector3.zero;
                Driver.Run = false;
                return true;
            }
            if (step == ClimbStep.Jump) Driver.Jump = true;
            var dir = Toward(pos);
            Driver.Move = dir.sqrMagnitude > 0.0001f ? dir.normalized : Vector3.zero;
            // Running costs the stamina the jumps need.
            Driver.Run = false;
            return true;
        }

        bool climbWaiting;

        public int RecoveryAttempts => stuck.Attempts;

        /// <summary>The failure for a bot the game will not move: over its
        /// carry weight with no stamina left. Stamina does not come back
        /// while it is over, so only carrying less frees it, and stuck
        /// recovery would only burn the time. Null when it can move.</summary>
        public static Outcome Stranded(Player p) =>
            p.IsEncumbered() && !p.HaveStamina()
                ? Outcome.Failed(Wire.ReasonEncumbered,
                    $"carrying {p.GetInventory().GetTotalWeight():0} of {p.GetMaxCarryWeight():0} with no stamina left")
                : null;
    }

    /// <summary>move_to: walk to a point or an entity and stop within stopDistance.</summary>
    sealed class MoveToTask : IBodyTask
    {
        readonly MoveToArgs args;
        readonly GameObject entity;
        readonly Walker walker;
        readonly BoundedTravel survey;
        readonly TravelEncounter encounter;
        Vector3 goal;

        public MoveToTask(MoveToArgs args, Vector3 goal, GameObject entity, string id = null)
        {
            this.args = args;
            this.goal = goal;
            this.entity = entity;
            if (args.RecoverThreats) encounter = new TravelEncounter(id, args.DefendThreats);
            var p = Player.m_localPlayer;
            if (args.Scope != null) survey = new BoundedTravel(args.Scope, Convert.ToV3(p.transform.position), Time.unscaledTime);
            walker = new Walker(survey == null ? (System.Func<V3, V3, bool>)null : survey.Clear);
            walker.SetGoal(p.transform.position, goal, args.StopDistance, reuseRoutes: args.To.Entity == null, dryGround: args.Dry, avoidHostiles: args.AvoidHostiles, wade: args.Wade);
            walker.ResetStuck(p.transform.position);
        }

        public Outcome Tick(double dt)
        {
            var p = Player.m_localPlayer;
            var at = p != null ? Convert.ToV3(p.transform.position) : V3.Zero;
            var refusal = survey?.Observe(at, Time.unscaledTime);
            if (refusal != null)
            {
                Driver.Stand();
                return Outcome.Failed(Wire.ReasonUnreachable, survey.Receipt(at, refusal));
            }
            var ended = Step(dt);
            if (ended != null && survey != null) ended.Detail = survey.Receipt(at, ended.Detail);
            if (ended != null) encounter?.Cancel();
            return ended;
        }

        Outcome Step(double dt)
        {
            var p = Player.m_localPlayer;
            if (p == null) return Outcome.Failed(Wire.ReasonStopped, "no player");
            if (args.To.Entity != null)
            {
                if (entity == null) return Outcome.Failed(Wire.ReasonTargetLost, "the entity is gone");
                if ((entity.transform.position - goal).sqrMagnitude > 4)
                {
                    goal = entity.transform.position;
                    walker.SetGoal(p.transform.position, goal, args.StopDistance, reuseRoutes: false, dryGround: args.Dry, avoidHostiles: args.AvoidHostiles, wade: args.Wade);
                }
            }
            var pos = p.transform.position;
            if (walker.WadingRefusal(pos) is Outcome wadingRefusal) { Driver.Stand(); return wadingRefusal; }
            if (encounter != null)
            {
                if (encounter.Tick(p, dt, out var recoveryFailure)) return recoveryFailure;
                if (encounter.Resumed)
                {
                    walker.SetGoal(pos, goal, args.StopDistance, reuseRoutes: false, dryGround: args.Dry, avoidHostiles: args.AvoidHostiles, wade: args.Wade);
                    walker.ResetStuck(pos);
                }
            }
            if (Arrival.Reached(Convert.ToV3(pos), Convert.ToV3(goal), args.StopDistance))
            {
                Driver.Stand();
                if (args.To.Entity == null) walker.Arrived(pos);
                return Outcome.Succeeded();
            }
            if (walker.Unreachable(pos, args.StopDistance) is Outcome under)
            {
                Driver.Stand();
                return under;
            }
            if (Walker.Stranded(p) is Outcome stranded)
            {
                Driver.Stand();
                return stranded;
            }
            if (walker.Refused() is Outcome wet)
            {
                Driver.Stand();
                return encounter != null && (wet.Reason == Wire.ReasonUnreachable || wet.Reason == Wire.ReasonDeepWater)
                    ? encounter.Blocked(p, wet) : wet;
            }
            if (!walker.Steer(pos, args.Run, climbObstacles: !args.Dry))
            {
                Driver.Stand();
                return Outcome.Failed(Wire.ReasonStuck,
                    $"no progress after {walker.RecoveryAttempts} recoveries{(walker.StraightLine ? " (no navmesh path)" : "")}");
            }
            return null;
        }

        public void Cancel() { encounter?.Cancel(); Driver.Stand(); }
    }

    /// <summary>
    /// pickup: walk to a drop and take it. The game's auto-pickup takes
    /// whatever is within about 2 m of the bot, as for any player, so the
    /// walk is usually all it takes; if the drop is still there once the bot
    /// stands on it, the task picks it up itself. It succeeds when the bot
    /// carries more of the item than it did, however that came about.
    /// </summary>
    sealed class PickupTask : IRecoverableWork
    {
        /// <summary>How close counts as standing on the drop.</summary>
        const float Close = (float)PickupTracking.PickupDistance;

        /// <summary>How long to leave auto-pickup to it before picking up by hand.</summary>
        const float AutoFor = 1.0f;

        /// <summary>How many times to pick it up by hand before giving up on
        /// a drop the game will not hand over (someone else holds it).</summary>
        const int Tries = 3;

        /// <summary>How far from where the drop was to look for the stack the
        /// game merged it into.</summary>
        const float MergedWithin = 2.5f;

        ItemDrop drop;
        readonly string prefab;
        readonly bool safeTravel;
        readonly bool allowMerge;
        readonly int before;
        int taken;
        readonly Walker walker = new Walker();
        Vector3 goal;
        float arrivedAt = -1;
        float ownershipAt = -1;
        int tries;
        WorkCheckpoint checkpoint;

        public PickupTask(ItemDrop drop, bool safeTravel = false, bool allowMerge = true)
        {
            this.drop = drop;
            this.safeTravel = safeTravel;
            this.allowMerge = allowMerge;
            prefab = drop.m_itemData.m_dropPrefab.name;
            var p = Player.m_localPlayer;
            before = Carried(p, prefab);
            goal = drop.transform.position;
            walker.SetGoal(p.transform.position, goal, within: Close, dryGround: safeTravel, avoidHostiles: safeTravel);
            walker.ResetStuck(p.transform.position);
        }

        WorkCheckpoint ObserveWork()
        {
            var p = Player.m_localPlayer;
            if (drop == null || p == null || drop.m_nview == null || !drop.m_nview.IsValid() || drop.m_itemData?.m_dropPrefab == null)
                return new WorkCheckpoint(null, null, 0, 0, false);
            return new WorkCheckpoint(Perceiver.Id(drop.m_nview), drop.m_itemData.m_dropPrefab.name,
                drop.m_itemData.m_stack, Carried(p, prefab), drop.gameObject.activeInHierarchy);
        }

        public bool Checkpoint() { checkpoint = ObserveWork(); return tries == 0; }
        public Outcome Resume()
        {
            if (checkpoint == null || !checkpoint.Matches(ObserveWork()))
                return Outcome.Failed(Wire.ReasonStopped, "pickup source or inventory changed during encounter; not replayed");
            goal = drop.transform.position;
            var pos = Player.m_localPlayer.transform.position;
            walker.SetGoal(pos, goal, within: Close, dryGround: safeTravel, avoidHostiles: safeTravel);
            walker.ResetStuck(pos);
            arrivedAt = ownershipAt = -1;
            return null;
        }

        public Outcome Tick(double dt)
        {
            var p = Player.m_localPlayer;
            if (p == null) return Outcome.Failed(Wire.ReasonStopped, "no player");
            // The most it has gained: using some on the way does not hide it.
            taken = System.Math.Max(taken, Carried(p, prefab) - before);
            var took = taken;
            if (allowMerge && drop == null && took == 0 && Merged() is ItemDrop merged)
            {
                // The game merges nearby drops of an item into one stack.
                drop = merged;
                goal = drop.transform.position;
                walker.SetGoal(p.transform.position, goal, within: Close, dryGround: safeTravel, avoidHostiles: safeTravel);
            }
            if (drop == null)
            {
                // Gone: into the bot's pack, or someone else's, or despawned.
                Driver.Stand();
                return took > 0
                    ? Outcome.Succeeded($"took {took} {prefab}")
                    : Outcome.Failed(Wire.ReasonTargetLost, $"the {prefab} is gone");
            }
            if (PickupTracking.Retarget(Convert.ToV3(goal), Convert.ToV3(drop.transform.position)))
            {
                // Auto-pickup pulls drops toward the player, and they roll.
                goal = drop.transform.position;
                walker.SetGoal(p.transform.position, goal, within: Close, dryGround: safeTravel, avoidHostiles: safeTravel);
            }
            var pos = p.transform.position;
            var reach = p.m_maxInteractDistance;
            // A normal visible hand pickup need not walk onto a product
            // beside a fire or inside an obstacle's navigation clearance.
            var byHand = float.IsFinite(reach) && reach > 0 &&
                PickupTracking.Reached(Convert.ToV3(pos), Convert.ToV3(drop.transform.position), reach) &&
                UseReach.Near(p, drop.gameObject, reach);
            if (byHand || PickupTracking.Reached(Convert.ToV3(pos), Convert.ToV3(drop.transform.position), Close))
            {
                Driver.Stand();
                // Standing still on purpose is not being stuck.
                walker.ResetStuck(pos);
                if (arrivedAt < 0) arrivedAt = Time.time;
                if (Time.time - arrivedAt < AutoFor) return null;
                if (!p.GetInventory().CanAddItem(drop.m_itemData, 1))
                {
                    // Full: what it took is still a pickup, the rest stays.
                    if (took > 0)
                        return Outcome.Succeeded($"took {took} {prefab}; no room for the other {drop.m_itemData.m_stack}");
                    return Outcome.Failed(Wire.ReasonInventoryFull, $"no room for {prefab}");
                }
                // The game's auto-pickup will not take what would overload the
                // bot, but Pickup by hand would.
                if (TooHeavy(p, drop.m_itemData))
                {
                    if (took > 0)
                        return Outcome.Succeeded($"took {took} {prefab}; the other {drop.m_itemData.m_stack} is too heavy to carry");
                    return Outcome.Failed(Wire.ReasonTooHeavy, $"{prefab} would take it past its carry weight");
                }
                if (!drop.CanPickup(false))
                {
                    if (ownershipAt < 0)
                    {
                        ownershipAt = Time.time;
                        Tracer.Emit("pickup", "requesting normal drop ownership", new Dictionary<string, object>
                        { ["entity"] = Perceiver.Id(drop.m_nview), ["prefab"] = prefab });
                    }
                    if (Time.time - ownershipAt > 5)
                        return Outcome.Failed(Wire.ReasonTimeout, "the drop owner did not hand over the item");
                    // Use the owner's normal RPC grant. Unlike ItemDrop.Pickup,
                    // this installs no deferred pickup callback after cancellation.
                    drop.RequestOwn();
                    return null;
                }
                if (ownershipAt >= 0)
                {
                    ownershipAt = -1;
                    Tracer.Emit("pickup", "normal drop ownership granted", new Dictionary<string, object>
                    { ["entity"] = Perceiver.Id(drop.m_nview), ["prefab"] = prefab });
                }
                if (tries++ >= Tries)
                    return took > 0
                        ? Outcome.Succeeded($"took {took} {prefab}; the game kept the other {drop.m_itemData.m_stack}")
                        : Outcome.Failed(Wire.ReasonInvalidTarget, $"the game will not hand over the {prefab}");
                p.Pickup(drop.gameObject, false, false);
                arrivedAt = Time.time; // give it a moment to land before trying again
                return null;
            }
            arrivedAt = -1;
            // Within the game's auto-pickup range it may yet pull the drop in.
            if (Vector3.Distance(pos, goal) > p.m_autoPickupRange && walker.Unreachable(pos, Close) is Outcome under)
            {
                Driver.Stand();
                return took > 0 ? Outcome.Succeeded($"took {took} {prefab}; the other {drop.m_itemData.m_stack} is out of reach") : under;
            }
            if (Walker.Stranded(p) is Outcome stranded)
            {
                Driver.Stand();
                return stranded;
            }
            if (walker.Refused() is Outcome wet)
            {
                Driver.Stand();
                return wet;
            }
            if (!walker.Steer(pos, true, climbObstacles: !safeTravel))
            {
                Driver.Stand();
                return Outcome.Failed(Wire.ReasonStuck,
                    $"no progress after {walker.RecoveryAttempts} recoveries{(walker.StraightLine ? " (no navmesh path)" : "")}");
            }
            return null;
        }

        public void Cancel() => Driver.Stand();

        /// <summary>A drop of the same item near where the target was, which
        /// the game may have merged it into.</summary>
        ItemDrop Merged()
        {
            ItemDrop best = null;
            var bestDistance = MergedWithin;
            foreach (var d in ItemDrop.s_instances)
            {
                if (d == null || d.m_itemData?.m_dropPrefab == null || d.m_itemData.m_dropPrefab.name != prefab) continue;
                var dist = Vector3.Distance(d.transform.position, goal);
                if (dist <= bestDistance) (best, bestDistance) = (d, dist);
            }
            return best;
        }

        /// <summary>Taking all of the drop would carry the bot past its
        /// carry weight, as the game's auto-pickup judges it.</summary>
        public static bool TooHeavy(Player p, ItemDrop.ItemData item) =>
            item.GetWeight() + p.GetInventory().GetTotalWeight() > p.GetMaxCarryWeight();

        static int Carried(Player p, string prefab)
        {
            var n = 0;
            foreach (var item in p.GetInventory().GetAllItems())
                if (item.m_dropPrefab != null && item.m_dropPrefab.name == prefab) n += item.m_stack;
            return n;
        }
    }

    /// <summary>follow: keep a player at a distance, at the leader's pace.</summary>
    sealed class FollowTask : IBodyTask
    {
        /// <summary>How long the leader may be out of sight before the task fails.</summary>
        const float LoseAfter = 5f;

        /// <summary>How far the leader moves before the path is worked out again.</summary>
        const float RepathAfter = 2f;

        readonly FollowArgs args;
        readonly bool safeTravel;
        readonly FollowPace pace = new FollowPace();
        NearLeaderWait stall = new NearLeaderWait();
        readonly Walker walker = new Walker();
        FollowPortalPassage portal;
        Vector3 pathedTo;
        float lastSeen;
        bool moving;
        Vector3 leaderLast;
        float leaderSpeed;
        // lastGait is the gait last traced, or Lost or Unset.
        const Gait Unset = (Gait)(-1), Lost = (Gait)(-2);
        Gait lastGait = Unset;

        /// <summary>Faster than a jog (4 m/s) means the leader is sprinting.</summary>
        const float RunningSpeed = 5.5f;

        public FollowTask(FollowArgs args, bool safeTravel = false)
        {
            this.args = args;
            this.safeTravel = safeTravel;
            lastSeen = Time.time;
            pathedTo = new Vector3(float.MaxValue, 0, 0);
        }

        public Outcome Tick(double dt)
        {
            var p = Player.m_localPlayer;
            if (p == null) return Outcome.Failed(Wire.ReasonStopped, "no player");
            var leader = Players.Find(args.Player);
            if (portal == null && (leader == null || !leader.IsDead()) && Players.Departure(args.Player, leader) is FollowPortalLink departure)
                portal = new FollowPortalPassage(departure);
            // Also observe a native trip already underway, without admitting
            // any trigger. It uses the same bounded teleport/arrival policy.
            if (portal == null && p.IsTeleporting()) portal = new FollowPortalPassage(null);
            if (portal != null)
            {
                var ended = portal.Tick(p, leader);
                if (ended != null || !portal.Done) return ended;
                portal = null;
                moving = false;
                stall = new NearLeaderWait();
                leaderLast = Vector3.zero;
                leaderSpeed = 0;
                lastGait = Unset;
                pathedTo = new Vector3(float.MaxValue, 0, 0);
                lastSeen = Time.time;
            }
            if (leader == null || leader.IsDead())
            {
                if (lastGait != Lost)
                {
                    Tracer.Emit("follow", leader == null ? $"{args.Player} out of sight" : $"{args.Player} is dead");
                    lastGait = Lost;
                }
                Driver.Stand();
                // Not steering: when the leader is back, start the walk
                // afresh, with no half-done climb or stuck count.
                moving = false;
                return Time.time - lastSeen > LoseAfter
                    ? Outcome.Failed(Wire.ReasonTargetLost, $"{args.Player} is gone")
                    : null;
            }
            lastSeen = Time.time;
            var pos = p.transform.position;
            var lead = leader.transform.position;
            var gap = (float)V3.FlatDistance(Convert.ToV3(pos), Convert.ToV3(lead));
            // A remote player's IsRunning is not synced to us, so their pace
            // is measured from how far they moved, smoothed over a few steps.
            if (leaderLast != Vector3.zero && dt > 0)
            {
                var flat = lead - leaderLast;
                flat.y = 0;
                leaderSpeed = Mathf.Lerp(leaderSpeed, flat.magnitude / (float)dt, 0.1f);
            }
            leaderLast = lead;
            if (stall.Waiting)
            {
                if (stall.Hold(Convert.ToV3(pos), Convert.ToV3(lead)))
                {
                    Driver.Stand();
                    Driver.LookAt = leader.GetHeadPoint();
                    Driver.LookUntil = Time.time + .5f;
                    return null;
                }
                Trace("resume", "leader moved on or out of reach", gap, pos, lead);
                // Start the walk afresh: a new path and a full set of recoveries.
                moving = false;
                pathedTo = new Vector3(float.MaxValue, 0, 0);
                lastGait = Unset;
            }
            var nearWall = gap <= args.Distance + pace.Slack && System.Math.Abs(pos.y - lead.y) <= Arrival.Level &&
                           !WallRoute.Clear(Convert.ToV3(pos), Convert.ToV3(lead), WorldNavigation.Walls);
            var mayStop = !safeTravel || System.Math.Abs(pos.y - lead.y) <= Arrival.Level &&
                WallRoute.Clear(Convert.ToV3(pos), Convert.ToV3(lead), WorldNavigation.Walls);
            var gait = pace.Decide(gap, args.Distance, args.Run, leaderSpeed > RunningSpeed, nearWall, mayStop);
            if (gait != lastGait)
            {
                Tracer.Emit("follow", gait.ToString().ToLowerInvariant(), new Dictionary<string, object>
                {
                    ["gap"] = System.Math.Round(gap, 2),
                    ["distance"] = args.Distance,
                    ["pos"] = Convert.ToArray(pos),
                    ["leader"] = Convert.ToArray(lead),
                    ["leaderSpeed"] = System.Math.Round(leaderSpeed, 2),
                });
                lastGait = gait;
            }
            if (gait == Gait.Stop)
            {
                if (moving) Driver.Stand();
                moving = false;
                return null;
            }
            if (!moving)
            {
                walker.ResetStuck(pos);
                moving = true;
            }
            if ((lead - pathedTo).sqrMagnitude > RepathAfter * RepathAfter)
            {
                walker.SetGoal(pos, lead, reuseRoutes: safeTravel, dryGround: safeTravel, avoidHostiles: safeTravel);
                pathedTo = lead;
            }
            if (Walker.Stranded(p) is Outcome stranded)
            {
                Driver.Stand();
                return stranded;
            }
            if (walker.Refused() is Outcome wet)
            {
                Driver.Stand();
                return wet;
            }
            if (!walker.Steer(pos, gait == Gait.Run))
            {
                Driver.Stand();
                // Near the leader but unable to get closer, as outside the
                // wall of the house they stand in: wait for them rather than
                // fail and walk home.
                if (stall.GiveUp(Convert.ToV3(pos), Convert.ToV3(lead), args.Distance))
                {
                    Trace("wait", $"cannot get closer after {walker.RecoveryAttempts} recoveries", gap, pos, lead);
                    moving = false;
                    return null;
                }
                return Outcome.Failed(Wire.ReasonStuck, $"no progress after {walker.RecoveryAttempts} recoveries");
            }
            return null;
        }

        void Trace(string msg, string reason, float gap, Vector3 pos, Vector3 lead) =>
            Tracer.Emit("follow", msg, new Dictionary<string, object>
            {
                ["reason"] = reason,
                ["gap"] = System.Math.Round(gap, 2),
                ["distance"] = args.Distance,
                ["pos"] = Convert.ToArray(pos),
                ["leader"] = Convert.ToArray(lead),
            });

        public void Cancel() { portal?.Cancel(); Driver.Stand(); }
    }

    static partial class Players
    {
        /// <summary>A player in range by name (case-insensitive), or by the
        /// id the chat stream reports.</summary>
        public static Player Find(string nameOrId)
        {
            foreach (var pl in Player.GetAllPlayers())
            {
                if (pl == null || pl == Player.m_localPlayer) continue;
                if (string.Equals(pl.GetPlayerName(), nameOrId, System.StringComparison.OrdinalIgnoreCase)) return pl;
                if (pl.GetZDOID().UserID.ToString() == nameOrId || pl.GetPlayerID().ToString() == nameOrId) return pl;
            }
            return null;
        }
    }
}
