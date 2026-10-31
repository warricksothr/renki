using System.Collections.Generic;

namespace Renki.Movement
{
    /// <summary>A work trip keeps outside known threats. Starting inside the
    /// clearance still permits moving away, never through the creature.</summary>
    public static class HostileClearance
    {
        // A Greyling can acquire and pursue a walking player before a 24 m
        // exclusion is reached. Leave room to turn before that approach.
        public const double Radius = 40;
        public static bool Clear(V3 from, V3 to, IReadOnlyList<V3> threats)
        {
            foreach (var threat in threats)
            {
                var distance = V3.FlatDistance(from, threat);
                if (distance < Radius)
                {
                    if (V3.FlatDistance(to, threat) <= distance + .0001 ||
                        V3.Dot((from - threat).Flat, (to - from).Flat) < 0) return false;
                    continue;
                }
                WallRoute.Closest(from.Flat, to.Flat, threat.Flat, threat.Flat, out var t, out _);
                if (V3.FlatDistance(from + (to - from) * t, threat) < Radius) return false;
            }
            return true;
        }
    }
}
