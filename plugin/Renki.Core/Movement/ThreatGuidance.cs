using System;
using System.Collections.Generic;

namespace Renki.Movement
{
    /// <summary>Bounded geometric guidance, never a checked movement path.
    /// Ring hints can avoid known water; only the caller's live step probe
    /// can authorize support, clearance or arrival.</summary>
    public sealed class ThreatGuidance
    {
        public const int ThreatLimit = 4, Spokes = 8, SampleLimit = 128, HintLimit = 4096, HintStepLimit = 64;
        readonly V3 goal;
        readonly HostileClearance.Threat[] threats;
        readonly List<V3> points = new List<V3>();
        readonly double[] costs;
        readonly Func<V3, bool> hint;
        readonly Dictionary<(int, int), bool> hints = new Dictionary<(int, int), bool>();
        public bool Active { get; }
        public int Waypoints => points.Count - 1;
        public int RejectedHints { get; private set; }
        public int HintSamples => hints.Count;
        public double InitialEstimate { get; }

        public ThreatGuidance(V3 start, V3 goal, IReadOnlyList<HostileClearance.Threat> threats,
            Func<V3, bool> hint = null)
        {
            if (!Finite(start) || !Finite(goal) || threats == null || threats.Count > SampleLimit)
                throw new ArgumentException("invalid threat guidance scope");
            this.goal = goal;
            this.hint = hint;
            this.threats = new HostileClearance.Threat[threats.Count];
            var selected = new List<HostileClearance.Threat>();
            for (var i = 0; i < threats.Count; i++)
            {
                var threat = threats[i];
                if (!Finite(threat.Pos) || !Finite(threat.Radius) || threat.Radius <= 0 || threat.Radius > 128)
                    throw new ArgumentException("invalid threat envelope");
                this.threats[i] = threat;
                selected.Add(threat);
            }
            points.Add(goal);
            InitialEstimate = V3.FlatDistance(start, goal);
            if (Clear(start, goal)) return;
            selected.Sort((a, b) => Relevance(start, goal, a).CompareTo(Relevance(start, goal, b)));
            for (var i = 0; i < Math.Min(ThreatLimit, selected.Count); i++)
            {
                var threat = selected[i];
                // Adjacent chords stay outside the actual clearance circle.
                var radius = threat.Radius / Math.Cos(Math.PI / Spokes) + 1;
                for (var spoke = 0; spoke < Spokes; spoke++)
                {
                    var angle = spoke * Math.PI * 2 / Spokes;
                    var at = new V3(threat.Pos.X + Math.Cos(angle) * radius, start.Y,
                        threat.Pos.Z + Math.Sin(angle) * radius);
                    if (!Outside(at) || !Hint(at)) { RejectedHints++; continue; }
                    if (!points.Exists(p => V3.FlatDistance(p, at) < .1)) points.Add(at);
                }
            }
            costs = new double[points.Count];
            var closed = new bool[points.Count];
            for (var i = 1; i < costs.Length; i++) costs[i] = double.PositiveInfinity;
            // Dijkstra over at most33 vertices. These edges are geometric
            // hints only; no terrain or body physics is inferred from them.
            for (var iteration = 0; iteration < points.Count; iteration++)
            {
                var next = -1;
                for (var i = 0; i < points.Count; i++)
                    if (!closed[i] && (next < 0 || costs[i] < costs[next])) next = i;
                if (next < 0 || double.IsPositiveInfinity(costs[next])) break;
                closed[next] = true;
                for (var i = 0; i < points.Count; i++)
                    if (!closed[i] && Clear(points[i], points[next]))
                        costs[i] = Math.Min(costs[i], costs[next] + V3.FlatDistance(points[i], points[next]));
            }
            var estimated = ViaGraph(start);
            if (!double.IsPositiveInfinity(estimated))
            {
                InitialEstimate = estimated;
                Active = true;
            }
        }

        public double Estimate(V3 at)
        {
            if (!Finite(at)) throw new ArgumentException("invalid guidance position");
            if (!Active) return V3.FlatDistance(at, goal);
            var estimate = ViaGraph(at);
            // A live-probed node can stand on a bridge over terrain classified
            // as water. Do not let that terrain hint disconnect real support.
            return double.IsPositiveInfinity(estimate) && !Hint(at) ? V3.FlatDistance(at, goal) : estimate;
        }

        double ViaGraph(V3 at)
        {
            if (Clear(at, goal)) return V3.FlatDistance(at, goal);
            var best = double.PositiveInfinity;
            for (var i = 1; i < points.Count; i++)
                if (!double.IsPositiveInfinity(costs[i]) && Clear(at, points[i]))
                    best = Math.Min(best, V3.FlatDistance(at, points[i]) + costs[i]);
            return best;
        }
        bool Clear(V3 from, V3 to)
        {
            if (!HostileClearance.Clear(from, to, threats)) return false;
            if (hint == null) return true;
            var steps = (int)Math.Max(1, Math.Min(HintStepLimit, Math.Ceiling(V3.FlatDistance(from, to) / 2)));
            for (var i = 0; i <= steps; i++)
                if (!Hint(from + (to - from) * ((double)i / steps))) return false;
            return true;
        }
        bool Hint(V3 at)
        {
            if (hint == null) return true;
            var cell = ((int)Math.Round(at.X / 2), (int)Math.Round(at.Z / 2));
            if (hints.TryGetValue(cell, out var dry)) return dry;
            // A missing hint never invents physical support or permission.
            if (hints.Count >= HintLimit) return true;
            dry = hint(new V3(cell.Item1 * 2, at.Y, cell.Item2 * 2));
            hints[cell] = dry;
            return dry;
        }
        bool Outside(V3 at)
        {
            foreach (var threat in threats)
                if (V3.FlatDistance(at, threat.Pos) < threat.Radius) return false;
            return true;
        }
        static double Relevance(V3 start, V3 goal, HostileClearance.Threat t) =>
            V3.FlatDistance(start, t.Pos) + V3.FlatDistance(t.Pos, goal) - 2 * t.Radius;
        static bool Finite(V3 p) => Finite(p.X) && Finite(p.Y) && Finite(p.Z);
        static bool Finite(double n) => !double.IsNaN(n) && !double.IsInfinity(n);
    }
}
