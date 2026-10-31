using System;

namespace Renki.Tasks
{
    public enum MeleeAction { Closing, Approach, Strike, Block, Dodge, Wait, Retreat, Lost, Won }

    // Values come from the live body each step, never inferred from missing AI.
    public sealed class MeleeFrame
    {
        public bool Valid, Dead, Eligible, Weapon, Unsafe, Busy, InRange, Incoming, CanDodge, CanBlock;
        public double Hp, HpMax, Stamina, AttackCost, DodgeCost, BlockCost, DistanceFromStart, TargetHp, TargetDistance;
    }

    /// <summary>One conservative encounter, with no autonomous target selection.</summary>
    public sealed class MeleeEncounter
    {
        public const double HealthFraction = .8, EscapeReserve = 10, Leash = 12, WorkLimit = 45, NoProgress = 12;
        public const double ClosingRange = 30, ClosingLimit = 12, ClosingNoProgress = 4, EngagementRange = 3;
        readonly bool defensive;
        double started, progressAt, bestHealth, closestDistance;
        public string Reason { get; private set; }
        public bool Engaged { get; private set; }

        public MeleeEncounter(double now, double targetHealth, double targetDistance = 0, bool defensive = false)
        {
            this.defensive = defensive;
            started = progressAt = now; bestHealth = targetHealth;
            closestDistance = targetDistance; Engaged = defensive || targetDistance <= EngagementRange;
        }

        public static bool Eligible(string prefab, int level, bool player, bool tamed, bool hostile) =>
            (prefab == "Greyling" || prefab == "Boar") && level == 1 && !player && !tamed && hostile;

        public static bool SearchEligible(string prefab, int level, bool player, bool tamed, bool hostile, bool explicitlyNamed) =>
            Eligible(prefab, level, player, tamed, hostile) && (prefab == "Greyling" || explicitlyNamed);

        public static bool DefensiveReady(double hp, double max, double stamina, double armour, bool fed, bool confirmed) =>
            confirmed && fed && armour >= 2 && hp >= 30 && max > 0 && hp >= max * .65 && stamina >= 35;

        public static bool DefensiveHealth(double hp, double max) => max > 0 && hp >= 25 && hp >= max * .6;

        public static bool RunApproach(MeleeFrame frame, double reach) =>
            !frame.Incoming && frame.TargetDistance > Math.Max(8, reach * 2) &&
            frame.Stamina >= frame.AttackCost + Math.Max(frame.BlockCost, frame.DodgeCost) + EscapeReserve + 20;

        public MeleeAction Decide(MeleeFrame f, double now)
        {
            Reason = null;
            if (!f.Valid) { Reason = "target is no longer loaded"; return MeleeAction.Lost; }
            if (!f.Eligible) return Retreat("target is no longer an eligible hostile");
            if (f.Dead) return MeleeAction.Won;
            if (!f.Weapon) return Retreat("no usable melee weapon");
            if (f.Unsafe) return Retreat("another creature, bystander or unsafe footing is nearby");
            if (defensive ? !DefensiveHealth(f.Hp, f.HpMax) : !(f.HpMax > 0) || f.Hp < f.HpMax * HealthFraction) return Retreat("health below combat threshold");
            if (f.Stamina < EscapeReserve) return Retreat("stamina needed for escape");
            if (defensive && (now - started >= 12 || f.TargetDistance > 6 || f.DistanceFromStart > 6))
                return Retreat("defensive encounter limit");
            if (!Engaged)
            {
                if (f.TargetDistance > ClosingRange || f.DistanceFromStart > ClosingRange)
                    return Retreat("target exceeded the closing area");
                if (now - started >= ClosingLimit) return Retreat("closing time limit");
                if (f.TargetDistance > EngagementRange)
                {
                    if (f.TargetDistance <= closestDistance - .5) { closestDistance = f.TargetDistance; progressAt = now; }
                    if (now - progressAt >= ClosingNoProgress) return Retreat("no progress closing on target");
                    return f.Busy ? MeleeAction.Wait : MeleeAction.Closing;
                }
                // The host anchors the encounter when the actual fight starts.
                Engaged = true; started = progressAt = now; bestHealth = f.TargetHp;
            }
            if (f.DistanceFromStart > Leash) return Retreat("encounter exceeded its pursuit leash");
            if (f.TargetDistance > ClosingRange) return Retreat("target left the closing area");
            if (now - started >= WorkLimit) return Retreat("combat time limit");
            if (f.TargetHp < bestHealth - .01) { bestHealth = f.TargetHp; progressAt = now; }
            if (now - progressAt >= NoProgress) return Retreat("no observed progress against target");
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
