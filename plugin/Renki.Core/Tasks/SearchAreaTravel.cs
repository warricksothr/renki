using System;

namespace Renki.Tasks
{
    /// <summary>Travel to one selected search area precedes its local search.
    /// These are finite travel bounds, not permission to skip physical probes.</summary>
    public static class SearchAreaTravel
    {
        public const double Range = 140, Seconds = 90, Travel = 200, EntryRadius = 8;
        public const int Attempts = 5;

        public static V3 Candidate(V3 center, int attempt)
        {
            if (attempt < 0 || attempt >= Attempts) throw new ArgumentOutOfRangeException(nameof(attempt));
            if (attempt == 0) return center;
            var angle = (attempt - 1) * Math.PI / 2;
            return center + new V3(Math.Cos(angle), 0, Math.Sin(angle)) * EntryRadius;
        }

        public static bool WithinBudget(V3 departure, V3 bot, double elapsed, double travelled) =>
            Finite(elapsed) && Finite(travelled) && elapsed >= 0 && elapsed < Seconds && travelled >= 0 && travelled <= Travel &&
            Finite(bot.X) && Finite(bot.Y) && Finite(bot.Z) && V3.FlatDistance(departure, bot) <= Range + EntryRadius;

        public static V3 LocalOrigin(V3 requested, V3 arrival) => new V3(requested.X, arrival.Y, requested.Z);
        public static bool WithinScope(V3 anchor, double limit, V3 bot) =>
            Finite(limit) && limit > 0 && Finite(bot.X) && Finite(bot.Y) && Finite(bot.Z) && (bot - anchor).Length <= limit;
        static bool Finite(double n) => !double.IsNaN(n) && !double.IsInfinity(n);
    }
}
