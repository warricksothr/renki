using System;
using System.Collections.Generic;

namespace Renki.Tasks
{
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
