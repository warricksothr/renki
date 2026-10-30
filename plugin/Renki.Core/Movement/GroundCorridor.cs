using System;

namespace Renki.Movement
{
    /// <summary>Follow supported ground incrementally. Elevation can accumulate
    /// on a hillside; local inclination and abrupt changes determine footing.</summary>
    public static class GroundCorridor
    {
        public const double Spacing = .5, MaxIncline = 30, MaxBend = 15, MaxLength = 12;

        public static bool Check(V3 from, V3 to, Func<V3, V3, V3?> probe, out string reason)
        {
            reason = null;
            if (!Finite(from) || !Finite(to)) { reason = "invalid_ground"; return false; }
            var length = V3.FlatDistance(from, to);
            if (length > MaxLength + .001) { reason = "corridor_limit"; return false; }
            // Resolve the actual support first: a character's transform can be
            // slightly above its feet, which would distort the first grade.
            var first = probe(from, from);
            if (!first.HasValue || !Finite(first.Value)) { reason = "blocked_or_unsupported"; return false; }
            var previous = first.Value;
            var count = Math.Max(1, (int)Math.Ceiling(length / Spacing));
            double? previousAngle = null;
            for (var i = 1; i <= count; i++)
            {
                var fraction = (double)i / count;
                var desired = new V3(from.X + (to.X - from.X) * fraction, previous.Y, from.Z + (to.Z - from.Z) * fraction);
                var result = probe(previous, desired);
                if (!result.HasValue || !Finite(result.Value)) { reason = "blocked_or_unsupported"; return false; }
                var next = result.Value;
                var angle = Math.Atan2(next.Y - previous.Y, V3.FlatDistance(previous, next)) * 180 / Math.PI;
                if (Math.Abs(angle) > MaxIncline + .01) { reason = "grade"; return false; }
                if (previousAngle.HasValue && Math.Abs(angle - previousAngle.Value) > MaxBend + .01)
                { reason = "sharp_change"; return false; }
                previous = next; previousAngle = angle;
            }
            return true;
        }

        static bool Finite(V3 point) => Finite(point.X) && Finite(point.Y) && Finite(point.Z);
        static bool Finite(double value) => !double.IsNaN(value) && !double.IsInfinity(value);
    }
}
