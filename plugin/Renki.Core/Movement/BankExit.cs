using System;

namespace Renki.Movement
{
    /// <summary>Short live recovery from a dry body inside the planning buffer.
    /// Not a route planner: every commanded step needs a fresh physical probe.</summary>
    public sealed class BankExit
    {
        public const double Seconds = 8, Distance = 3, ActualMargin = .3;
        // Complete above the waiting threshold so small settling or liquid
        // variation spends the reserve instead of immediately restarting.
        public const double WaitingMargin = DryTravel.TravelMargin + .1;
        public const double PlanningMargin = DryTravel.TravelMargin + .3;
        public const int ProbeLimit = 128;
        static readonly int[] angles = { 0, 23, -23, 45, -45, 68, -68, 90, -90, 113, -113, 135, -135, 158, -158, 180 };
        double started = double.NaN, travelled;
        V3 previous;
        V3? target;
        double closest, progressAt;
        int direction;
        public int Samples { get; private set; }
        public int Stalls { get; private set; }
        public bool Refused { get; private set; }
        public string RefusalReason { get; private set; }

        // Finish above the entry threshold, accounting for body settling.
        // Rounded support at exactly the planning margin must not alternate
        // recovery/route planning every frame and spend the escape budget idle.
        public static bool Recovered(V3 body, V3 footing, double water, double groundNormalY = 1) =>
            DryTravel.WalkableNormal(groundNormalY) &&
            DryTravel.PreferFooting(body.Y, footing.Y, null) &&
            DryTravel.Clear(body, water, PlanningMargin) &&
            DryTravel.Clear(footing, water, PlanningMargin);

        public static bool NeedsDeparture(V3 body, V3 footing, double water, bool waitingForRoute) =>
            !DryTravel.Clear(body, water, waitingForRoute ? WaitingMargin : DryTravel.TravelMargin) ||
            !DryTravel.Clear(footing, water, waitingForRoute ? WaitingMargin : DryTravel.TravelMargin);

        void Refuse(string reason) { Refused = true; RefusalReason = reason; }

        public V3? Next(V3 from, V3 toward, double now, Func<V3, V3, V3?> probe)
        {
            if (Refused) return null;
            if (double.IsNaN(started)) { started = now; previous = from; }
            travelled += V3.FlatDistance(previous, from); previous = from;
            if (now - started >= Seconds) { Refuse("time_limit"); return null; }
            if (travelled >= Distance) { Refuse("distance_limit"); return null; }
            if (Samples >= ProbeLimit) { Refuse("probe_limit"); return null; }
            if (target.HasValue && V3.FlatDistance(from, target.Value) <= .12)
            { target = null; direction = 0; }
            if (target.HasValue)
            {
                var remaining = V3.FlatDistance(from, target.Value);
                if (remaining <= closest - .03) { closest = remaining; progressAt = now; }
                // A clear probe does not prove the body can execute the step.
                // Continue the bearing sweep after one second without3cm of
                // forward progress; bobbing cannot renew the original bounds.
                if (now - progressAt >= 1) { target = null; Stalls++; }
            }
            if (target.HasValue)
            {
                // Live execution is checked even while holding one short goal.
                var checkedTarget = probe(from, target.Value);
                if (checkedTarget.HasValue) return checkedTarget;
                target = null;
            }
            var heading = (toward - from).Flat.Normalized;
            if (heading.Length < .5) heading = V3.Forward;
            for (var n = 0; n < 4 && direction < angles.Length && Samples < ProbeLimit; n++)
            {
                var angle = angles[direction++] * Math.PI / 180;
                // Native uphill ray hits can lie centimetres beyond the requested
                // point. Leave room inside the unchanged physical step bound.
                var step = new V3(heading.X * Math.Cos(angle) - heading.Z * Math.Sin(angle), 0,
                    heading.X * Math.Sin(angle) + heading.Z * Math.Cos(angle)) * .45;
                Samples++;
                target = probe(from, from + step);
                if (target.HasValue)
                {
                    closest = V3.FlatDistance(from, target.Value); progressAt = now;
                    return target;
                }
            }
            if (Samples >= ProbeLimit) Refuse("probe_limit");
            else if (direction == angles.Length) Refuse("bearings_exhausted");
            return null;
        }

        public static bool Clear(V3 body, V3 footing, V3 middle, V3 to,
            double waterFrom, double waterMiddle, double waterTo)
        {
            if (V3.FlatDistance(body, to) > .501 || !DryTravel.PreferFooting(body.Y, footing.Y, null) ||
                !DryTravel.ContinuousSupport(body, footing, middle, to)) return false;
            // Account for the measured body/footing offset and an extra 10cm
            // settling allowance. A recovery may never wade or descend.
            var offset = Math.Max(0, footing.Y - body.Y) + .1;
            var before = Math.Min(body.Y, footing.Y) - .1 - waterFrom;
            var halfway = middle.Y - offset - waterMiddle;
            var after = to.Y - offset - waterTo;
            return DryTravel.Clear(body, waterFrom, ActualMargin) &&
                DryTravel.Clear(footing, waterFrom, ActualMargin) &&
                DryTravel.Clear(middle, waterMiddle, ActualMargin + offset) &&
                DryTravel.Clear(to, waterTo, ActualMargin + offset) &&
                middle.Y >= footing.Y && to.Y >= middle.Y &&
                halfway >= before && after >= before;
        }
    }
}
