using System;
using System.Collections.Generic;

namespace Renki.Movement
{
    /// <summary>Loaded entrances guide a raised-floor search. Each selected
    /// floor point still requires a complete physically checked route.</summary>
    public sealed class EntranceHints
    {
        readonly List<V3> tried = new List<V3>();
        public const int Limit = 3;
        public void Reset() => tried.Clear();
        public V3? Select(V3 from, V3 goal, IEnumerable<V3> floors)
        {
            if (tried.Count >= Limit || goal.Y - from.Y <= Arrival.Level || V3.FlatDistance(from, goal) > 32) return null;
            V3? best = null;
            var cost = double.PositiveInfinity;
            foreach (var floor in floors)
            {
                if (Math.Abs(floor.Y - goal.Y) > Arrival.Level || V3.FlatDistance(floor, goal) > 12 ||
                    tried.Exists(old => (old - floor).Length < 1)) continue;
                var next = V3.FlatDistance(from, floor) + V3.FlatDistance(floor, goal);
                if (next >= cost) continue;
                best = floor; cost = next;
            }
            if (best.HasValue) tried.Add(best.Value);
            return best;
        }
    }
}
