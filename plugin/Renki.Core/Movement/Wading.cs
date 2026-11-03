using System;

namespace Renki.Movement
{
    /// <summary>Supported walking in shallow water, with reserve before the
    /// game's swim threshold. Never grants swimming or unsupported movement.</summary>
    public static class Wading
    {
        public const double MaxDepth = .8, SwimOffset = .4, SwimReserve = .3, Settling = .15;

        public static double DepthLimit(double swimDepth) =>
            double.IsNaN(swimDepth) || double.IsInfinity(swimDepth) ? 0 :
            Math.Max(0, Math.Min(MaxDepth, swimDepth - SwimOffset - SwimReserve));

        public static bool Clear(V3 at, double water, double depth) =>
            depth > 0 && depth <= MaxDepth && !double.IsNaN(water) && !double.IsInfinity(water) &&
            !double.IsNaN(at.Y) && !double.IsInfinity(at.Y) && at.Y >= water - depth + Settling;

        // The native query supplies footing only after verifying a walking
        // support normal. Character's transient contact accumulator is not
        // reliable proof: the game clears it during its own physics update.
        public static bool Supported(V3 body, V3? footing, double water, double depth,
            bool grounded, bool swimming) =>
            grounded && !swimming && footing.HasValue &&
            DryTravel.PreferFooting(body.Y, footing.Value.Y, null) &&
            Clear(body, water, depth) && Clear(footing.Value, water, depth);

        public static bool Edge(V3 body, V3 footing, V3 middle, V3 to,
            double waterFrom, double waterMiddle, double waterTo, double depth) =>
            DryTravel.ContinuousSupport(body, footing, middle, to) &&
            Clear(body, waterFrom, depth) && Clear(footing, waterFrom, depth) &&
            Clear(middle, waterMiddle, depth) && Clear(to, waterTo, depth);
    }
}
