using System;
using System.Collections.Generic;
using Renki.Movement;

namespace Renki.Tasks
{
    public sealed class PursuitObservation
    {
        public string Id;
        public V3 Pos;
        public bool Loaded = true, Dead, TargetKnown, Targeting, HuntingPlayers;
    }

    public sealed class IsolationEvidence
    {
        public int Members;
        public readonly List<string> Pursuers = new List<string>();
        public readonly List<string> Unknown = new List<string>();
        public double QuietS;
        public bool Isolated;
        public string Reason;
    }

    /// <summary>Tracks the entire observed group for one native episode.
    /// Missing/remote members remain uncertain; only observed death removes risk.</summary>
    public sealed class PursuitGroup
    {
        public const int Maximum = 32;
        public const double Radius = 60, Separation = 24, QuietTime = 3, FreshFor = 1;
        sealed class Member
        {
            public PursuitObservation Observation;
            public double At, FollowingUntil;
        }
        readonly Dictionary<string, Member> members = new Dictionary<string, Member>();
        readonly string target;
        double sampledAt = double.NegativeInfinity, quietAt = double.NaN;
        bool invalid;

        public PursuitGroup(string target) { this.target = target; }
        public bool Contains(string id) => members.ContainsKey(id);

        public void Sample(IEnumerable<PursuitObservation> observations, V3 bot, double now)
        {
            if (!Finite(now) || now <= sampledAt) { invalid = true; quietAt = double.NaN; return; }
            sampledAt = now;
            var seen = new HashSet<string>();
            foreach (var o in observations)
            {
                if (o == null || string.IsNullOrEmpty(o.Id) || !seen.Add(o.Id) || !Position(o.Pos))
                { invalid = true; continue; }
                if (!members.TryGetValue(o.Id, out var m))
                {
                    if (members.Count >= Maximum) { invalid = true; continue; }
                    members.Add(o.Id, m = new Member());
                    quietAt = double.NaN; // Fresh arrivals cannot inherit quiet evidence.
                }
                if (m.Observation != null && o.Loaded && !o.Dead && m.Observation.Loaded && !m.Observation.Dead &&
                    now - m.At <= FreshFor && Kiting.ObservedPursuit(m.Observation.Pos, o.Pos, bot))
                    m.FollowingUntil = now + 2;
                m.Observation = o; m.At = now;
            }
        }

        public IsolationEvidence Assess(V3 bot, double now)
        {
            var e = new IsolationEvidence { Members = members.Count };
            var safe = !invalid && Position(bot) && Finite(now) && now >= sampledAt && now - sampledAt <= FreshFor;
            var selected = false;
            foreach (var pair in members)
            {
                var m = pair.Value; var o = m.Observation;
                if (o.Dead && o.Loaded) continue;
                var known = o.Loaded && o.TargetKnown && m.At == sampledAt && now - m.At <= FreshFor;
                if (!known) { e.Unknown.Add(pair.Key); safe = false; }
                var following = o.Targeting || now < m.FollowingUntil || pair.Key != target && o.HuntingPlayers;
                if (following) e.Pursuers.Add(pair.Key);
                if (pair.Key == target) selected = known && following;
                else if (!known || following || V3.FlatDistance(bot, o.Pos) < Separation) safe = false;
            }
            safe &= selected && e.Pursuers.Count == 1;
            if (!safe) quietAt = double.NaN;
            else if (double.IsNaN(quietAt)) quietAt = now;
            e.QuietS = safe ? Math.Max(0, now - quietAt) : 0;
            e.Isolated = safe && e.QuietS >= QuietTime;
            e.Reason = invalid ? "incomplete or invalid group observation" : !selected ? "selected pursuit is unconfirmed" :
                e.Unknown.Count > 0 ? "group targeting or visibility is unknown" : !safe ? "another group member is pursuing or too close" :
                !e.Isolated ? "confirming continuous single pursuit" : "one isolated confirmed pursuer";
            return e;
        }

        public static bool WithinBudget(double elapsed, double displacement) =>
            Finite(elapsed) && Finite(displacement) && elapsed >= 0 && elapsed < 60 && displacement >= 0 && displacement <= 40;

        public static bool Prepared(double hp, double max, double stamina, bool fed, double armour, double weaponFraction, double armourFraction) =>
            Finite(hp) && Finite(max) && Finite(stamina) && Finite(armour) && Finite(weaponFraction) && Finite(armourFraction) &&
            max > 0 && hp >= Math.Max(30, max * .8) && stamina >= 40 && fed && armour >= 2 && weaponFraction >= .25 && armourFraction >= .25;

        // Prefer the exposed edge nearest the bot. A candidate hidden behind a
        // competitor is never made attractive merely by belonging to a crowd.
        public static double EdgeScore(V3 bot, V3 candidate, IEnumerable<V3> competitors)
        {
            var distance = V3.FlatDistance(bot, candidate);
            var nearest = 12d;
            foreach (var other in competitors)
            {
                if (Flee.PastThreat(bot, other, candidate) && V3.FlatDistance(bot, other) < distance) return double.PositiveInfinity;
                nearest = Math.Min(nearest, V3.FlatDistance(candidate, other));
            }
            return distance - nearest * .5;
        }

        static bool Finite(double n) => !double.IsNaN(n) && !double.IsInfinity(n);
        static bool Position(V3 p) => Finite(p.X) && Finite(p.Y) && Finite(p.Z);
    }
}
