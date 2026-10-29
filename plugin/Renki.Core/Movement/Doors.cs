using System;

namespace Renki.Movement
{
    /// <summary>
    /// When the walker opens a door and when it shuts one behind it, kept
    /// apart from the game so it can be tested. Distances are on the ground
    /// plane; a door counts from its centre.
    /// </summary>
    public static class DoorRules
    {
        /// <summary>How close a door must be to open it while walking.</summary>
        public const double Reach = 2.5;

        /// <summary>How close a door must be to open it when stuck, in any direction.</summary>
        public const double StuckReach = 4;

        /// <summary>How far past a door the bot shuts it again.</summary>
        public const double ShutPast = 3.5;

        /// <summary>How close the bot must come to a door it opened before
        /// leaving it counts as going through: a door opened from a few
        /// metres away is not passed until the bot has been in it.</summary>
        public const double Through = 1.5;

        /// <summary>Half the angle, in degrees, in front of the bot that counts as ahead.</summary>
        public const double Cone = 70;

        /// <summary>The door is within Reach and in front of the way the bot is going.</summary>
        public static bool Ahead(V3 pos, V3 heading, V3 door)
        {
            var to = new V3(door.X - pos.X, 0, door.Z - pos.Z);
            var d = Math.Sqrt(to.X * to.X + to.Z * to.Z);
            if (d > Reach) return false;
            if (d < 0.5) return true; // in the doorway
            var h = Math.Sqrt(heading.X * heading.X + heading.Z * heading.Z);
            if (h < 1e-6) return false;
            var cos = (to.X * heading.X + to.Z * heading.Z) / (d * h);
            return cos >= Math.Cos(Cone * Math.PI / 180);
        }

        /// <summary>How far above or below a door still counts as level with it.</summary>
        public const double Level = 2;

        /// <summary>The bot is in the doorway of a door it opened: close to it
        /// and level with it, not on a floor above or below.</summary>
        public static bool In(V3 pos, V3 door) =>
            V3.FlatDistance(pos, door) <= Through && Math.Abs(pos.Y - door.Y) <= Level;

        /// <summary>The bot, having been in the doorway, is far enough past
        /// the door to shut it.</summary>
        public static bool Past(V3 pos, V3 door) => V3.FlatDistance(pos, door) > ShutPast;
    }
}
