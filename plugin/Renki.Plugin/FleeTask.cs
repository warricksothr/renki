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
        readonly FleeProgress progress = new FleeProgress();
        readonly bool avoidHostiles;
        readonly HostileRoute hostiles = new HostileRoute();
        readonly WorldNavigation.PlanningProbe bearingProbe;
        FleeBearings bearings;
        Outcome aimFailure;
        Vector3 goal;
        float nextAim;
        bool aimed, straight, atPlace, cancelled;

        public FleeTask(FleeArgs args, Vector3 threatPos, GameObject threat, Vector3? towardPos, GameObject toward, bool avoidHostiles = false)
        {
            this.args = args;
            this.threat = threat;
            this.threatPos = threatPos;
            this.towardPos = towardPos;
            this.toward = toward;
            this.avoidHostiles = avoidHostiles;
            bearingProbe = new WorldNavigation.PlanningProbe(diagnostics: true, dry: true, corridor: hostiles.Clear);
            walker.ResetStuck(Player.m_localPlayer.transform.position);
        }

        public Outcome Tick(double dt)
        {
            if (cancelled) return Outcome.Failed(Wire.ReasonStopped, "flee cancelled");
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
            if (progress.Blocked(Convert.ToV3(pos), gap, Time.unscaledTime))
            {
                Driver.Stand();
                Tracer.Emit("flee", "blocked escape", new Dictionary<string, object>
                { ["pos"] = Convert.ToArray(pos), ["threat"] = Convert.ToArray(threatPos), ["goal"] = Convert.ToArray(goal), ["gap"] = gap });
                return Outcome.Failed(Wire.ReasonBlocked, "less than 1 m physical progress in 4 s with a threat within 15 m");
            }
            if (avoidHostiles && gap < args.Distance && walker.DepartDryBank(pos,
                Convert.ToVector3(Flee.Goal(Convert.ToV3(pos), Convert.ToV3(threatPos), null, args.Distance))))
                return walker.Refused();
            if ((!aimed || bearings?.Status == RouteStatus.Searching || Time.time >= nextAim) && !Aim(pos))
            { Driver.Stand(); return aimFailure; }
            // Far enough, unless it is on its way to safety: a hurt bot
            // standing in the open is only waiting for the next creature.
            if (gap >= args.Distance && straight)
            {
                Driver.Stand();
                return Outcome.Succeeded($"{gap:0} m away");
            }
            if (!straight && V3.FlatDistance(Convert.ToV3(pos), Convert.ToV3(goal)) <= AtSafety)
            {
                // Reaching an owner or home is only safe with clearance
                // from the threat. Otherwise keep running instead of
                // standing beside the owner while still being attacked.
                if (gap >= args.Distance)
                {
                    Driver.Stand();
                    return Outcome.Succeeded($"at safety, {gap:0} m from the threat");
                }
                // At a place, such as home, that the threat followed it to:
                // on, straight away.
                atPlace = true;
                if (!Aim(pos)) { Driver.Stand(); return aimFailure; }
            }
            if (Walker.Stranded(p) is Outcome stranded)
            {
                Driver.Stand();
                return stranded;
            }
            if (walker.Refused() is Outcome wet)
            {
                Driver.Stand();
                if (TryAnotherBearing(pos, wet.Reason)) return null;
                return wet;
            }
            if (!walker.Steer(pos, true))
            {
                Driver.Stand();
                if (TryAnotherBearing(pos, Wire.ReasonStuck)) return null;
                return Outcome.Failed(Wire.ReasonStuck,
                    $"no progress after {walker.RecoveryAttempts} recoveries{(walker.StraightLine ? " (no navmesh path)" : "")}");
            }
            return null;
        }

        bool Aim(Vector3 pos)
        {
            nextAim = Time.time + Reaim;
            Vector3? safe = towardPos;
            if (atPlace) safe = null;
            else if (toward != null) safe = toward.transform.position;
            else if (args.Toward?.Entity != null) safe = null; // gone
            var next = Flee.Goal(Convert.ToV3(pos), Convert.ToV3(threatPos), safe.HasValue ? Convert.ToV3(safe.Value) : (V3?)null, args.Distance);
            var nextStraight = !safe.HasValue || Flee.PastThreat(Convert.ToV3(pos), Convert.ToV3(threatPos), Convert.ToV3(safe.Value));
            if (nextStraight && V3.FlatDistance(Convert.ToV3(pos), Convert.ToV3(threatPos)) >= args.Distance)
            { straight = true; return true; }
            if (avoidHostiles && nextStraight)
            {
                if (bearings == null) bearings = new FleeBearings(Convert.ToV3(pos), Convert.ToV3(threatPos), args.Distance, bearingProbe.Probe);
                if (bearings.Status == RouteStatus.Searching)
                {
                    bearingProbe.BeginSlice();
                    if (bearings.Advance() == RouteStatus.Found)
                        Tracer.Emit("flee", "checked escape bearing selected", new Dictionary<string, object>
                        {
                            ["pos"] = Convert.ToArray(pos),
                            ["threat"] = Convert.ToArray(threatPos),
                            ["candidate"] = bearings.Candidate,
                            ["degrees"] = bearings.Degrees,
                            ["samples"] = bearings.Samples,
                            ["physicsCasts"] = bearingProbe.PhysicsCasts,
                            ["goal"] = bearings.Goal.ToArray(),
                        });
                }
                if (bearings.Status != RouteStatus.Found)
                {
                    if (bearings.Status == RouteStatus.NoRoute)
                    {
                        Tracer.Emit("flee", "no checked escape bearing", new Dictionary<string, object>
                        {
                            ["pos"] = Convert.ToArray(pos),
                            ["threat"] = Convert.ToArray(threatPos),
                            ["samples"] = bearings.Samples,
                            ["failures"] = bearingProbe.Failures,
                            ["hostiles"] = hostiles.Snapshot(),
                        });
                        aimFailure = Outcome.Failed(Wire.ReasonUnreachable, "no checked escape prefix across seven bounded bearings");
                    }
                    return false;
                }
                next = bearings.Reaim(Convert.ToV3(pos), Convert.ToV3(threatPos), RepathAfter);
            }
            var nextGoal = Convert.ToVector3(next);
            if (aimed && nextStraight == straight && Vector3.Distance(nextGoal, goal) < RepathAfter) return true;
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
            walker.SetGoal(pos, goal, dryGround: true, avoidHostiles: avoidHostiles);
            return true;
        }

        bool TryAnotherBearing(Vector3 pos, string why)
        {
            if (!avoidHostiles || !straight || bearings?.Status != RouteStatus.Found ||
                why != Wire.ReasonUnreachable && why != Wire.ReasonDeepWater && why != Wire.ReasonStuck) return false;
            Tracer.Emit("flee", "escape goal refused; checking another bearing", new Dictionary<string, object>
            {
                ["pos"] = Convert.ToArray(pos),
                ["candidate"] = bearings.Candidate,
                ["degrees"] = bearings.Degrees,
                ["reason"] = why,
            });
            bearings.Reject(Convert.ToV3(pos), Convert.ToV3(threatPos));
            aimed = false;
            walker.ResetStuck(pos);
            return bearings.Status == RouteStatus.Searching;
        }

        public void Cancel() { cancelled = true; bearings?.Cancel(); Driver.Stand(); }
    }
}
