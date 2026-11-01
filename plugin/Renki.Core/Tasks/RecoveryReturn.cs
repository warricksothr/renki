using System;

namespace Renki.Tasks
{
    /// <summary>An explicitly armed, expiring return to a known same-world home.
    /// It never grants ordinary body tasks control during startup recovery.</summary>
    public sealed class RecoveryReturnOrder
    {
        public string World;
        public double[] Pos;
        public long Expires;
        public string Profile;

        public bool WorkTrip => Profile == "work_trip";
        public double Seconds => WorkTrip ? 240 : 120;
        public double Travel => WorkTrip ? 400 : 160;
        public double Radius => WorkTrip ? 200 : 80;

        public bool Valid(string world, long now) => !string.IsNullOrWhiteSpace(World) &&
            World == world && Pos != null && Pos.Length == 3 &&
            Finite(Pos[0]) && Finite(Pos[1]) && Finite(Pos[2]) &&
            Expires > now && Expires <= now + 600 &&
            (string.IsNullOrEmpty(Profile) || Profile == "local" || WorkTrip);

        internal static bool Finite(double v) => !double.IsNaN(v) && !double.IsInfinity(v);
        public V3 Home => new V3(Pos[0], Pos[1], Pos[2]);
    }

    public enum RecoveryReturnAction { Hold, Move, Arrived, Logout }

    public sealed class RecoveryReturn
    {
        readonly RecoveryReturnOrder order;
        bool admitted, observed, ended;
        double since = -1, previousHp, travelled;
        V3 previousPos;
        public string Reason { get; private set; }

        public RecoveryReturn(RecoveryReturnOrder order) { this.order = order; }
        public void Cancel() { ended = true; Reason = "return cancelled"; }

        RecoveryReturnAction Refuse(string reason)
        {
            ended = true; Reason = reason;
            return RecoveryReturnAction.Logout;
        }

        public RecoveryReturnAction Decide(string world, long utc, double now, V3 pos,
            double hp, double max, double stamina, int foods, bool grounded,
            bool water, bool sheltered, double threatDistance, bool session, bool settled,
            double foodSeconds = 0)
        {
            if (ended) return RecoveryReturnAction.Logout;
            if (since < 0) since = now;
            if (order == null || !order.Valid(world, utc)) return Refuse("invalid, expired or wrong-world home");
            if (!RecoveryReturnOrder.Finite(now) || !RecoveryReturnOrder.Finite(hp) ||
                !RecoveryReturnOrder.Finite(max) || !RecoveryReturnOrder.Finite(stamina) ||
                !RecoveryReturnOrder.Finite(pos.X) || !RecoveryReturnOrder.Finite(pos.Y) ||
                !RecoveryReturnOrder.Finite(pos.Z) || double.IsNaN(threatDistance) ||
                hp < 50 || hp < max * .7 || max <= 0 || water || foods < (order.WorkTrip ? 2 : 1) || threatDistance < 6)
                return Refuse("unsafe health, food, water or threat clearance");
            if (observed)
            {
                if (hp < Math.Min(previousHp, max) - .15) return Refuse("injury during startup return");
                travelled += (pos - previousPos).Length;
            }
            observed = true; previousHp = hp; previousPos = pos;
            if (now - since >= order.Seconds || travelled > order.Travel || (pos - order.Home).Length > order.Radius)
                return Refuse("startup return time or travel bound");
            if (!session)
            {
                if (admitted || now - since >= 10) return Refuse("startup return lost movement session");
                return RecoveryReturnAction.Hold;
            }
            if (!settled) return RecoveryReturnAction.Hold;
            if (!admitted)
            {
                if (!grounded || foods < (order.WorkTrip ? 3 : 2) || stamina < (order.WorkTrip ? 35 : 25) ||
                    order.WorkTrip && (hp < max * .9 || !RecoveryReturnOrder.Finite(foodSeconds) || foodSeconds < order.Seconds + 30))
                    return Refuse("startup return admission reserves");
                admitted = true;
            }
            if (stamina < 8) return Refuse("startup return stamina reserve");
            // A floor below home never establishes arrival or recovery shelter.
            if (V3.FlatDistance(pos, order.Home) <= 3 && Math.Abs(pos.Y - order.Home.Y) <= 1.75)
            {
                if (!grounded || !sheltered || threatDistance < 12) return Refuse("home is not safe shelter");
                ended = true; Reason = "home floor and shelter observed";
                return RecoveryReturnAction.Arrived;
            }
            return RecoveryReturnAction.Move;
        }
    }
}
