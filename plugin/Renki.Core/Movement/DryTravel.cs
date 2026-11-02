using System;

namespace Renki.Movement
{
    /// <summary>Dry travel uses measured support, including bridges/floors,
    /// rather than the terrain below them or navmesh hints over water.</summary>
    public static class DryTravel
    {
        public const double Margin = .2;
        // Leave room for the body's step/slope offset below a sampled floor.
        public const double TravelMargin = Margin + .6;
        // Valheim players slide above38 degrees. Reserve a small normal
        // margin for native contact smoothing; height alone is insufficient.
        // Native float normals on rotated flat floors can exceed one by
        // a few ulps. Keep the lower slope bound exact; tolerate only the
        // numerical overshoot at the upper, flat-surface bound.
        public static bool WalkableNormal(double y) => y >= .8 && y <= 1 + 1e-6;

        // A full grid edge may span two ordinary stair treads. Its midpoint
        // must share the rise; a half-metre floor lip is not a staircase.
        public static bool ContinuousSupport(V3 from, V3 middle, V3 to) =>
            Math.Abs(middle.Y - from.Y) <= .3 + 1e-6 && Math.Abs(to.Y - middle.Y) <= .3 + 1e-6;

        // The body's capsule can stand above the surface on a slope or log.
        // Compare surface heights on all three samples. A normal stair tread
        // can overlap the foot during stepping; a higher floor cannot.
        public static bool PreferFooting(double bodyY, double candidateY, double? previousY) =>
            candidateY <= bodyY + .3 + 1e-6 && bodyY - candidateY <= .6 + 1e-6 &&
            (!previousY.HasValue || Math.Abs(candidateY - bodyY) < Math.Abs(previousY.Value - bodyY));
        public static bool ContinuousSupport(V3 body, V3 supportedFrom, V3 middle, V3 to) =>
            PreferFooting(body.Y, supportedFrom.Y, null) &&
            ContinuousSupport(supportedFrom, middle, to);
        // Do not probe through an obstacle beyond a checked short corner.
        // Strafing leaves that route, so it retains the full lookahead.
        public static double ClearanceDistance(V3 from, V3? next, bool strafing = false) =>
            !strafing && next.HasValue ? Math.Min(.5, V3.FlatDistance(from, next.Value)) : .5;

        // A distant geometric shore hint cannot authorize the intervening
        // movement. The caller supplies the full live dry/hostile/body probe.
        public static V3? ShoreStep(V3 from, V3 shore, Func<V3, V3, V3?> probe)
        {
            var delta = (shore - from).Flat;
            if (delta.Length <= 1e-6) return null;
            return Probe(from, from + delta.Normalized * Math.Min(.5, delta.Length), probe);
        }
        // The body checks a half-metre flat lookahead while following a
        // route. A longer diagonal planning edge must validate that exact
        // prefix too: its midpoint and endpoint can miss a support spike.
        public static V3? Probe(V3 from, V3 desired, Func<V3, V3, V3?> physical)
        {
            var result = physical(from, desired);
            if (!result.HasValue) return null;
            var delta = (desired - from).Flat;
            if (delta.Length > .5 + 1e-6 && !physical(from, from + delta.Normalized * .5).HasValue) return null;
            return result;
        }
        public static bool Clear(V3 support, double water, double margin = Margin) =>
            !double.IsNaN(support.Y) && !double.IsInfinity(support.Y) &&
            !double.IsNaN(water) && !double.IsInfinity(water) && support.Y >= water + margin;
    }
}
