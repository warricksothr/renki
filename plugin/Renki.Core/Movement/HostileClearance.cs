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
        public static double RadiusFor(string prefab, int level = 1) => level != 1 ? Radius : prefab == "Neck" ? 16 : prefab == "Boar" ? 24 : Radius;

        public readonly struct Threat
        {
            public readonly V3 Pos;
            public readonly double Radius;
            public Threat(V3 pos, string prefab = null, int level = 1) { Pos = pos; Radius = RadiusFor(prefab, level); }
        }

        public static bool Clear(V3 from, V3 to, IReadOnlyList<V3> threats)
        {
            foreach (var threat in threats)
                if (!Clear(from, to, threat, Radius)) return false;
            return true;
        }

        public static bool Clear(V3 from, V3 to, IReadOnlyList<Threat> threats)
        {
            foreach (var threat in threats)
                if (!Clear(from, to, threat.Pos, threat.Radius)) return false;
            return true;
        }

        static bool Clear(V3 from, V3 to, V3 threat, double radius)
        {
            // Checked paths include their starting point and final duplicate.
            // Standing still cannot approach a threat or prevent backing out.
            if (V3.FlatDistance(from, to) <= .0001) return true;
            var distance = V3.FlatDistance(from, threat);
            if (distance < radius)
                return V3.FlatDistance(to, threat) > distance + .0001 &&
                    V3.Dot((from - threat).Flat, (to - from).Flat) >= 0;
            WallRoute.Closest(from.Flat, to.Flat, threat.Flat, threat.Flat, out var t, out _);
            return V3.FlatDistance(from + (to - from) * t, threat) >= radius;
        }
    }
}
