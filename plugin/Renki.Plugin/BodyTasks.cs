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
        readonly List<Vector3> scratch = new List<Vector3>();
        Vector3 goal;
        bool hasGoal;
        float strafeUntil;
        float strafeSign = 1;

        public bool StraightLine { get; private set; }

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
            if (StraightLine)
            {
                scratch.Clear();
                scratch.Add(goal);
            }
            path.Set(scratch.Select(Convert.ToV3));
            Tracer.Emit("walker", StraightLine ? "no navmesh path; straight line" : "path",
                new Dictionary<string, object>
                {
                    ["from"] = Convert.ToArray(from),
                    ["to"] = Convert.ToArray(goal),
                    ["waypoints"] = scratch.Count,
                });
        }

        public void ResetStuck(Vector3 pos) => stuck.Reset(Convert.ToV3(pos), Time.time);

        /// <summary>Steers toward the goal for this step. Returns false when stuck recovery gave up.</summary>
        public bool Steer(Vector3 pos, bool run)
        {
            if (!hasGoal) return true;
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
                case Recovery.GiveUp:
                    return false;
            }
            var next = path.Next(Convert.ToV3(pos));
            var target = next.HasValue ? Convert.ToVector3(next.Value) : goal;
            var dir = target - pos;
            dir.y = 0;
            if (Time.time < strafeUntil)
                dir = Vector3.Cross(Vector3.up, dir.normalized) * strafeSign + dir.normalized * 0.3f;
            Driver.Move = dir.sqrMagnitude > 0.0001f ? dir.normalized : Vector3.zero;
            doors.Walk(Player.m_localPlayer, pos, Driver.Move);
            // A strafe walks, so it cannot carry the bot out of the stuck
            // circle on its own and pass for progress.
            Driver.Run = run && Time.time >= strafeUntil;
            return true;
        }

        public int RecoveryAttempts => stuck.Attempts;
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
            if (V3.FlatDistance(Convert.ToV3(pos), Convert.ToV3(goal)) <= args.StopDistance)
            {
                Driver.Stand();
                return Outcome.Succeeded();
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
            if (V3.FlatDistance(Convert.ToV3(pos), Convert.ToV3(goal)) <= Close)
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
                if (tries++ >= Tries)
                    return took > 0
                        ? Outcome.Succeeded($"took {took} {prefab}; the game kept the other {drop.m_itemData.m_stack}")
                        : Outcome.Failed(Wire.ReasonInvalidTarget, $"the game will not hand over the {prefab}");
                p.Pickup(drop.gameObject, false, false);
                arrivedAt = Time.time; // give it a moment to land before trying again
                return null;
            }
            arrivedAt = -1;
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
