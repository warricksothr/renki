using System;

namespace Renki.Movement
{
    /// <summary>Room to fight near the center, with at least one supported way out.
    /// Actual dodges and retreats still validate their chosen direction at runtime.</summary>
    public static class FightingGround
    {
        public const double StanceRadius = 3, EscapeLength = 6;

        public static GroundCorridor.Failure Inspect(V3 center, Func<V3, V3?> project,
            Func<V3, V3, V3?> probe, out string region, out V3? escape)
        {
            escape = null;
            region = "stance";
            for (var i = 0; i < 4; i++)
            {
                var radius = Direction(i) * StanceRadius;
                var edge = center - radius;
                var support = project(edge);
                if (!support.HasValue) return new GroundCorridor.Failure("wet_or_unloaded_terrain", 0, edge, edge);
                // Complete diameters preserve the checks for sharp crests/dips
                // at the center, which separate outward rays would miss.
                var failure = GroundCorridor.Inspect(support.Value, center + radius, probe);
                if (failure != null) return failure;
            }
            region = "escape";
            GroundCorridor.Failure last = null;
            for (var i = 0; i < 8; i++)
            {
                var end = center + Direction(i) * EscapeLength;
                var support = project(end);
                if (!support.HasValue)
                {
                    last = new GroundCorridor.Failure("wet_or_unloaded_terrain", 0, center, end);
                    continue;
                }
                last = GroundCorridor.Inspect(center, support.Value, probe);
                if (last != null) continue;
                escape = support;
                region = null;
                return null;
            }
            return last;
        }

        static V3 Direction(int i) => new V3(Math.Cos(i * Math.PI / 4), 0, Math.Sin(i * Math.PI / 4));
    }
}
