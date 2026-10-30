using System;

namespace Renki.Tasks
{
    public enum MeleeAction { Approach, Strike, Block, Dodge, Wait, Retreat, Lost, Won }

    // Values come from the live body each step, never inferred from missing AI.
    public sealed class MeleeFrame
    {
        public bool Valid, Dead, Eligible, Weapon, Unsafe, Busy, InRange, Incoming, CanDodge, CanBlock;
        public double Hp, HpMax, Stamina, AttackCost, DodgeCost, BlockCost, DistanceFromStart, TargetHp;
    }

    /// <summary>One conservative encounter, with no autonomous target selection.</summary>
    public sealed class MeleeEncounter
    {
        public const double HealthFraction = .8, EscapeReserve = 10, Leash = 12, WorkLimit = 45, NoProgress = 12;
        double started, progressAt, bestHealth;
        public string Reason { get; private set; }

        public MeleeEncounter(double now, double targetHealth)
        {
            started = progressAt = now; bestHealth = targetHealth;
        }

        public static bool Eligible(string prefab, int level, bool player, bool tamed, bool hostile) =>
            prefab == "Greyling" && level == 1 && !player && !tamed && hostile;

        public MeleeAction Decide(MeleeFrame f, double now)
        {
            Reason = null;
            if (!f.Valid) { Reason = "target is no longer loaded"; return MeleeAction.Lost; }
            if (!f.Eligible) return Retreat("target is no longer an eligible hostile");
            if (f.Dead) return MeleeAction.Won;
            if (!f.Weapon) return Retreat("no usable melee weapon");
            if (f.Unsafe) return Retreat("another creature, bystander or unsafe footing is nearby");
            if (!(f.HpMax > 0) || f.Hp < f.HpMax * HealthFraction) return Retreat("health below combat threshold");
            if (f.DistanceFromStart > Leash) return Retreat("encounter exceeded its pursuit leash");
            if (now - started >= WorkLimit) return Retreat("combat time limit");
            if (f.TargetHp < bestHealth - .01) { bestHealth = f.TargetHp; progressAt = now; }
            if (now - progressAt >= NoProgress) return Retreat("no observed progress against target");
            if (f.Stamina < EscapeReserve) return Retreat("stamina needed for escape");
            if (f.Busy) return MeleeAction.Wait; // Do not queue a dodge or chain swing.
            if (f.Incoming)
            {
                if (f.CanDodge && f.Stamina >= f.DodgeCost + EscapeReserve) return MeleeAction.Dodge;
                if (f.CanBlock && f.Stamina >= f.BlockCost + EscapeReserve) return MeleeAction.Block;
                return Retreat("cannot afford defense and escape");
            }
            if (!f.InRange) return MeleeAction.Approach;
            if (f.Stamina >= f.AttackCost + Math.Max(f.BlockCost, f.DodgeCost) + EscapeReserve)
                return MeleeAction.Strike;
            // Recover without holding block, which impedes regeneration.
            return MeleeAction.Wait;
        }

        MeleeAction Retreat(string reason) { Reason = reason; return MeleeAction.Retreat; }
    }
}
