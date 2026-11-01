using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using Renki.Protocol;
using Renki.Tasks;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>Native escape/observe/resume child of a single move_to task.</summary>
    sealed class TravelEncounter
    {
        readonly TravelRecovery policy = new TravelRecovery();
        readonly List<Character> characters = new List<Character>();
        Character threat;
        FleeTask escape;
        float nextScan;
        bool saturated;
        public bool Resumed { get; private set; }
        string escapeDetail;
        string blockedVersion, observedVersion;
        Outcome blockedFailure;

        public bool Tick(Player p, double dt, out Outcome failure)
        {
            failure = null; Resumed = false;
            var now = Time.unscaledTime;
            if (now >= nextScan)
            {
                nextScan = now + .25f;
                Scan(p);
                if (policy.Mode == TravelMode.Blocked) observedVersion = RouteVersion(p);
            }
            var previous = policy.Mode;
            policy.Decide(now, threat != null, saturated || !TravelRecovery.Safe(p.GetHealth(), p.GetMaxHealth(),
                p.GetStamina(), p.GetFoods().Count, p.InWater() || p.IsSwimming(), p.IsEncumbered()),
                routeChanged: observedVersion != blockedVersion);
            if (policy.Mode != previous) Emit();
            if (policy.Mode == TravelMode.Escape)
            {
                if (escape == null)
                {
                    Driver.Stand();
                    escapeDetail = null;
                    if (threat == null || threat.m_nview == null || !threat.m_nview.IsValid())
                    { policy.Escaped(false); escapeDetail = "travel threat identity unavailable"; Emit(); }
                    else escape = new FleeTask(new FleeArgs
                    { From = new Target { Entity = Perceiver.Id(threat.m_nview) }, Distance = TravelRecovery.EscapeDistance },
                        threat.transform.position, threat.gameObject, null, null, avoidHostiles: true);
                }
                if (escape != null)
                {
                    var ended = escape.Tick(dt);
                    if (ended != null)
                    {
                        escape.Cancel(); escape = null; nextScan = 0;
                        escapeDetail = ended.Detail ?? ended.Reason;
                        policy.Escaped(ended.Status == Wire.StatusSucceeded);
                        Emit();
                    }
                }
            }
            if (policy.Mode == TravelMode.Refused)
            {
                escape?.Cancel(); escape = null; Driver.Stand();
                failure = Outcome.Failed(Wire.ReasonRefused, policy.Reason +
                    (escapeDetail == null ? "" : ": " + escapeDetail) +
                    (blockedFailure == null ? "" : ": " + blockedFailure.Detail));
                return true;
            }
            if (policy.Mode == TravelMode.Observe || policy.Mode == TravelMode.Blocked) Driver.Stand();
            Resumed = policy.Mode == TravelMode.Walk && (previous == TravelMode.Observe || previous == TravelMode.Blocked);
            return policy.Mode != TravelMode.Walk;
        }

        public Outcome Blocked(Player p, Outcome failure)
        {
            blockedFailure = failure;
            blockedVersion = observedVersion = RouteVersion(p);
            policy.RouteBlocked();
            Emit(); Driver.Stand();
            return policy.Mode == TravelMode.Blocked ? null :
                Outcome.Failed(failure.Reason, policy.Reason + ": " + failure.Detail);
        }

        // Changes can request a new physical search, never grant movement.
        // Quantize away small idle motion and sample only at the scan cadence.
        string RouteVersion(Player p)
        {
            var at = Convert.ToV3(p.transform.position);
            var result = new StringBuilder();
            result.Append(WorldNavigation.GeometryRevision).Append(':')
                .Append(Math.Round(WorldNavigation.WaterLevel(at) * 4)).Append(':')
                .Append(Math.Round(at.X * 4)).Append(',').Append(Math.Round(at.Y * 4)).Append(',').Append(Math.Round(at.Z * 4));
            foreach (var c in characters.Where(c => Relevant(c, p)).OrderBy(c => c.GetInstanceID()))
            {
                var pos = c.transform.position;
                result.Append('|').Append(c.GetInstanceID()).Append(':').Append(c.GetLevel()).Append(':')
                    .Append(Math.Round(pos.x * 2)).Append(',').Append(Math.Round(pos.y * 2)).Append(',').Append(Math.Round(pos.z * 2));
            }
            return result.ToString();
        }

        static bool Relevant(Character c, Player p) => c != null && !(c is Player) && !c.IsDead() &&
            !c.IsTamed() && c.GetBaseAI() is MonsterAI && BaseAI.IsEnemy(p, c);

        void Scan(Player p)
        {
            characters.Clear(); threat = null;
            Character.GetCharactersInRange(p.transform.position, 64, characters);
            saturated = characters.Count > 128;
            if (saturated) return;
            var nearest = float.MaxValue;
            foreach (var c in characters)
            {
                if (!Relevant(c, p)) continue;
                var ai = (MonsterAI)c.GetBaseAI();
                var gap = Vector3.Distance(p.transform.position, c.transform.position);
                if (gap >= nearest || !TravelRecovery.Engaged(gap, ai.GetTargetCreature() == p, ai.HaveTarget())) continue;
                nearest = gap; threat = c;
            }
        }

        void Emit() => Tracer.Emit("journey", "travel encounter transition", new Dictionary<string, object>
        {
            ["mode"] = policy.Mode.ToString().ToLowerInvariant(),
            ["reason"] = policy.Reason,
            ["attempts"] = policy.Attempts,
            ["threat"] = threat?.m_nview != null && threat.m_nview.IsValid() ? Perceiver.Id(threat.m_nview) : null,
            ["escapeDetail"] = escapeDetail,
            ["routeFailure"] = blockedFailure?.Detail,
            ["routeChanged"] = observedVersion != blockedVersion,
        });

        public void Cancel() { policy.Cancel(); escape?.Cancel(); escape = null; Driver.Stand(); }
    }
}
