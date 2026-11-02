using System;

namespace Renki.Tasks
{
    public enum TravelMode { Walk, Escape, Observe, Blocked, Refused }

    /// <summary>One travel task keeps its target through bounded native escapes.
    /// No decision here grants physical movement or combat permission.</summary>
    public sealed class TravelRecovery
    {
        public const int AttemptLimit = 3;
        public const double AttemptSeconds = 30, QuietSeconds = 5, TargetRange = 30, EscapeDistance = 35;
        public TravelMode Mode { get; private set; }
        public int Attempts { get; private set; }
        public string Reason { get; private set; }
        double began, quiet = -1, last = -1;

        public static bool Safe(double hp, double max, double stamina, int foods, bool wet, bool encumbered) =>
            Finite(hp) && Finite(max) && Finite(stamina) && max > 0 && hp >= Math.Max(20, max * .4) &&
            stamina >= 8 && foods > 0 && !wet && !encumbered;

        public static bool Engaged(double gap, bool targeting, bool hasTarget) =>
            Finite(gap) && gap >= 0 && (gap < 15 || targeting && gap <= TargetRange || hasTarget && gap <= 20);

        public TravelMode Decide(double now, bool danger, bool unsafeBody, bool routeChanged = false)
        {
            if (Mode == TravelMode.Refused) return Mode;
            if (double.IsNaN(now) || double.IsInfinity(now) || now < 0 || now < last)
                return Refuse("invalid encounter clock");
            last = now;
            if (unsafeBody) return Refuse("unsafe travel encounter reserves or footing");
            if (Mode != TravelMode.Walk && now - began >= AttemptSeconds)
                return Refuse("travel encounter recovery time exhausted");
            if (danger && Mode != TravelMode.Escape)
            {
                if (Attempts >= AttemptLimit) return Refuse("travel encounter recovery attempts exhausted");
                Attempts++; began = now; quiet = -1;
                Reason = "nearby engaged threat; withdrawing";
                Mode = TravelMode.Escape;
            }
            if (Mode == TravelMode.Observe)
            {
                if (quiet < 0) quiet = now;
                if (now - quiet >= QuietSeconds)
                { Mode = TravelMode.Walk; Reason = "quiet period observed; replan original destination"; }
            }
            if (Mode == TravelMode.Blocked && routeChanged && now - began >= QuietSeconds)
            { Mode = TravelMode.Walk; Reason = "route conditions changed; replan original destination"; }
            return Mode;
        }

        public TravelMode RouteBlocked()
        {
            if (Mode != TravelMode.Walk) return Mode;
            if (last < 0) return Refuse("route blocked before observation");
            if (Attempts >= AttemptLimit) return Refuse("travel encounter recovery attempts exhausted");
            Attempts++; began = last; quiet = -1;
            Reason = "route blocked; waiting for changed conditions";
            return Mode = TravelMode.Blocked;
        }

        public void Escaped(bool succeeded)
        {
            if (Mode != TravelMode.Escape) return;
            if (!succeeded) { Refuse("checked travel escape failed"); return; }
            Mode = TravelMode.Observe; quiet = -1;
            Reason = "escape completed; observing before resuming";
        }

        public void Cancel() => Refuse("travel cancelled");
        TravelMode Refuse(string reason) { Reason = reason; return Mode = TravelMode.Refused; }
        static bool Finite(double value) => !double.IsNaN(value) && !double.IsInfinity(value);
    }
}
