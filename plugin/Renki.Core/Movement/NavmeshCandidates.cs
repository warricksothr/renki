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
        readonly List<V3[]> attempted = new List<V3[]>();
        V3? origin;

        public void Reset() { attempted.Clear(); origin = null; }

        public RouteValidation Try(V3 from, V3 goal, IReadOnlyList<V3> points,
            Func<V3, V3, V3?> probe, double within)
        {
            if (!origin.HasValue || (from - origin.Value).Length >= 2)
            {
                attempted.Clear(); origin = from;
            }
            if (points.Count == 0 || points.Count > RouteValidation.MaxSamples ||
                !Arrival.Reached(points[points.Count - 1], goal, within) ||
                attempted.Count >= MaxAttempts || attempted.Any(old => Same(old, points))) return null;
            var copy = points.ToArray();
            attempted.Add(copy);
            return new RouteValidation(from, goal, new RouteHint(0, copy), probe, within);
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
