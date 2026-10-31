using System;

namespace Renki.Movement
{
    /// <summary>Follow supported ground incrementally. Elevation can accumulate
    /// on a hillside; local inclination and abrupt changes determine footing.</summary>
    public static class GroundCorridor
    {
        public const double Spacing = .5, MaxIncline = 30, MaxBend = 15, MaxLength = 12;
        public const double MaxRoughness = .15;

        public sealed class Failure
        {
            public readonly string Reason;
            public readonly int Segment;
            public readonly V3 From, Desired;
            public readonly V3? Supported;
            public readonly double? Angle, PreviousAngle, Roughness;

            internal Failure(string reason, int segment, V3 from, V3 desired, V3? supported = null,
                double? angle = null, double? previousAngle = null, double? roughness = null)
            {
                Reason = reason; Segment = segment; From = from; Desired = desired;
                Supported = supported; Angle = angle; PreviousAngle = previousAngle;
                Roughness = roughness;
            }
        }

        public static bool Check(V3 from, V3 to, Func<V3, V3, V3?> probe, out string reason)
        {
            var failure = Inspect(from, to, probe);
            reason = failure?.Reason;
            return failure == null;
        }

        public static Failure Inspect(V3 from, V3 to, Func<V3, V3, V3?> probe) => new Cursor().Inspect(from, to, probe);

        /// <summary>Continuous ground history across route corners. Fork for
        /// each search branch; zero-length corners never clear its grade.</summary>
        public sealed class Cursor
        {
            public V3? At { get; private set; }
            public double? LastAngle { get; private set; }
            public int Steps { get; private set; }

            public Cursor Fork() => new Cursor { At = At, LastAngle = LastAngle, Steps = Steps };

            public Failure Inspect(V3 from, V3 to, Func<V3, V3, V3?> probe)
            {
                if (!Finite(from) || !Finite(to)) return new Failure("invalid_ground", 0, from, to);
                var length = V3.FlatDistance(from, to);
                if (length > MaxLength + .001) return new Failure("corridor_limit", 0, from, to);
                // Resolve the actual support first: a character's transform can be
                // slightly above its feet, which would distort the first grade.
                var first = probe(from, from);
                if (!first.HasValue || !Finite(first.Value)) return new Failure("blocked_or_unsupported", 0, from, from);
                if (At.HasValue && (first.Value - At.Value).Length > .05)
                    return new Failure("ground_changed", Steps, At.Value, from, first);
                var previous = first.Value;
                At = previous;
                if (length < 1e-9) return null;
                var count = Math.Max(1, (int)Math.Ceiling(length / Spacing));
                for (var i = 1; i <= count; i++)
                {
                    Steps++;
                    var fraction = (double)i / count;
                    var desired = new V3(from.X + (to.X - from.X) * fraction, previous.Y, from.Z + (to.Z - from.Z) * fraction);
                    var result = probe(previous, desired);
                    if (!result.HasValue || !Finite(result.Value)) return new Failure("blocked_or_unsupported", Steps, previous, desired);
                    var next = result.Value;
                    var run = V3.FlatDistance(previous, next);
                    var rise = next.Y - previous.Y;
                    var angle = Math.Atan2(rise, run) * 180 / Math.PI;
                    if (Math.Abs(angle) > MaxIncline + .01) return new Failure("grade", Steps, previous, desired, next, angle, LastAngle);
                    // Terrain triangles can meet at a noticeable angle while the
                    // actual rise change is small. Reject physically sharp joins,
                    // not ordinary faceting of otherwise walkable ground.
                    var roughness = LastAngle.HasValue ? Math.Abs(rise - Math.Tan(LastAngle.Value * Math.PI / 180) * run) : 0;
                    if (LastAngle.HasValue && Math.Abs(angle - LastAngle.Value) > MaxBend + .01 && roughness > MaxRoughness + .00001)
                        return new Failure("sharp_change", Steps, previous, desired, next, angle, LastAngle, roughness);
                    previous = next; At = next; LastAngle = angle;
                }
                return null;
            }
        }

        static bool Finite(V3 point) => Finite(point.X) && Finite(point.Y) && Finite(point.Z);
        static bool Finite(double value) => !double.IsNaN(value) && !double.IsInfinity(value);
    }
}
