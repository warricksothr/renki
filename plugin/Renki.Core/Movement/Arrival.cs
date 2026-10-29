using System;

namespace Renki.Movement
{
    /// <summary>
    /// Whether the bot has reached a goal: near it across the ground and on
    /// its level. Across the ground alone, a bot under a raised floor is
    /// "home" when home is on the floor above it.
    /// </summary>
    public static class Arrival
    {
        /// <summary>How far above or below the goal still counts: less than
        /// a storey (2 m), more than the lie of the ground in a stride.</summary>
        public const double Level = 1.75;

        public static bool Reached(V3 pos, V3 goal, double within) =>
            V3.FlatDistance(pos, goal) <= within && Math.Abs(pos.Y - goal.Y) <= Level;
    }
}
