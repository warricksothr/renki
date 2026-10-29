using System;

namespace Renki.Movement
{
    /// <summary>
    /// Where a fleeing bot runs (docs/BRIDGE-PROTOCOL.md 7.2): toward a place
    /// of safety it was given, such as its owner or home, unless that lies
    /// past the threat; otherwise straight away from the threat.
    /// </summary>
    public static class Flee
    {
        /// <summary>Within this angle of the threat's bearing, a place lies
        /// past it, and running there would run into it.</summary>
        public const double PastThreatDegrees = 60;

        /// <summary>How much farther than the distance asked the bot aims
        /// when it runs straight away, so it does not stop on the line.</summary>
        public const double Beyond = 5;

        /// <summary>Where to run from pos, away from threat to at least
        /// distance, by way of toward when it does not lie past the threat.
        /// Y is pos's: the walker finds the ground.</summary>
        public static V3 Goal(V3 pos, V3 threat, V3? toward, double distance)
        {
            var away = (pos - threat).Flat.Normalized;
            // On top of the threat: any way out is as good as another.
            if (away.Length < 0.5) away = V3.Forward;
            if (toward is V3 safe && !PastThreat(pos, threat, safe)) return safe;
            var run = Math.Max(distance - V3.FlatDistance(pos, threat), 0) + Beyond;
            var goal = pos + away * run;
            return new V3(goal.X, pos.Y, goal.Z);
        }

        /// <summary>Whether a place lies past the threat, as seen from pos:
        /// its bearing within PastThreatDegrees of the threat's.</summary>
        public static bool PastThreat(V3 pos, V3 threat, V3 place)
        {
            var toThreat = (threat - pos).Flat.Normalized;
            var toPlace = (place - pos).Flat.Normalized;
            if (toThreat.Length < 0.5 || toPlace.Length < 0.5) return false;
            return V3.Dot(toThreat, toPlace) > Math.Cos(PastThreatDegrees * Math.PI / 180);
        }
    }
}
