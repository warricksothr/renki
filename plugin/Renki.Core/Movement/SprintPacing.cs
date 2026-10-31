using System;

namespace Renki.Movement
{
    /// <summary>Walking regenerates a useful buffer before another sprint.</summary>
    public sealed class SprintPacing
    {
        bool recovering;
        public bool Run(bool requested, double stamina, double maximum)
        {
            if (!requested) return false;
            if (!(maximum > 0) || double.IsNaN(stamina) || double.IsInfinity(stamina)) return false;
            if (stamina <= Math.Max(10, maximum * .2)) recovering = true;
            if (recovering && stamina >= Math.Max(20, maximum * .65)) recovering = false;
            return !recovering;
        }
    }
}
