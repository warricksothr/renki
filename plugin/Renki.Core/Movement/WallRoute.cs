using System;
using System.Collections.Generic;
using Renki.Protocol;

namespace Renki.Movement
{
    /// <summary>Height-aware capsule clearance against the walls already
    /// measured for perception. Upper-storey walls do not block travel below.</summary>
    public static class WallRoute
    {
        public const double Clearance = .7;
        public const double Height = 1.8;

        public static bool Clear(V3 from, V3 to, IReadOnlyList<Wall> walls, double clearance = Clearance)
        {
            foreach (var wall in walls)
            {
                var a = new V3(wall.A[0], wall.A[1], wall.A[2]);
                var b = new V3(wall.B[0], wall.B[1], wall.B[2]);
                Closest(from, to, a, b, out var t, out var u);
                var p = from + (to - from) * t;
                var q = a + (b - a) * u;
                if (p.Y + Height <= q.Y + .05 || p.Y >= q.Y + wall.H - .05) continue;
                var gap = V3.FlatDistance(p, q);
                if (gap >= clearance) continue;
                // A bot already pressed against a wall may back away on
                // its current side, but may never cross through the wall.
                Closest(from, from, a, b, out _, out var initialU);
                var initial = V3.FlatDistance(from, a + (b - a) * initialU);
                Closest(to, to, a, b, out _, out var finalU);
                var final = V3.FlatDistance(to, a + (b - a) * finalU);
                var sideFrom = (b.X - a.X) * (from.Z - a.Z) - (b.Z - a.Z) * (from.X - a.X);
                var sideTo = (b.X - a.X) * (to.Z - a.Z) - (b.Z - a.Z) * (to.X - a.X);
                if (initial < clearance && sideFrom * sideTo > 0 && gap >= initial - .001 &&
                    final > initial + .01) continue;
                return false;
            }
            return true;
        }

        public static bool Clear(V3 from, IReadOnlyList<V3> path, IReadOnlyList<Wall> walls)
        {
            foreach (var p in path)
            {
                if (!Clear(from, p, walls)) return false;
                from = p;
            }
            return true;
        }

        // Closest parameters on two finite segments in the horizontal plane.
        static void Closest(V3 p, V3 p1, V3 q, V3 q1, out double s, out double t)
        {
            var d = (p1 - p).Flat;
            var e = (q1 - q).Flat;
            var r = (p - q).Flat;
            var a = V3.Dot(d, d);
            var b = V3.Dot(d, e);
            var c = V3.Dot(d, r);
            var f = V3.Dot(e, r);
            var g = V3.Dot(e, e);
            if (a < 1e-9) { s = 0; t = g < 1e-9 ? 0 : Clamp(f / g); return; }
            if (g < 1e-9) { t = 0; s = Clamp(-c / a); return; }
            var denominator = a * g - b * b;
            s = denominator > 1e-9 ? Clamp((b * f - c * g) / denominator) : 0;
            t = (b * s + f) / g;
            if (t < 0) { t = 0; s = Clamp(-c / a); }
            else if (t > 1) { t = 1; s = Clamp((b - c) / a); }
        }

        static double Clamp(double n) => Math.Max(0, Math.Min(1, n));
    }
}
