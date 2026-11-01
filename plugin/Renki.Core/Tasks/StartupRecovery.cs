namespace Renki.Tasks
{
    public enum RecoveryAction { Continue, Hold, Logout }

    /// <summary>Opt-in, one-shot healing guard; no movement or combat while recovering.</summary>
    public sealed class StartupRecovery
    {
        public bool Complete { get; private set; }
        public string Reason { get; private set; }
        double previousHp;
        bool observed;
        public StartupRecovery(bool enabled) { Complete = !enabled; }
        public void Abort() { Complete = true; }

        public RecoveryAction Decide(bool inWorld, double hp, double max, bool settled,
            bool sheltered, bool grounded, bool food, bool wet, bool hostileNear)
        {
            if (Complete) return RecoveryAction.Continue;
            if (!inWorld) return RecoveryAction.Hold;
            var injured = observed && hp < System.Math.Min(previousHp, max) - .15;
            observed = true; previousHp = hp;
            if (hostileNear) return Unsafe("hostile within startup recovery range");
            if (injured) return Unsafe("injury during startup recovery");
            if (hp <= 0 || max <= 0) return Unsafe("no living body for startup recovery");
            if (hp >= 25 && hp >= max * .95) { Complete = true; return RecoveryAction.Continue; }
            // Allow ground/shelter/food state to initialize, while checking threats immediately.
            if (settled && (!sheltered || !grounded || !food || wet))
                return Unsafe("startup recovery needs shelter, grounded dry footing and active food");
            return RecoveryAction.Hold;
        }

        RecoveryAction Unsafe(string reason) { Reason = reason; return RecoveryAction.Logout; }
    }
}
