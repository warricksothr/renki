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

        // Rotated logs can have a high pivot and a low reachable end. Never
        // begin support projection above both the pivot and measured contact.
        public static double GroundProbeTop(double pivotY, double contactY) => Math.Min(pivotY, contactY) + 2;

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

        // Sight is measured toward physical strike contact, never the center
        // of a tall collider's bounds. Walls and other first hits still block.
        public static bool CanStrike(V3 feet, V3 contact, double height, double reach, IEnumerable<SightHit> hits) =>
            InRange(feet, contact, height, reach) && InteractionSight.Visible(hits);
    }

    /// <summary>Seeds along a fallen trunk, outside its measured bounds.
    /// Every seed still needs physical contact, ground, sight and routing.</summary>
    public static class HarvestContactSearch
    {
        public const int Limit = 20;

        // Upright trees need trunk-height candidates, not canopy bounds.
        // Bias the first candidate toward the bot, then cover both sides
        // on three small rings. Physics still owns contact and ground.
        public static V3 TreeSeed(V3 resource, V3 from, int attempt)
        {
            if (attempt < 0 || attempt >= Limit) throw new ArgumentOutOfRangeException(nameof(attempt));
            var away = (from - resource).Flat.Normalized;
            if (away.Length < .01) away = V3.Forward;
            var side = new V3(-away.Z, 0, away.X);
            var index = attempt % 8;
            var angle = ((index + 1) / 2) * (index % 2 == 1 ? 1 : -1) * Math.PI / 4;
            return resource + (away * Math.Cos(angle) + side * Math.Sin(angle)) * (2 + attempt / 8);
        }

        public static V3 Seed(V3 min, V3 max, int attempt)
        {
            if (attempt < 0 || attempt >= Limit) throw new ArgumentOutOfRangeException(nameof(attempt));
            var center = (min + max) * .5;
            var extent = (max - min) * .5;
            var station = attempt / 4;
            var fraction = station == 0 ? 0 : (station <= 2 ? .5 : .9) * (station % 2 == 1 ? 1 : -1);
            var alongX = extent.X >= extent.Z;
            var along = alongX ? new V3(extent.X * fraction, 0, 0) : new V3(0, 0, extent.Z * fraction);
            // The two long sides give different chopping heights on slopes;
            // the other sides also sample the physical ends of the trunk.
            var side = attempt % 4;
            if (side >= 2) along = alongX ? new V3(0, 0, extent.Z * fraction) : new V3(extent.X * fraction, 0, 0);
            var offset = alongX
                ? side < 2 ? new V3(0, 0, (extent.Z + 2) * (side == 0 ? 1 : -1)) : new V3((extent.X + 2) * (side == 2 ? 1 : -1), 0, 0)
                : side < 2 ? new V3((extent.X + 2) * (side == 0 ? 1 : -1), 0, 0) : new V3(0, 0, (extent.Z + 2) * (side == 2 ? 1 : -1));
            return center + along + offset;
        }
    }

    public sealed class HarvestContactProgress
    {
        double since = double.NaN;
        int hits;
        public void Reset() => since = double.NaN;
        public void Contact(double now, int localHits)
        {
            if (!double.IsNaN(since)) return;
            since = now; hits = localHits;
        }
        public bool Stalled(double now, int localHits)
        {
            if (double.IsNaN(since)) return false;
            if (hits != localHits) { since = now; hits = localHits; }
            return now - since > 8;
        }
    }

    public static class HarvestHealth
    {
        public static bool Ready(double hp, double maximum) => maximum > 0 && hp >= maximum * .8;
        public static bool Hurt(double started, double current, double maximum) => current < Math.Min(started, maximum) - .01;
        public static bool Interrupted(bool damaged, double previous, double previousMaximum, double current, double maximum, bool foodRefreshed) =>
            damaged || Hurt(previous, current, maximum) && !(foodRefreshed && maximum > previousMaximum);
    }

    public static class HarvestEscape
    {
        public const int CandidateLimit = 7;

        // A long trunk's pivot can be far from the actual strike contact.
        // Complete the checked escape leg even when already far from it.
        public static bool NeedsRetreat(V3 feet, V3 escape) => V3.FlatDistance(feet, escape) > .5;

        // Outward then alternating sides, through 90 degrees. Every segment
        // increases distance from the resource; none crosses toward the trunk.
        public static V3 Candidate(V3 resource, V3 feet, int attempt)
        {
            if (attempt < 0 || attempt >= CandidateLimit) throw new ArgumentOutOfRangeException(nameof(attempt));
            var away = (feet - resource).Flat.Normalized;
            if (away.Length < .01) throw new ArgumentException("escape needs a side of the resource");
            var angle = ((attempt + 1) / 2) * (attempt % 2 == 1 ? 1 : -1) * Math.PI / 6;
            var side = new V3(-away.Z, 0, away.X);
            return feet + (away * Math.Cos(angle) + side * Math.Sin(angle)) * 6;
        }
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
