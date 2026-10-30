using System;

namespace Renki.Tasks
{
    public enum ScoutFootingAction { Ready, Wait, Abort }

    /// <summary>Pause shallow scouting steps; never permit airborne combat or travel.</summary>
    public sealed class ScoutFooting
    {
        public const double SettleLimit = .5, SupportLimit = .4;
        double? airborneAt;

        public ScoutFootingAction Decide(double now, bool grounded, bool scouting, bool drySupport, double supportGap)
        {
            if (double.IsNaN(now) || double.IsInfinity(now)) return ScoutFootingAction.Abort;
            if (grounded) { airborneAt = null; return ScoutFootingAction.Ready; }
            if (!scouting || !drySupport || double.IsNaN(supportGap) || double.IsInfinity(supportGap) ||
                Math.Abs(supportGap) > SupportLimit) return ScoutFootingAction.Abort;
            airborneAt ??= now;
            if (now < airborneAt || now - airborneAt >= SettleLimit) return ScoutFootingAction.Abort;
            return ScoutFootingAction.Wait;
        }
    }
}
