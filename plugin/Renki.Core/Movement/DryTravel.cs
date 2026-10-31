using System;

namespace Renki.Movement
{
    /// <summary>Dry travel uses measured support, including bridges/floors,
    /// rather than the terrain below them or navmesh hints over water.</summary>
    public static class DryTravel
    {
        public const double Margin = .2;
        // Do not probe through an obstacle beyond a checked short corner.
        // Strafing leaves that route, so it retains the full lookahead.
        public static double ClearanceDistance(V3 from, V3? next, bool strafing = false) =>
            !strafing && next.HasValue ? Math.Min(.5, V3.FlatDistance(from, next.Value)) : .5;
        public static bool Clear(V3 support, double water) =>
            !double.IsNaN(support.Y) && !double.IsInfinity(support.Y) &&
            !double.IsNaN(water) && !double.IsInfinity(water) && support.Y >= water + Margin;
    }
}
