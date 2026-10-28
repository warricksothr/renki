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
            RenkiPlugin.Log?.LogInfo($"walker: path {From(from)} -> {From(goal)}: " +
                                     (StraightLine ? "no navmesh path, straight line" : $"{scratch.Count} waypoints"));
        }

        public void ResetStuck(Vector3 pos) => stuck.Reset(Convert.ToV3(pos), Time.time);

        /// <summary>Steers toward the goal for this step. Returns false when stuck recovery gave up.</summary>
        public bool Steer(Vector3 pos, bool run)
        {
            if (!hasGoal) return true;
            var recovery = stuck.Update(Convert.ToV3(pos), Time.time);
            if (recovery != Recovery.None)
                RenkiPlugin.Log?.LogInfo($"walker: stuck at {From(pos)}, trying {recovery} (attempt {stuck.Attempts})");
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
            // A strafe walks, so it cannot carry the bot out of the stuck
            // circle on its own and pass for progress.
            Driver.Run = run && Time.time >= strafeUntil;
            return true;
        }

        public int RecoveryAttempts => stuck.Attempts;

        internal static string From(Vector3 v) => $"({v.x:F1}, {v.y:F1}, {v.z:F1})";
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
        Gait lastGait = (Gait)(-1);

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
                RenkiPlugin.Log?.LogInfo($"follow: {lastGait} -> {gait}, gap {gap:F1} m, me {Walker.From(pos)}, " +
                                         $"{args.Player} {Walker.From(lead)}");
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
