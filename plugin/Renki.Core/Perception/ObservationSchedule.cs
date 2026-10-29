using System;

namespace Renki.Perception
{
    /// <summary>State establishes the scope of subsequent world observations.
    /// A new phase/world/player must publish immediately, even between periodic
    /// state ticks; otherwise a controller can discard the first inventory.</summary>
    public sealed class ObservationSchedule
    {
        double next = double.NegativeInfinity;
        string phase, world;
        object player;

        public void Reset() => next = double.NegativeInfinity;

        public bool Due(double now, string currentPhase, string currentWorld, object currentPlayer, double interval, out bool changed)
        {
            changed = phase != currentPhase || world != currentWorld || !ReferenceEquals(player, currentPlayer);
            if (!changed && now < next) return false;
            phase = currentPhase; world = currentWorld; player = currentPlayer;
            next = now + Math.Max(0, interval);
            return true;
        }
    }
}
