using System;

namespace Renki.Movement
{
    /// <summary>Measure contact under the current grounded capsule when its
    /// centre ray misses a floor edge. Never supplies future walking support.</summary>
    public static class FootprintSupport
    {
        public const double Radius = .3;
        public const int Limit = 8;

        public static V3? Find(V3 at, V3 body, bool grounded, string refusal,
            Func<V3, (V3? Point, bool Saturated)> probe, out int queries, out bool saturated)
        {
            queries = 0; saturated = false;
            if (!grounded || refusal != "no_hits" || !((at - body).Length <= .05)) return null;
            V3? best = null;
            for (var i = 0; i < Limit; i++)
            {
                var angle = i * Math.PI * 2 / Limit;
                var sample = at + new V3(Math.Cos(angle) * Radius, 0, Math.Sin(angle) * Radius);
                var hit = probe(sample); queries++;
                if (hit.Saturated) { saturated = true; return null; }
                if (hit.Point.HasValue && V3.FlatDistance(at, hit.Point.Value) <= Radius + 1e-6 &&
                    DryTravel.PreferFooting(body.Y, hit.Point.Value.Y, best?.Y)) best = hit.Point;
            }
            return best;
        }
    }
}
