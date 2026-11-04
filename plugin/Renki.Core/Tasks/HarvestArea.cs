using System;
using Renki.Movement;
using Renki.Protocol;

namespace Renki.Tasks
{
    /// <summary>A saved base protected in the horizontal plane at any height.</summary>
    public sealed class HarvestArea
    {
        public V3 Center;
        public double Radius;
        static bool Finite(double n) => !double.IsNaN(n) && !double.IsInfinity(n);
        public bool Valid => Finite(Center.X) && Finite(Center.Y) && Finite(Center.Z) &&
            Math.Abs(Center.X) <= 1e6 && Math.Abs(Center.Y) <= 1e6 && Math.Abs(Center.Z) <= 1e6 &&
            Finite(Radius) && Radius >= 10 && Radius <= 250;
        public bool Contains(V3 point) => V3.FlatDistance(Center, point) <= Radius;

        public static bool DryRoot(V3 point, double? water) => water.HasValue && Finite(water.Value) &&
            Finite(point.X) && Finite(point.Y) && Finite(point.Z) &&
            Math.Abs(point.X) <= 1e6 && Math.Abs(point.Y) <= 1e6 && Math.Abs(point.Z) <= 1e6 && point.Y > water.Value + .2;

        public static string DestructibleKind(string prefab)
        {
            var lower = (prefab ?? "").ToLowerInvariant();
            if (lower.Contains("stub") || lower.Contains("stump")) return Wire.ResourceStump;
            if (lower.Contains("log")) return Wire.ResourceLog;
            return Wire.ResourceTree;
        }
    }
}
