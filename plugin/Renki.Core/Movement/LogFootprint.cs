using System;
using System.Collections.Generic;

namespace Renki.Movement
{
    /// <summary>A conservative travel exclusion around one log collider,
    /// not a claim that its entire bounding box is solid. Live body/support
    /// checks still validate every accepted leg.</summary>
    public readonly struct LogFootprint
    {
        public const double Padding = .4;
        public readonly V3 Center;
        public readonly double HalfX, HalfZ, MinY, MaxY, Yaw;
        readonly double cosine, sine;
        public LogFootprint(V3 center, double halfX, double halfZ, double minY, double maxY, double yaw = 0)
        {
            Center = center; HalfX = halfX; HalfZ = halfZ; MinY = minY; MaxY = maxY; Yaw = yaw;
            cosine = Math.Cos(yaw); sine = Math.Sin(yaw);
        }
        public ObstacleShape Envelope => new ObstacleShape(Center, HalfX, HalfZ, MinY, MaxY, Yaw);
        V3 Local(V3 p)
        {
            var d = p - Center;
            return new V3(d.X * cosine + d.Z * sine, 0, -d.X * sine + d.Z * cosine);
        }
        double Clearance(V3 p)
        {
            var x = Math.Abs(p.X) - HalfX; var z = Math.Abs(p.Z) - HalfZ;
            return Math.Sqrt(Math.Max(0, x) * Math.Max(0, x) + Math.Max(0, z) * Math.Max(0, z)) + Math.Min(0, Math.Max(x, z));
        }
        public bool Leaving(V3 from, V3 to)
        {
            var a = Local(from); var b = Local(to); var d = b - a;
            var initial = Clearance(a);
            if (initial > Padding || Clearance(b) <= initial + .0001) return false;
            var x = Math.Abs(a.X) - HalfX; var z = Math.Abs(a.Z) - HalfZ;
            // A nonnegative outward derivative guarantees clearance cannot
            // initially decrease on this straight leg (distance is convex).
            var dx = a.X == 0 ? Math.Abs(d.X) : Math.Sign(a.X) * d.X;
            var dz = a.Z == 0 ? Math.Abs(d.Z) : Math.Sign(a.Z) * d.Z;
            if (x > 0 || z > 0) return Math.Max(0, x) * dx + Math.Max(0, z) * dz >= 0;
            if (Math.Abs(x - z) < .000001) return Math.Max(dx, dz) > .0001;
            return x > z ? dx > .0001 : dz > .0001;
        }
        public bool Blocks(V3 from, V3 to)
        {
            // Preserve vertical extent: an elevated log need not block a
            // route below it, and feet already above it may move away.
            if (MaxY <= Math.Min(from.Y, to.Y) + .05 || MinY >= Math.Max(from.Y, to.Y) + 1.8) return false;
            if (Leaving(from, to)) return false;
            var a = Local(from); var b = Local(to);
            if (Clearance(a) < Padding || Clearance(b) < Padding) return true;
            for (var i = 0; i < 4; i++)
            {
                var c = Local(Corner(i)); var d = Local(Corner((i + 1) % 4));
                WallRoute.Closest(a, b, c, d, out var t, out var u);
                if (V3.FlatDistance(a + (b - a) * t, c + (d - c) * u) < Padding) return true;
            }
            return false;
        }
        public V3 Corner(int i)
        {
            var x = i == 0 || i == 3 ? -HalfX : HalfX;
            var z = i < 2 ? -HalfZ : HalfZ;
            return new V3(Center.X + x * cosine - z * sine, MinY, Center.Z + x * sine + z * cosine);
        }
        public static LogFootprint? Measure(IReadOnlyList<V3> corners, V3 heading)
        {
            if (corners.Count == 0) return null;
            var yaw = Math.Atan2(heading.Z, heading.X); var cos = Math.Cos(yaw); var sin = Math.Sin(yaw);
            var minX = double.PositiveInfinity; var minZ = minX; var minY = minX;
            var maxX = double.NegativeInfinity; var maxZ = maxX; var maxY = maxX;
            foreach (var p in corners)
            {
                if (double.IsNaN(p.X + p.Y + p.Z) || double.IsInfinity(p.X + p.Y + p.Z)) return null;
                var x = p.X * cos + p.Z * sin; var z = -p.X * sin + p.Z * cos;
                minX = Math.Min(minX, x); maxX = Math.Max(maxX, x);
                minZ = Math.Min(minZ, z); maxZ = Math.Max(maxZ, z);
                minY = Math.Min(minY, p.Y); maxY = Math.Max(maxY, p.Y);
            }
            var hx = (maxX - minX) * .5; var hz = (maxZ - minZ) * .5;
            if (hx <= 0 || hz <= 0 || hx > 32 || hz > 32 || maxY - minY > 64) return null;
            var cx = (minX + maxX) * .5; var cz = (minZ + maxZ) * .5;
            return new LogFootprint(new V3(cx * cos - cz * sin, (minY + maxY) * .5, cx * sin + cz * cos), hx, hz, minY, maxY, yaw);
        }
    }

    /// <summary>Transient collider IDs. Replacing a moved record removes its
    /// old cells; destroyed parents never lend identity to their new halves.</summary>
    public sealed class LogFootprintIndex
    {
        readonly ObstacleIndex broad = new ObstacleIndex();
        readonly Dictionary<int, LogFootprint> records = new Dictionary<int, LogFootprint>();
        readonly Func<int, bool> current;
        readonly Func<int, V3, V3, bool> test;
        public LogFootprintIndex(Func<int, bool> current = null) { this.current = current; test = Test; }
        public int Count => records.Count;
        public void Put(int id, LogFootprint footprint) { records[id] = footprint; broad.Put(id, footprint.Envelope); }
        public void Remove(int id) { records.Remove(id); broad.Remove(id); }
        public void Clear() { records.Clear(); broad.Clear(); }
        bool Test(int id, V3 from, V3 to) => records[id].Blocks(from, to) && (current == null || current(id));
        public bool Blocked(V3 from, V3 to) => broad.Any(from, to, test);
    }
}
