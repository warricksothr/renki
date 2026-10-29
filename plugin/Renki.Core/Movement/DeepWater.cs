using System;

namespace Renki.Movement
{
    /// <summary>
    /// Deep water on the way, and the way out of it. The game's sea and lakes
    /// share one water level; the ground under a point says how deep it is
    /// there. The bot wades what is shallower than Deep and will not be sent
    /// across what is deeper on a straight line, where it drowns: it swims
    /// slowly, cannot regain stamina while swimming, and at no stamina loses
    /// health.
    /// </summary>
    public static class DeepWater
    {
        /// <summary>Deeper than this, in metres, the bot would swim.</summary>
        public const double Deep = 1.4;

        /// <summary>How often the straight line is sampled, in metres.</summary>
        public const double Step = 2;

        /// <summary>The first point on the straight line from a to b where the
        /// water is deeper than Deep, or null when it stays wadeable. Deep
        /// water the line starts in does not count: the bot is in it already,
        /// and the way out must stay open. ground answers NaN where it does
        /// not know, as in a zone not loaded, and that does not count either.</summary>
        public static V3? Crossing(V3 a, V3 b, Func<double, double, double> ground, double level)
        {
            var d = V3.FlatDistance(a, b);
            var n = Math.Max(1, (int)Math.Ceiling(d / Step));
            var leaving = true;
            for (var i = 0; i <= n; i++)
            {
                double t = (double)i / n, x = a.X + (b.X - a.X) * t, z = a.Z + (b.Z - a.Z) * t;
                var g = ground(x, z);
                if (double.IsNaN(g)) continue;
                var deep = level - g > Deep;
                if (deep && !leaving) return new V3(x, g, z);
                if (!deep) leaving = false;
            }
            return null;
        }

        /// <summary>The nearest point within reach where the bot can stand,
        /// searched in rings around pos, or null if there is none.</summary>
        public static V3? Shore(V3 pos, Func<double, double, double> ground, double level, double reach = 60)
        {
            const int directions = 16;
            for (var r = 4.0; r <= reach; r += 4)
            {
                V3? best = null;
                var bestDepth = double.MaxValue;
                for (var i = 0; i < directions; i++)
                {
                    var a = 2 * Math.PI * i / directions;
                    double x = pos.X + r * Math.Cos(a), z = pos.Z + r * Math.Sin(a);
                    var depth = level - ground(x, z); // NaN where unknown, and never dry
                    // Of the dry points on the nearest ring, the highest.
                    if (depth < 0.3 && depth < bestDepth)
                    {
                        best = new V3(x, ground(x, z), z);
                        bestDepth = depth;
                    }
                }
                if (best != null) return best;
            }
            return null;
        }
    }
}
