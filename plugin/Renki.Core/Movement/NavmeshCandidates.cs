using System;
using System.Collections.Generic;
using System.Linq;

namespace Renki.Movement
{
    /// <summary>Bound speculative navmesh checks while local planning waits for
    /// asynchronous tiles. A candidate is still only a hint: RouteValidation
    /// must check every leg before it can become movement authority.</summary>
    public sealed class NavmeshCandidates
    {
        public const int MaxAttempts = 8;
        public const double MinProgress = 6;
        readonly List<V3[]> attempted = new List<V3[]>();
        V3? origin;
        double nearest = double.PositiveInfinity;

        public void Reset() { attempted.Clear(); origin = null; nearest = double.PositiveInfinity; }

        // Observe actual travel, including local detours. Moving away from the
        // goal must not make an old partial endpoint count as progress again.
        public void Observe(V3 at, V3 goal) => nearest = Math.Min(nearest, V3.FlatDistance(at, goal));

        public RouteValidation Try(V3 from, V3 goal, IReadOnlyList<V3> points,
            Func<V3, V3, V3?> probe, double within, double arrivalLevel = Arrival.Level)
        {
            Observe(from, goal);
            if (!origin.HasValue || (from - origin.Value).Length >= 2)
            {
                attempted.Clear(); origin = from;
            }
            if (points.Count == 0 || points.Count > RouteValidation.MaxSamples) return null;
            var end = points[points.Count - 1];
            var complete = Arrival.Reached(end, goal, within, arrivalLevel);
            // A checked prefix can bring local search to the obstacle without
            // inventing the missing final leg or declaring the goal reached.
            if ((!complete && nearest - V3.FlatDistance(end, goal) < MinProgress) ||
                attempted.Count >= MaxAttempts || attempted.Any(old => Same(old, points))) return null;
            var copy = points.ToArray();
            attempted.Add(copy);
            return new RouteValidation(from, complete ? goal : end, new RouteHint(0, copy), probe, complete ? within : .15, complete ? arrivalLevel : Arrival.Level);
        }

        /// <summary>A failure can leave useful checked ground before the
        /// obstacle. Never include the refused sample or invent a final leg.</summary>
        public IReadOnlyList<V3> CheckedPrefix(RouteValidation route, V3 from, V3 goal)
        {
            Observe(from, goal);
            return route.Path.Count > 0 && nearest - V3.FlatDistance(route.Path[route.Path.Count - 1], goal) >= MinProgress
                ? route.Path : Array.Empty<V3>();
        }

        static bool Same(IReadOnlyList<V3> a, IReadOnlyList<V3> b)
        {
            if (a.Count != b.Count) return false;
            for (var i = 0; i < a.Count; i++)
                if ((a[i] - b[i]).Length > .1) return false;
            return true;
        }
    }
}
