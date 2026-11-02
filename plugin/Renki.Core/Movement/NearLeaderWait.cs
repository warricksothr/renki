using System;

namespace Renki.Movement
{
    /// <summary>
    /// What a follower does once the walker has spent its recoveries
    /// (docs/BRIDGE-PROTOCOL.md 7.3). Within the follow distance plus Margin
    /// of its leader, across the ground, the bot is with them but cannot get
    /// closer, for example outside the wall of the house they stand in. It
    /// stands and waits rather than failing <c>stuck</c>, and walks again
    /// once the leader has moved more than Resume from where they stood when
    /// the wait began, or the bot itself has been pushed more than Resume
    /// from where it stood.
    /// Farther away, or a floor above or below, spent recoveries are still a
    /// failure: a leader upstairs is reached by finding the stairs, which a
    /// fresh attempt may do and standing still never will.
    /// </summary>
    public sealed class NearLeaderWait
    {
        /// <summary>How far beyond the follow distance still counts as near.</summary>
        public const double Margin = 1.5;

        /// <summary>How far the leader, or the bot, moves before the bot
        /// tries again: the follower's re-path distance.</summary>
        public const double Resume = 2.0;

        V3? anchor;
        V3 spot;

        public bool Waiting => anchor.HasValue;

        /// <summary>The walker gave up at pos. True: wait here. False: the
        /// leader is too far, or on another level, for waiting, so the follow
        /// has failed.</summary>
        public bool GiveUp(V3 pos, V3 leader, double distance)
        {
            if (V3.FlatDistance(pos, leader) > distance + Margin || Math.Abs(pos.Y - leader.Y) > Arrival.Level)
            {
                anchor = null;
                return false;
            }
            anchor = leader;
            spot = pos;
            return true;
        }

        /// <summary>Whether to go on waiting at pos, given where the leader
        /// is now. Once either has moved more than Resume from where they
        /// stood when the wait began (the leader walking off, the bot knocked
        /// back), the wait ends and this returns false. Each is measured from
        /// its own start, so the leader moving about the room, or the bot
        /// settling a few centimetres at the edge of the near ring, does not
        /// end the wait.</summary>
        public bool Hold(V3 pos, V3 leader)
        {
            if (anchor is V3 at && (leader - at).Length <= Resume && (pos - spot).Length <= Resume) return true;
            anchor = null;
            return false;
        }

        public void Clear() => anchor = null;
    }
}
