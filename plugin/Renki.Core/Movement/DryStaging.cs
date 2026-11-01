using System.Collections.Generic;

namespace Renki.Movement
{
    /// <summary>At most two checked approach legs per goal, never arrival or
    /// a renewed bank/task budget. The native caller rechecks water and body.</summary>
    public sealed class DryStaging
    {
        public const int Limit = 2;
        public const double MinimumDistance = 3, MinimumProgress = 1, MaximumCost = 12, WaitSeconds = 2;
        // The retained shore rose .89m while planning. Stage with headroom
        // beyond that observation and the unchanged .8m walking buffer;
        // this is a destination preference, never a water-clearance waiver.
        public const double WaterMargin = 2;
        public int Used { get; private set; }
        public bool Available => Used < Limit;
        public bool Take(double waited, bool eligibleStart, IReadOnlyList<V3> path)
        {
            if (!Available || !eligibleStart || !(waited >= WaitSeconds) || double.IsInfinity(waited) || path == null || path.Count < 2) return false;
            Used++;
            return true;
        }
    }
}
