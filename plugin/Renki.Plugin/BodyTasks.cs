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
        readonly List<Vector3> scratch = new List<Vector3>();
        Vector3 goal;
        bool hasGoal;
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
            Blocked != null && Player.m_localPlayer?.IsSwimming() != true
                ? Outcome.Failed(Wire.ReasonDeepWater, Blocked) : null;

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

        /// <summary>The failure for a bot that has gone as far toward a goal
        /// as the navmesh reaches, under or beside it but off its level, or
        /// null while it can still get there.</summary>
        public Outcome Unreachable(Vector3 pos, double within)
        {
            var here = Convert.ToV3(pos);
            if (!OffLevel || (V3.FlatDistance(here, Convert.ToV3(goal)) > within &&
                              V3.FlatDistance(here, Convert.ToV3(end)) > AtEnd)) return null;
            var dy = goal.y - pos.y;
            return Outcome.Failed(Wire.ReasonUnreachable,
                $"the goal is {System.Math.Abs(dy):0.0} m {(dy > 0 ? "above" : "below")} and the navmesh does not reach it");
        }

        public void SetGoal(Vector3 from, Vector3 to)
        {
            goal = to;
            hasGoal = true;
            Repath(from);
        }

        void Repath(Vector3 from)
        {
            scratch.Clear();
            var pf = Pathfinding.instance;
            StraightLine = !(pf != null &&
                             pf.GetPath(from, goal, scratch, Pathfinding.AgentType.Humanoid, false, true, false) &&
                             scratch.Count > 0);
            Blocked = null;
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
            var shortBy = Vector3.Distance(end, goal);
            // Off level only when the end is near the goal across the ground:
            // a path that stops short farther off may yet go on, through a
            // door or after a re-path.
            OffLevel = !StraightLine && System.Math.Abs(end.y - goal.y) > Arrival.Level &&
                       V3.FlatDistance(Convert.ToV3(end), Convert.ToV3(goal)) <= NearGoal;
            var data = new Dictionary<string, object>
            {
                ["from"] = Convert.ToArray(from),
                ["to"] = Convert.ToArray(goal),
                ["waypoints"] = scratch.Count,
            };
            if (shortBy > 1) data["end"] = Convert.ToArray(end);
            Tracer.Emit("walker", StraightLine ? "no navmesh path; straight line"
                : shortBy > 1 ? $"path stops {shortBy:0.0} m short" : "path", data);
        }

        public void ResetStuck(Vector3 pos)
        {
            stuck.Reset(Convert.ToV3(pos), Time.time);
            climb.Stop();
            climbWaiting = false;
        }

        /// <summary>Steers toward the goal for this step. Returns false when stuck recovery gave up.</summary>
        public bool Steer(Vector3 pos, bool run)
        {
            if (!hasGoal) return true;
            if (Swim(pos, ref run)) return true;
            if (climb.Active && Climb(pos)) return true;
            var recovery = stuck.Update(Convert.ToV3(pos), Time.time);
            if (recovery != Recovery.None)
                Tracer.Emit("walker", recovery == Recovery.GiveUp ? "stuck; giving up" : $"stuck; trying {recovery}",
                    new Dictionary<string, object>
                    {
                        ["attempt"] = stuck.Attempts,
                        ["pos"] = Convert.ToArray(pos),
                        ["center"] = stuck.Center.ToArray(),
                        ["goal"] = Convert.ToArray(goal),
                    });
            // A closed door beside a stuck bot is the likeliest wall: open it
            // in place of this round's jump, strafe or re-path.
            if (recovery != Recovery.None && recovery != Recovery.GiveUp && doors.Stuck(Player.m_localPlayer, pos))
                recovery = Recovery.None;
            switch (recovery)
            {
                case Recovery.Jump:
                    Driver.Jump = true;
                    break;
                case Recovery.Strafe:
                    strafeUntil = Time.time + 1f;
                    strafeSign = -strafeSign;
                    break;
                case Recovery.Repath:
                    Repath(pos);
                    break;
                case Recovery.Climb:
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
            // A strafe walks, so it cannot carry the bot out of the stuck
            // circle on its own and pass for progress.
            Driver.Run = run && Time.time >= strafeUntil;
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
            var p = Player.m_localPlayer;
            if (p == null || !p.IsSwimming())
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
                shore = DeepWater.Shore(Convert.ToV3(pos), Ground, WaterLevel, tried: triedShores);
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
            dir.y = 0;
            Driver.Move = dir.sqrMagnitude > 0.0001f ? dir.normalized : Vector3.zero;
            Driver.Run = false;
            return true;
        }

        /// <summary>The flat direction to the next waypoint, or the goal.</summary>
        Vector3 Toward(Vector3 pos)
        {
            var next = path.Next(Convert.ToV3(pos));
            var target = next.HasValue ? Convert.ToVector3(next.Value) : goal;
            var dir = target - pos;
            dir.y = 0;
            return dir;
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
        readonly Walker walker = new Walker();
        Vector3 goal;

        public MoveToTask(MoveToArgs args, Vector3 goal, GameObject entity)
        {
            this.args = args;
            this.goal = goal;
            this.entity = entity;
            var p = Player.m_localPlayer;
            walker.SetGoal(p.transform.position, goal);
            walker.ResetStuck(p.transform.position);
        }

        public Outcome Tick(double dt)
        {
            var p = Player.m_localPlayer;
            if (p == null) return Outcome.Failed(Wire.ReasonStopped, "no player");
            if (args.To.Entity != null)
            {
                if (entity == null) return Outcome.Failed(Wire.ReasonTargetLost, "the entity is gone");
                if ((entity.transform.position - goal).sqrMagnitude > 4)
                {
                    goal = entity.transform.position;
                    walker.SetGoal(p.transform.position, goal);
                }
            }
            var pos = p.transform.position;
            if (Arrival.Reached(Convert.ToV3(pos), Convert.ToV3(goal), args.StopDistance))
            {
                Driver.Stand();
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
                return wet;
            }
            if (!walker.Steer(pos, args.Run))
            {
                Driver.Stand();
                return Outcome.Failed(Wire.ReasonStuck,
                    $"no progress after {walker.RecoveryAttempts} recoveries{(walker.StraightLine ? " (no navmesh path)" : "")}");
            }
            return null;
        }

        public void Cancel() => Driver.Stand();
    }

    /// <summary>
    /// pickup: walk to a drop and take it. The game's auto-pickup takes
    /// whatever is within about 2 m of the bot, as for any player, so the
    /// walk is usually all it takes; if the drop is still there once the bot
    /// stands on it, the task picks it up itself. It succeeds when the bot
    /// carries more of the item than it did, however that came about.
    /// </summary>
    sealed class PickupTask : IBodyTask
    {
        /// <summary>How close counts as standing on the drop.</summary>
        const float Close = 1.0f;

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
        readonly int before;
        int taken;
        readonly Walker walker = new Walker();
        Vector3 goal;
        float arrivedAt = -1;
        int tries;

        public PickupTask(ItemDrop drop)
        {
            this.drop = drop;
            prefab = drop.m_itemData.m_dropPrefab.name;
            var p = Player.m_localPlayer;
            before = Carried(p, prefab);
            goal = drop.transform.position;
            walker.SetGoal(p.transform.position, goal);
            walker.ResetStuck(p.transform.position);
        }

        public Outcome Tick(double dt)
        {
            var p = Player.m_localPlayer;
            if (p == null) return Outcome.Failed(Wire.ReasonStopped, "no player");
            // The most it has gained: using some on the way does not hide it.
            taken = System.Math.Max(taken, Carried(p, prefab) - before);
            var took = taken;
            if (drop == null && took == 0 && Merged() is ItemDrop merged)
            {
                // The game merges nearby drops of an item into one stack.
                drop = merged;
                goal = drop.transform.position;
                walker.SetGoal(p.transform.position, goal);
            }
            if (drop == null)
            {
                // Gone: into the bot's pack, or someone else's, or despawned.
                Driver.Stand();
                return took > 0
                    ? Outcome.Succeeded($"took {took} {prefab}")
                    : Outcome.Failed(Wire.ReasonTargetLost, $"the {prefab} is gone");
            }
            if ((drop.transform.position - goal).sqrMagnitude > 1)
            {
                // Auto-pickup pulls drops toward the player, and they roll.
                goal = drop.transform.position;
                walker.SetGoal(p.transform.position, goal);
            }
            var pos = p.transform.position;
            if (Arrival.Reached(Convert.ToV3(pos), Convert.ToV3(goal), Close))
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
            if (!walker.Steer(pos, true))
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
        readonly FollowPace pace = new FollowPace();
        readonly Walker walker = new Walker();
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

        public FollowTask(FollowArgs args)
        {
            this.args = args;
            lastSeen = Time.time;
            pathedTo = new Vector3(float.MaxValue, 0, 0);
        }

        public Outcome Tick(double dt)
        {
            var p = Player.m_localPlayer;
            if (p == null) return Outcome.Failed(Wire.ReasonStopped, "no player");
            var leader = Players.Find(args.Player);
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
            var gait = pace.Decide(gap, args.Distance, args.Run, leaderSpeed > RunningSpeed);
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
                walker.SetGoal(pos, lead);
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
                return Outcome.Failed(Wire.ReasonStuck, $"no progress after {walker.RecoveryAttempts} recoveries");
            }
            return null;
        }

        public void Cancel() => Driver.Stand();
    }

    static class Players
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
