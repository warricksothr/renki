using System;
using System.Collections.Generic;
using Renki.Movement;

namespace Renki.Tasks
{
    public static class HarvestSafety
    {
        // PvP only gates ordinary player strikes. Falling trunks and other
        // environmental hazards retain their exclusion zone.
        public static bool PlayerBlocks(bool environmentalHazard, bool? attackerPvp, bool? targetPvp) =>
            environmentalHazard || !attackerPvp.HasValue || !targetPvp.HasValue ||
            (attackerPvp.Value && targetPvp.Value);
    }

    public static class HarvestReach
    {
        // Leave a real margin within the weapon's attack ray without making
        // the bot press its capsule into a stump's wider base.
        public static double Range(double attackRange) => Math.Min(1.4, attackRange * .65);
        public static double StandOff(double reach, bool closer) => Math.Min(closer ? .65 : .95, reach * .7);

        public static V3 StandAt(V3 resource, V3 contact, V3 from, double standOff)
        {
            var away = (from - contact).Flat.Normalized;
            if (away.Length < .01) away = (from - resource).Flat.Normalized;
            if (away.Length < .01) away = V3.Forward;
            var at = contact + away * standOff;
            return new V3(at.X, resource.Y, at.Z);
        }
        public static bool InRange(V3 feet, V3 contact, double height, double reach) =>
            (contact - (feet + new V3(0, height, 0))).Length <= reach;
    }

    public static class HarvestHealth
    {
        public static bool Ready(double hp, double maximum) => maximum > 0 && hp >= maximum * .8;
        public static bool Hurt(double started, double current, double maximum) => current < Math.Min(started, maximum) - .01;
    }

    /// <summary>Require two quiet seconds, including remotely simulated
    /// motion. Compare to the quiet anchor so slow drift accumulates.</summary>
    public sealed class HarvestStillness
    {
        V3 position, up, forward;
        double quietAfter = double.NaN;
        public bool Observe(V3 at, V3 axisUp, V3 axisForward, bool moving, double now)
        {
            if (double.IsNaN(quietAfter) || moving || (at - position).Length > .02 ||
                (axisUp - up).Length > .0087 || (axisForward - forward).Length > .0087)
            {
                position = at; up = axisUp; forward = axisForward;
                quietAfter = now + 2;
            }
            return now >= quietAfter;
        }
    }

    public static class HarvestTool
    {
        // A high-tier sword is not an axe/pickaxe. Require actual resource
        // damage as well as tier and usable durability, then prefer damage.
        public static T Best<T>(IEnumerable<T> items, int minimum,
            Func<T, int> tier, Func<T, double> damage, Func<T, bool> usable) where T : class
        {
            T best = null;
            var bestDamage = 0.0;
            foreach (var item in items)
            {
                var amount = damage(item);
                if (!usable(item) || tier(item) < minimum || double.IsNaN(amount) || double.IsInfinity(amount) || amount <= 0) continue;
                if (best != null && amount <= bestDamage) continue;
                best = item;
                bestDamage = amount;
            }
            return best;
        }
    }

    /// <summary>A disappearing entity alone is never a successful harvest.
    /// Require a recent real hit by this player, with no later observed
    /// competing hit. This correlates remote destruction; it is not an
    /// authoritative multiplayer kill receipt.</summary>
    public sealed class HarvestEvidence
    {
        double localHit = double.NegativeInfinity;
        bool lastWasLocal;
        public int LocalHits { get; private set; }
        public void Hit(bool local, double now)
        {
            lastWasLocal = local;
            if (local) { localHit = now; LocalHits++; }
        }
        public bool Broken(double now) => lastWasLocal && now >= localHit && now - localHit <= 2;
    }
}
