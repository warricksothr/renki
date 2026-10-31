using System;

namespace Renki.Movement
{
    /// <summary>Dry travel uses measured support, including bridges/floors,
    /// rather than the terrain below them or navmesh hints over water.</summary>
    public static class DryTravel
    {
        public const double Margin = .2;
        public static bool Clear(V3 support, double water) =>
            !double.IsNaN(support.Y) && !double.IsInfinity(support.Y) &&
            !double.IsNaN(water) && !double.IsInfinity(water) && support.Y >= water + Margin;
    }
}
