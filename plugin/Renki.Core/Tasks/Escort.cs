using System;

namespace Renki.Tasks
{
    public enum EscortAction { Follow, Collect, Defend, Escape, Wait }

    /// <summary>Bounded companion decisions; proximity alone never authorizes a fight.</summary>
    public static class EscortPolicy
    {
        public const double Leash = 10, DetourSeconds = 12, FightLeash = 16;

        public static EscortAction Decide(bool leaderVisible, double gap, bool leaderStill,
            int threats, bool eligible, bool ready, bool defend, bool collect, bool hasSpace,
            bool detour, double detourAge)
        {
            if (threats > 0)
                return defend && leaderVisible && gap <= FightLeash && threats == 1 && eligible && ready
                    ? EscortAction.Defend : EscortAction.Escape;
            if (!leaderVisible) return EscortAction.Wait;
            if (gap > Leash || !leaderStill || detour && detourAge >= DetourSeconds) return EscortAction.Follow;
            return collect && hasSpace && gap <= 6 ? EscortAction.Collect : EscortAction.Follow;
        }

        public static bool Ready(double hp, double max, double stamina, double armour, bool food, bool weapon) =>
            hp >= 30 && max > 0 && hp >= max * .8 && stamina >= 35 && armour >= 2 && food && weapon;

        // Melee exemption is limited to named allies with known PvP protection.
        public static bool SafeAlly(bool protectedPlayer, bool? ownPvp, bool? otherPvp) =>
            protectedPlayer && !HarvestSafety.PlayerBlocks(false, ownPvp, otherPvp);
    }
}
