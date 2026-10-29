using System;
using System.Collections.Generic;
using Renki.Movement;

namespace Renki.Tasks
{
    public static class HarvestHealth
    {
        public static bool Ready(double hp, double maximum) => maximum > 0 && hp >= maximum * .8;
        public static bool Hurt(double started, double current) => current < started - .01;
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
