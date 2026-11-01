using System;

namespace Renki.Movement
{
    /// <summary>Actual building collision triangles, not a filled envelope.
    /// Main-thread queries clip each face to a height shared by the moving
    /// body's central cylinder. Floors, ramps and openings remain separate.</summary>
    public sealed class BuildingMesh
    {
        public const int TriangleLimit = 2048;
        readonly V3[] faces;
        readonly V3[] first = new V3[8], second = new V3[8];
        public int Triangles => faces.Length / 3;

        public BuildingMesh(V3[] triangles)
        {
            if (triangles == null || triangles.Length == 0 || triangles.Length % 3 != 0 || triangles.Length / 3 > TriangleLimit)
                throw new ArgumentException("invalid bounded building triangles");
            foreach (var p in triangles)
                if (!Finite(p.X) || !Finite(p.Y) || !Finite(p.Z)) throw new ArgumentException("invalid building vertex");
            faces = (V3[])triangles.Clone();
        }
        static bool Finite(double x) => !double.IsNaN(x) && !double.IsInfinity(x);

        static int Clip(V3[] input, int count, V3[] output, double height, bool above)
        {
            var n = 0;
            for (var i = 0; i < count; i++)
            {
                var a = input[i]; var b = input[(i + 1) % count];
                var insideA = above ? a.Y >= height : a.Y <= height;
                var insideB = above ? b.Y >= height : b.Y <= height;
                if (insideA) output[n++] = a;
                if (insideA != insideB) output[n++] = a + (b - a) * ((height - a.Y) / (b.Y - a.Y));
            }
            return n;
        }

        static double Distance(V3 p, V3[] polygon, int count)
        {
            var distance = double.PositiveInfinity; var positive = false; var negative = false; var area = 0d;
            for (var i = 0; i < count; i++)
            {
                var a = polygon[i]; var b = polygon[(i + 1) % count];
                WallRoute.Closest(p, p, a, b, out _, out var u);
                distance = Math.Min(distance, V3.FlatDistance(p, a + (b - a) * u));
                var cross = (b.X - a.X) * (p.Z - a.Z) - (b.Z - a.Z) * (p.X - a.X);
                positive |= cross > 1e-8; negative |= cross < -1e-8;
                area += a.X * b.Z - b.X * a.Z;
            }
            return Math.Abs(area) > 1e-8 && !(positive && negative) ? 0 : distance;
        }

        public bool Blocks(V3 from, V3 to, double radius = .4)
        {
            var low = Math.Max(from.Y, to.Y) + 1.05;
            var high = Math.Min(from.Y, to.Y) + 1.4;
            if (low > high) return false;
            for (var face = 0; face < faces.Length; face += 3)
            {
                var a = faces[face]; var b = faces[face + 1]; var c = faces[face + 2];
                if (Math.Max(a.Y, Math.Max(b.Y, c.Y)) < low || Math.Min(a.Y, Math.Min(b.Y, c.Y)) > high) continue;
                if (Math.Max(a.X, Math.Max(b.X, c.X)) < Math.Min(from.X, to.X) - radius ||
                    Math.Min(a.X, Math.Min(b.X, c.X)) > Math.Max(from.X, to.X) + radius ||
                    Math.Max(a.Z, Math.Max(b.Z, c.Z)) < Math.Min(from.Z, to.Z) - radius ||
                    Math.Min(a.Z, Math.Min(b.Z, c.Z)) > Math.Max(from.Z, to.Z) + radius) continue;
                first[0] = a; first[1] = b; first[2] = c;
                var count = Clip(first, 3, second, low, true);
                count = Clip(second, count, first, high, false);
                if (count == 0 || Distance(from, first, count) <= radius) continue;
                if (Distance(to, first, count) < radius) return true;
                for (var i = 0; i < count; i++)
                {
                    a = first[i]; b = first[(i + 1) % count];
                    WallRoute.Closest(from, to, a, b, out var t, out var u);
                    if (V3.FlatDistance(from + (to - from) * t, a + (b - a) * u) < radius) return true;
                }
            }
            return false;
        }
    }
}
