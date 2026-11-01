using System;

namespace Renki.Movement
{
    /// <summary>Short live recovery from a dry body inside the planning buffer.
    /// Not a route planner: every commanded step needs a fresh physical probe.</summary>
    public sealed class BankExit
    {
        public const double Seconds = 8, Distance = 3, ActualMargin = .3;
        public const int ProbeLimit = 128;
        static readonly int[] angles = { 0, 23, -23, 45, -45, 68, -68, 90, -90, 113, -113, 135, -135, 158, -158, 180 };
        double started = double.NaN, travelled;
        V3 previous;
        V3? target;
        int direction;
        public int Samples { get; private set; }
        public bool Refused { get; private set; }

        public V3? Next(V3 from, V3 toward, double now, Func<V3, V3, V3?> probe)
        {
            if (Refused) return null;
            if (double.IsNaN(started)) { started = now; previous = from; }
            travelled += V3.FlatDistance(previous, from); previous = from;
            if (now - started >= Seconds || travelled >= Distance || Samples >= ProbeLimit)
            { Refused = true; return null; }
            if (target.HasValue && V3.FlatDistance(from, target.Value) <= .12)
            { target = null; direction = 0; }
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
                var step = new V3(heading.X * Math.Cos(angle) - heading.Z * Math.Sin(angle), 0,
                    heading.X * Math.Sin(angle) + heading.Z * Math.Cos(angle)) * .5;
                Samples++;
                target = probe(from, from + step);
                if (target.HasValue) return target;
            }
            if (direction == angles.Length || Samples >= ProbeLimit) Refused = true;
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
            var before = Math.Min(body.Y, footing.Y) - waterFrom;
            var halfway = middle.Y - offset - waterMiddle;
            var after = to.Y - offset - waterTo;
            return DryTravel.Clear(body, waterFrom, ActualMargin) &&
                DryTravel.Clear(footing, waterFrom, ActualMargin) &&
                DryTravel.Clear(middle, waterMiddle, ActualMargin + offset) &&
                DryTravel.Clear(to, waterTo, ActualMargin + offset) &&
                middle.Y >= footing.Y && to.Y >= middle.Y &&
                halfway >= before && after > before + .001;
        }
    }
}
