using System;

namespace Renki.Tasks
{
    /// <summary>Bounded candidate offsets, never claims of support or reach.
    /// Fixed bearings keep a narrow usable side independent of arrival angle.</summary>
    public static class InteractionApproaches
    {
        const int RelativeSpokes = 12, FixedSpokes = 24, Spokes = RelativeSpokes + FixedSpokes;
        public const int Limit = Spokes * 4;

        public static V3 Offset(int index, double arrivalAngle)
        {
            if (index < 0 || index >= Limit || double.IsNaN(arrivalAngle) || double.IsInfinity(arrivalAngle))
                throw new ArgumentOutOfRangeException(nameof(index));
            var spoke = index % Spokes;
            var radius = 1.25 + index / Spokes;
            double bearing;
            if (spoke < RelativeSpokes)
            {
                var turn = spoke % 2 == 0 ? spoke / 2 : -(spoke + 1) / 2;
                bearing = arrivalAngle + turn * Math.PI / 6;
            }
            else bearing = (spoke - RelativeSpokes) * Math.PI * 2 / FixedSpokes;
            return new V3(Math.Cos(bearing) * radius, 0, Math.Sin(bearing) * radius);
        }
    }
}
