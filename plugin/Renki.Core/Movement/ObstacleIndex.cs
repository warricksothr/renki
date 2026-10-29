using System;
using System.Collections.Generic;

namespace Renki.Movement
{
    /// <summary>A solid upright box or the cylindrical middle of a capsule.
    /// Mesh bounds are not solids and must never be put here.</summary>
    public readonly struct ObstacleShape
    {
        public readonly V3 Center;
        public readonly double HalfX, HalfZ, MinY, MaxY, Yaw;
        public readonly bool Circle;
        readonly double cosine, sine;
        public ObstacleShape(V3 center, double halfX, double halfZ, double minY, double maxY, double yaw = 0, bool circle = false)
        {
            Center = center; HalfX = halfX; HalfZ = halfZ; MinY = minY; MaxY = maxY; Yaw = yaw; Circle = circle;
            cosine = Math.Cos(yaw); sine = Math.Sin(yaw);
        }

        V3 Local(V3 p)
        {
            var d = p - Center;
            return new V3(d.X * cosine + d.Z * sine, d.Y, -d.X * sine + d.Z * cosine);
        }
        double PointDistance(V3 p)
        {
            if (Circle) return Math.Max(0, p.Flat.Length - HalfX);
            var x = Math.Max(0, Math.Abs(p.X) - HalfX);
            var z = Math.Max(0, Math.Abs(p.Z) - HalfZ);
            return Math.Sqrt(x * x + z * z);
        }
        public bool Blocks(V3 from, V3 to, double radius = .4)
        {
            // Only prune when the solid intersects a height shared by the
            // body's central cylinder along this entire leg. Ramps/steps
            // with no such overlap still use the authoritative live query.
            var low = Math.Max(from.Y, to.Y) + 1.05;
            var high = Math.Min(from.Y, to.Y) + 1.4;
            if (low > high || MaxY <= low || MinY >= high) return false;
            var a = Local(from); var b = Local(to);
            // A body touching/overlapping a shape may be backing out. Let
            // live physics decide instead of trapping it with cached bounds.
            if (PointDistance(a) <= radius) return false;
            if (Circle)
            {
                WallRoute.Closest(a, b, V3.Zero, V3.Zero, out var t, out _);
                return (a + (b - a) * t).Flat.Length < HalfX + radius;
            }
            if (PointDistance(b) < radius) return true;
            var hx = HalfX; var hz = HalfZ;
            V3 Corner(int i) => i == 0 ? new V3(-hx, 0, -hz) : i == 1 ? new V3(hx, 0, -hz)
                : i == 2 ? new V3(hx, 0, hz) : new V3(-hx, 0, hz);
            for (var i = 0; i < 4; i++)
            {
                var c = Corner(i); var d = Corner((i + 1) % 4);
                WallRoute.Closest(a, b, c, d, out var t, out var u);
                if (V3.FlatDistance(a + (b - a) * t, c + (d - c) * u) < radius) return true;
            }
            return false;
        }
        public (double X, double Z) Extents => Circle ? (HalfX, HalfX) :
            (Math.Abs(cosine) * HalfX + Math.Abs(sine) * HalfZ,
             Math.Abs(sine) * HalfX + Math.Abs(cosine) * HalfZ);
    }

    /// <summary>Main-thread spatial hash: inspect only primitives near a leg.</summary>
    public sealed class ObstacleIndex
    {
        const double Cell = 4;
        readonly Dictionary<int, ObstacleShape> shapes = new Dictionary<int, ObstacleShape>();
        readonly Dictionary<(int, int), HashSet<int>> cells = new Dictionary<(int, int), HashSet<int>>();
        readonly HashSet<int> checkedIds = new HashSet<int>();
        public int Count => shapes.Count;
        public long Tested { get; private set; }
        public long Rejected { get; private set; }
        public long Revision { get; private set; }
        static int At(double x) => (int)Math.Floor(x / Cell);
        static (int, int, int, int) Area(ObstacleShape s)
        {
            var e = s.Extents;
            return (At(s.Center.X - e.X), At(s.Center.Z - e.Z), At(s.Center.X + e.X), At(s.Center.Z + e.Z));
        }
        public void Put(int id, ObstacleShape shape)
        {
            Remove(id);
            shapes[id] = shape;
            var (x0, z0, x1, z1) = Area(shape);
            for (var x = x0; x <= x1; x++) for (var z = z0; z <= z1; z++)
                {
                    if (!cells.TryGetValue((x, z), out var bucket)) cells[(x, z)] = bucket = new HashSet<int>();
                    bucket.Add(id);
                }
            Revision++;
        }
        public void Remove(int id)
        {
            if (!shapes.TryGetValue(id, out var old)) return;
            var (x0, z0, x1, z1) = Area(old);
            for (var x = x0; x <= x1; x++) for (var z = z0; z <= z1; z++)
                    if (cells.TryGetValue((x, z), out var bucket)) { bucket.Remove(id); if (bucket.Count == 0) cells.Remove((x, z)); }
            shapes.Remove(id);
            Revision++;
        }
        // Validate a possible blocker against the scene before rejecting a
        // leg. Stale or evicted records never establish that space is free.
        // The callback must not mutate this index during iteration.
        public bool Blocked(V3 from, V3 to, Func<int, bool> current = null)
        {
            checkedIds.Clear();
            for (var x = At(Math.Min(from.X, to.X) - .4); x <= At(Math.Max(from.X, to.X) + .4); x++)
                for (var z = At(Math.Min(from.Z, to.Z) - .4); z <= At(Math.Max(from.Z, to.Z) + .4); z++)
                    if (cells.TryGetValue((x, z), out var bucket)) foreach (var id in bucket)
                            if (checkedIds.Add(id))
                            {
                                Tested++;
                                if (shapes[id].Blocks(from, to) && (current == null || current(id))) { Rejected++; return true; }
                            }
            return false;
        }
        public void Clear() { shapes.Clear(); cells.Clear(); checkedIds.Clear(); Revision++; }

        public static double Priority(V3 bot, V3 toward, V3 obstacle)
        {
            var d = (obstacle - bot).Flat;
            if (d.Length <= 8) return d.Length;
            var heading = (toward - bot).Flat.Normalized;
            var ahead = V3.Dot(d, heading);
            var side = Math.Abs(d.X * heading.Z - d.Z * heading.X);
            return 8 + d.Length * (ahead > 0 && ahead <= 32 && side <= 4 + ahead * .2 ? .4 : 1.5);
        }
    }
}
