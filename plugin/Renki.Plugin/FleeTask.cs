using System.Collections.Generic;
using Renki.Movement;
using Renki.Protocol;
using Renki.Tasks;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>
    /// flee: run from a threat until it is Distance away (docs/BRIDGE-PROTOCOL.md
    /// 7.2), toward safety when the controller names it and it does not lie
    /// past the threat, else straight away. It re-aims as the threat moves.
    /// </summary>
    sealed class FleeTask : IBodyTask
    {
        /// <summary>How often the goal is worked out again.</summary>
        const float Reaim = 0.5f;

        /// <summary>How far the goal moves before the path is worked out again.</summary>
        const float RepathAfter = 3f;

        /// <summary>How near safety counts as there.</summary>
        const double AtSafety = 3;

        readonly FleeArgs args;
        readonly GameObject threat;
        Vector3 threatPos;
        readonly GameObject toward;
        readonly Vector3? towardPos;
        readonly Walker walker = new Walker();
        Vector3 goal;
        float nextAim;
        bool aimed, straight;

        public FleeTask(FleeArgs args, Vector3 threatPos, GameObject threat, Vector3? towardPos, GameObject toward)
        {
            this.args = args;
            this.threat = threat;
            this.threatPos = threatPos;
            this.towardPos = towardPos;
            this.toward = toward;
            walker.ResetStuck(Player.m_localPlayer.transform.position);
        }

        public Outcome Tick(double dt)
        {
            var p = Player.m_localPlayer;
            if (p == null) return Outcome.Failed(Wire.ReasonStopped, "no player");
            var pos = p.transform.position;
            if (args.From.Entity != null)
            {
                // Gone, or dead: nothing left to run from.
                if (threat == null || (threat.TryGetComponent<Character>(out var c) && c.IsDead()))
                {
                    Driver.Stand();
                    return Outcome.Succeeded("the threat is gone");
                }
                threatPos = threat.transform.position;
            }
            var gap = V3.FlatDistance(Convert.ToV3(pos), Convert.ToV3(threatPos));
            if (gap >= args.Distance)
            {
                Driver.Stand();
                return Outcome.Succeeded($"{gap:0} m away");
            }
            if (!aimed || Time.time >= nextAim) Aim(pos);
            // At safety with the threat still close: safety was the owner or
            // home, and the rest is theirs. Running on past it would leave them.
            if (!straight && V3.FlatDistance(Convert.ToV3(pos), Convert.ToV3(goal)) <= AtSafety)
            {
                Driver.Stand();
                return Outcome.Succeeded($"at safety, {gap:0} m from the threat");
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

        void Aim(Vector3 pos)
        {
            nextAim = Time.time + Reaim;
            Vector3? safe = towardPos;
            if (toward != null) safe = toward.transform.position;
            else if (args.Toward?.Entity != null) safe = null; // the entity is gone
            var next = Flee.Goal(Convert.ToV3(pos), Convert.ToV3(threatPos), safe.HasValue ? Convert.ToV3(safe.Value) : (V3?)null, args.Distance);
            var nextGoal = Convert.ToVector3(next);
            var nextStraight = !safe.HasValue || Flee.PastThreat(Convert.ToV3(pos), Convert.ToV3(threatPos), Convert.ToV3(safe.Value));
            if (aimed && nextStraight == straight && Vector3.Distance(nextGoal, goal) < RepathAfter) return;
            if (!aimed || nextStraight != straight)
                Tracer.Emit("flee", nextStraight ? "running away" : "running to safety", new Dictionary<string, object>
                {
                    ["pos"] = Convert.ToArray(pos),
                    ["threat"] = Convert.ToArray(threatPos),
                    ["goal"] = next.ToArray(),
                });
            goal = nextGoal;
            straight = nextStraight;
            aimed = true;
            walker.SetGoal(pos, goal);
        }

        public void Cancel() => Driver.Stand();
    }
}
