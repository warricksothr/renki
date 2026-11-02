using System.Collections.Generic;
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

        public bool Tick(Player p, double dt, out Outcome failure)
        {
            failure = null; Resumed = false;
            var now = Time.unscaledTime;
            if (now >= nextScan)
            {
                nextScan = now + .25f;
                Scan(p);
            }
            var previous = policy.Mode;
            policy.Decide(now, threat != null, saturated || !TravelRecovery.Safe(p.GetHealth(), p.GetMaxHealth(),
                p.GetStamina(), p.GetFoods().Count, p.InWater() || p.IsSwimming(), p.IsEncumbered()));
            if (policy.Mode != previous) Emit();
            if (policy.Mode == TravelMode.Escape)
            {
                if (escape == null)
                {
                    Driver.Stand();
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
                    (escapeDetail == null ? "" : ": " + escapeDetail));
                return true;
            }
            if (policy.Mode == TravelMode.Observe) Driver.Stand();
            Resumed = policy.Mode == TravelMode.Walk && previous == TravelMode.Observe;
            return policy.Mode != TravelMode.Walk;
        }

        void Scan(Player p)
        {
            characters.Clear(); threat = null;
            Character.GetCharactersInRange(p.transform.position, 32, characters);
            saturated = characters.Count > 128;
            if (saturated) return;
            var nearest = float.MaxValue;
            foreach (var c in characters)
            {
                if (c == null || c is Player || c.IsDead() || c.IsTamed() ||
                    !(c.GetBaseAI() is MonsterAI ai) || !BaseAI.IsEnemy(p, c)) continue;
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
        });

        public void Cancel() { policy.Cancel(); escape?.Cancel(); escape = null; Driver.Stand(); }
    }
}
