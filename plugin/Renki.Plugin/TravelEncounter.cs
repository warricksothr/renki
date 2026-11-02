using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using Renki.Protocol;
using Renki.Tasks;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>Native defend/escape/observe/resume children of one body task.</summary>
    sealed class TravelEncounter : IWorkEncounter
    {
        internal static TravelEncounter Active;
        readonly TravelRecovery policy = new TravelRecovery();
        readonly List<Character> characters = new List<Character>();
        readonly string id;
        readonly string task;
        readonly bool defend;
        Character threat;
        Character opponent;
        string opponentEntity;
        MeleeTask defense;
        FleeTask escape;
        int threatCount;
        float nextScan;
        bool saturated;
        public bool Resumed { get; private set; }
        string escapeDetail;
        string defenseDetail;
        string blockedVersion, observedVersion;
        Outcome blockedFailure;
        internal JourneyStatus Status => new JourneyStatus
        {
            Id = id,
            Task = task,
            Mode = policy.Mode.ToString().ToLowerInvariant(),
            Attempts = policy.Attempts,
            Reason = policy.Reason
        };

        public TravelEncounter(string id, bool defend, string task = TaskNames.MoveTo)
        {
            this.id = id; this.defend = defend && !string.IsNullOrEmpty(id);
            this.task = task;
            if (!string.IsNullOrEmpty(id)) Active = this;
        }

        static bool Confirmed(Player p, Character c) => c != null &&
            (c.GetBaseAI()?.GetTargetCreature() == p || c == MeleeTask.RecentAttacker);

        bool CanDefend(Player p) => defend && policy.Mode != TravelMode.Defend && policy.Mode != TravelMode.Escape &&
            threat != null && threat.m_nview != null && threat.m_nview.IsValid() &&
            TravelRecovery.DefenseReady(defend, threat.m_nview.GetPrefabName(), threat.GetLevel(), threatCount,
                Confirmed(p, threat), Vector3.Distance(p.transform.position, threat.transform.position),
                p.GetHealth(), p.GetMaxHealth(), p.GetStamina(), p.GetBodyArmor(), p.GetFoods().Count > 0,
                MeleeTask.Best(p) != null, p.IsOnGround());

        bool IWorkEncounter.Tick(double dt, bool canSuspend, out Outcome failure)
        {
            var p = Player.m_localPlayer;
            if (p != null && !p.IsDead()) return Tick(p, dt, out failure, canSuspend);
            failure = Outcome.Failed(Wire.ReasonStopped, "collection body unavailable");
            return true;
        }

        public bool Tick(Player p, double dt, out Outcome failure, bool canSuspend = true)
        {
            failure = null; Resumed = false;
            var now = Time.unscaledTime;
            if (now >= nextScan)
            {
                nextScan = now + .25f;
                Scan(p);
                if (policy.Mode == TravelMode.Blocked) observedVersion = RouteVersion(p);
            }
            if (!canSuspend && threat != null)
            {
                Driver.Stand();
                failure = Outcome.Failed(Wire.ReasonStopped, "collection use is unresolved at encounter; not suspended or replayed");
                return true;
            }
            var previous = policy.Mode;
            policy.Decide(now, threat != null, saturated || !TravelRecovery.Safe(p.GetHealth(), p.GetMaxHealth(),
                p.GetStamina(), p.GetFoods().Count, p.InWater() || p.IsSwimming(), p.IsEncumbered()),
                routeChanged: observedVersion != blockedVersion, canDefend: CanDefend(p));
            if (policy.Mode != previous) Emit();
            if (policy.Mode != TravelMode.Defend && defense != null)
            { defense.Cancel(); defense = null; opponent = null; opponentEntity = null; }
            if (policy.Mode == TravelMode.Defend)
            {
                if (defense == null)
                {
                    opponent = threat;
                    opponentEntity = Perceiver.Id(opponent.m_nview);
                    try
                    {
                        defense = new MeleeTask(id, new AttackArgs { Entity = opponentEntity }, opponent.gameObject);
                    }
                    catch (TaskRefused no)
                    { defenseDetail = no.Message; policy.Defended(false); Emit(); }
                }
                if (defense != null)
                {
                    Outcome ended = null;
                    if (!defense.ObservedDeath && (threatCount > 1 || threat != opponent || !Confirmed(p, opponent)))
                        ended = Outcome.Failed(Wire.ReasonRefused, "travel defense target evidence or threat count changed");
                    else ended = defense.Tick(dt);
                    // The outer task owns escape and its shared time budget.
                    if (ended != null || defense.Status.Mode == "retreat")
                    {
                        var won = ended != null && ended.Status == Wire.StatusSucceeded && ended.Detail == "observed Greyling death" &&
                            ended.Entity == opponentEntity;
                        defenseDetail = ended?.Detail ?? defense.Status.Reason;
                        defense.Cancel(); defense = null; opponent = null; opponentEntity = null; nextScan = 0;
                        policy.Defended(won); Emit();
                    }
                }
            }
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
                    (defenseDetail == null ? "" : ": defense: " + defenseDetail) +
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
            characters.Clear(); threat = null; threatCount = 0;
            Character.GetCharactersInRange(p.transform.position, 64, characters);
            saturated = characters.Count > 128;
            if (saturated) return;
            var nearest = float.MaxValue;
            foreach (var c in characters)
            {
                if (!Relevant(c, p)) continue;
                var ai = (MonsterAI)c.GetBaseAI();
                var gap = Vector3.Distance(p.transform.position, c.transform.position);
                if (gap <= TravelRecovery.TargetRange) threatCount++;
                if (gap >= nearest || !TravelRecovery.Engaged(gap, ai.GetTargetCreature() == p, ai.HaveTarget())) continue;
                nearest = gap; threat = c;
            }
        }

        void Emit() => Tracer.Emit("journey", "travel encounter transition", new Dictionary<string, object>
        {
            ["id"] = id,
            ["task"] = task,
            ["mode"] = policy.Mode.ToString().ToLowerInvariant(),
            ["reason"] = policy.Reason,
            ["attempts"] = policy.Attempts,
            ["threat"] = threat?.m_nview != null && threat.m_nview.IsValid() ? Perceiver.Id(threat.m_nview) : null,
            ["threatCount"] = threatCount,
            ["escapeDetail"] = escapeDetail,
            ["defenseDetail"] = defenseDetail,
            ["routeFailure"] = blockedFailure?.Detail,
            ["routeChanged"] = observedVersion != blockedVersion,
        });

        public void Cancel()
        {
            policy.Cancel(); defense?.Cancel(); defense = null; opponent = null; opponentEntity = null;
            escape?.Cancel(); escape = null; Driver.Stand(); Driver.ClearCombat();
            if (ReferenceEquals(Active, this)) Active = null;
        }
    }
}
