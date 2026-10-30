using System;
using Renki.Movement;

namespace Renki.Tasks
{
    public enum KiteAction { Draw, Lead, Hold, Engage, Retreat }

    public sealed class KiteFrame
    {
        public V3 Bot, Enemy;
        public double Hp, HpMax, Stamina;
        public bool Valid = true, Eligible = true, Weapon = true, Safe = true, Following, Targeting;
    }

    /// <summary>Bounded local lure. The host measures pursuit and validates all movement.</summary>
    public sealed class Kiting
    {
        public const double SearchRange = 30, AreaRadius = 6, TravelLimit = 40, WorkLimit = 40;
        public const int SurveyAttempts = 34;

        public static V3 SurveyPoint(V3 origin, int attempt)
        {
            if (attempt < 0 || attempt >= SurveyAttempts) throw new ArgumentOutOfRangeException(nameof(attempt));
            var index = attempt % 17;
            if (index == 0) return origin;
            // The second pass fills angular gaps instead of repeating the
            // same rejected centers. Keep one center retry for scene loading.
            var angle = (index - 1) % 8 * Math.PI / 4 + attempt / 17 * Math.PI / 8;
            return origin + new V3(Math.Cos(angle), 0, Math.Sin(angle)) * (index <= 8 ? 6 : 12);
        }
        readonly V3 start, area;
        readonly double began;
        double lastFollow, progressAt, bestAreaDistance, bestGap;
        bool drawn;
        public string Reason { get; private set; }
        public bool Run { get; private set; }

        public Kiting(V3 start, V3 area, V3 enemy, double now)
            : this(start, area, enemy, start, now) { }

        public Kiting(V3 start, V3 area, V3 enemy, V3 bot, double now)
        {
            this.start = start; this.area = area; began = progressAt = lastFollow = now;
            bestGap = V3.FlatDistance(bot, enemy); bestAreaDistance = V3.FlatDistance(area, enemy);
        }

        public KiteAction Decide(KiteFrame f, double now)
        {
            Reason = null; Run = false;
            if (!f.Valid || !f.Eligible) return Abort("target lost or no longer eligible");
            if (!f.Weapon) return Abort("no usable melee weapon");
            if (!f.Safe) return Abort("unsafe kiting ground or another character nearby");
            if (f.HpMax <= 0 || f.Hp < f.HpMax * .8) return Abort("health below combat threshold");
            if (f.Stamina < MeleeEncounter.EscapeReserve) return Abort("stamina needed for escape");
            if ((f.Bot - start).Length > TravelLimit || (f.Enemy - start).Length > TravelLimit) return Abort("kiting travel limit");
            if (now - began >= WorkLimit) return Abort("kiting time limit");
            var gap = V3.FlatDistance(f.Bot, f.Enemy);
            // A confirmed attacker already at our checked fighting stance needs
            // no lure. Hostility or proximity alone still cannot skip pursuit.
            if (!drawn && f.Targeting && gap <= 4 &&
                V3.FlatDistance(f.Bot, area) <= 2 && V3.FlatDistance(f.Enemy, area) <= AreaRadius)
                return KiteAction.Engage;
            if (f.Following)
            {
                if (!drawn) { drawn = true; progressAt = now; bestAreaDistance = V3.FlatDistance(area, f.Enemy); }
                lastFollow = now;
            }
            if (!drawn)
            {
                if (gap <= bestGap - .5) { bestGap = gap; progressAt = now; }
                if (now - progressAt >= 6 || now - began >= 12) return Abort("enemy did not follow");
                if (gap < 4) return Abort("enemy too close before pursuit was established");
                return gap > 8 ? KiteAction.Draw : KiteAction.Hold;
            }
            if (now - lastFollow >= 6) return Abort("enemy stopped following");
            var enemyToArea = V3.FlatDistance(f.Enemy, area);
            if (enemyToArea <= bestAreaDistance - .5) { bestAreaDistance = enemyToArea; progressAt = now; }
            if (V3.FlatDistance(f.Bot, area) <= 2 && enemyToArea <= AreaRadius) return KiteAction.Engage;
            if (now - progressAt >= 8) return Abort("no progress toward fighting ground");
            if (gap > 12 || f.Stamina < 30 && gap >= 6) return KiteAction.Hold;
            if (gap < 5)
            {
                if (f.Stamina < 30) return Abort("cannot maintain spacing with an escape reserve");
                Run = true;
            }
            return KiteAction.Lead;
        }

        public V3 Goal(KiteFrame f, KiteAction action)
        {
            if (action == KiteAction.Draw)
            {
                var away = (f.Bot - f.Enemy).Flat.Normalized;
                return f.Enemy + away * 8;
            }
            if (!Flee.PastThreat(f.Bot, f.Enemy, area)) return area;
            // First go around the threat rather than cut across its attack range.
            var awayFromEnemy = (f.Bot - f.Enemy).Flat.Normalized;
            var side = new V3(-awayFromEnemy.Z, 0, awayFromEnemy.X);
            if (V3.Dot(side, (area - f.Bot).Flat) < 0) side = side * -1;
            return f.Bot + (side + awayFromEnemy * .5).Normalized * 5;
        }

        public static bool ObservedPursuit(V3 previous, V3 current, V3 bot)
        {
            var movement = (current - previous).Flat;
            return movement.Length >= .2 && V3.Dot(movement.Normalized, (bot - previous).Flat.Normalized) > .5;
        }

        KiteAction Abort(string reason) { Reason = reason; return KiteAction.Retreat; }
    }
}
