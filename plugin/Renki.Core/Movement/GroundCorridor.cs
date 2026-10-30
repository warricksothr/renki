using System;

namespace Renki.Movement
{
    /// <summary>Follow supported ground incrementally. Elevation can accumulate
    /// on a hillside; local inclination and abrupt changes determine footing.</summary>
    public static class GroundCorridor
    {
        public const double Spacing = .5, MaxIncline = 30, MaxBend = 15, MaxLength = 12;

        public sealed class Failure
        {
            public readonly string Reason;
            public readonly int Segment;
            public readonly V3 From, Desired;
            public readonly V3? Supported;
            public readonly double? Angle, PreviousAngle;

            internal Failure(string reason, int segment, V3 from, V3 desired, V3? supported = null,
                double? angle = null, double? previousAngle = null)
            {
                Reason = reason; Segment = segment; From = from; Desired = desired;
                Supported = supported; Angle = angle; PreviousAngle = previousAngle;
            }
        }

        public static bool Check(V3 from, V3 to, Func<V3, V3, V3?> probe, out string reason)
        {
            var failure = Inspect(from, to, probe);
            reason = failure?.Reason;
            return failure == null;
        }

        public static Failure Inspect(V3 from, V3 to, Func<V3, V3, V3?> probe)
        {
            if (!Finite(from) || !Finite(to)) return new Failure("invalid_ground", 0, from, to);
            var length = V3.FlatDistance(from, to);
            if (length > MaxLength + .001) return new Failure("corridor_limit", 0, from, to);
            // Resolve the actual support first: a character's transform can be
            // slightly above its feet, which would distort the first grade.
            var first = probe(from, from);
            if (!first.HasValue || !Finite(first.Value)) return new Failure("blocked_or_unsupported", 0, from, from);
            var previous = first.Value;
            var count = Math.Max(1, (int)Math.Ceiling(length / Spacing));
            double? previousAngle = null;
            for (var i = 1; i <= count; i++)
            {
                var fraction = (double)i / count;
                var desired = new V3(from.X + (to.X - from.X) * fraction, previous.Y, from.Z + (to.Z - from.Z) * fraction);
                var result = probe(previous, desired);
                if (!result.HasValue || !Finite(result.Value)) return new Failure("blocked_or_unsupported", i, previous, desired);
                var next = result.Value;
                var angle = Math.Atan2(next.Y - previous.Y, V3.FlatDistance(previous, next)) * 180 / Math.PI;
                if (Math.Abs(angle) > MaxIncline + .01) return new Failure("grade", i, previous, desired, next, angle, previousAngle);
                if (previousAngle.HasValue && Math.Abs(angle - previousAngle.Value) > MaxBend + .01)
                    return new Failure("sharp_change", i, previous, desired, next, angle, previousAngle);
                previous = next; previousAngle = angle;
            }
            return null;
        }

        static bool Finite(V3 point) => Finite(point.X) && Finite(point.Y) && Finite(point.Z);
        static bool Finite(double value) => !double.IsNaN(value) && !double.IsInfinity(value);
    }
}
