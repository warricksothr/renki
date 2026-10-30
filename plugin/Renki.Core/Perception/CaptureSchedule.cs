using System;

namespace Renki.Perception
{
    // Config changes never reset the hard ceiling or permit overlapping work.
    public sealed class CaptureSchedule
    {
        double last = double.NegativeInfinity;
        bool busy;
        public bool Begin(double now, double interval, bool enabled)
        {
            if (!enabled || busy || double.IsNaN(now) || double.IsInfinity(now)) return false;
            if (double.IsNaN(interval) || double.IsInfinity(interval)) interval = 5;
            if (now - last < Math.Max(1, Math.Min(300, interval))) return false;
            last = now; busy = true; return true;
        }
        public void Complete() => busy = false;
    }
}
