using System;

namespace Renki.Movement
{
    public enum DoorCleanup { Wait, Close, Closed, OpenRefused, CloseRefused, Abandoned }

    /// <summary>One leaf opened by the bot, retained until closure is observed.
    /// The doorway's fixed forward normal works for either hinge of a gate.</summary>
    public sealed class DoorPassage
    {
        public const double SettleFor = 3;
        public const double AbandonReach = 8;
        const double ClearPlane = 1.5, LeftAt = 4.5;
        readonly double startingSide;
        double requestedAt;
        bool through, seenOpen;
        int closeAttempts;

        public DoorPassage(V3 start, V3 door, V3 normal, double now)
        {
            startingSide = Side(start, door, normal);
            requestedAt = now;
        }

        public static bool TrackOpening(bool enabled, bool wasOpen, bool accepted, bool hold) =>
            enabled && !wasOpen && accepted && !hold;

        public bool Settling(double now) => now - requestedAt < SettleFor;

        public void Closing(double now)
        {
            requestedAt = now;
            closeAttempts++;
        }

        public DoorCleanup Observe(V3 pos, V3 door, V3 normal, bool open, bool canClose, double now)
        {
            var away = V3.FlatDistance(pos, door);
            var level = Math.Abs(pos.Y - door.Y) <= DoorRules.Level;
            if (closeAttempts > 0)
            {
                if (!open) return DoorCleanup.Closed;
                if (Settling(now)) return DoorCleanup.Wait;
                if (!canClose || !level || away > AbandonReach) return DoorCleanup.Abandoned;
                return closeAttempts < 2 ? DoorCleanup.Close : DoorCleanup.CloseRefused;
            }
            if (!open)
                return !seenOpen && Settling(now) ? DoorCleanup.Wait
                    : seenOpen ? DoorCleanup.Closed : DoorCleanup.OpenRefused;

            seenOpen = true;
            if (DoorRules.In(pos, door)) through = true;
            if (Settling(now)) return DoorCleanup.Wait;
            // Close after clearing the actual doorway, including an off-centre
            // double gate. Proximity to either hinge alone is not passage.
            var side = Side(pos, door, normal);
            var crossed = Math.Abs(startingSide) > .1 && startingSide * side < 0 &&
                Math.Abs(side) >= ClearPlane && away <= DoorRules.StuckReach;
            if (!level || !(crossed || (through ? DoorRules.Past(pos, door) : away > LeftAt)))
                return away > AbandonReach ? DoorCleanup.Abandoned : DoorCleanup.Wait;
            return canClose && away <= AbandonReach ? DoorCleanup.Close : DoorCleanup.Abandoned;
        }

        static double Side(V3 pos, V3 door, V3 normal)
        {
            var length = Math.Sqrt(normal.X * normal.X + normal.Z * normal.Z);
            return length < 1e-6 ? 0 : ((pos.X - door.X) * normal.X + (pos.Z - door.Z) * normal.Z) / length;
        }
    }
}
