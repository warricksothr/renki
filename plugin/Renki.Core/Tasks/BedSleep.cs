namespace Renki.Tasks
{
    public enum BedStep { Use, Wait, Sleeping, Taken, TimedOut }

    /// <summary>A bed claim is an asynchronous ownership RPC. Wait for it
    /// before the second use, without ever repeating a claim indefinitely.</summary>
    public sealed class BedSleep
    {
        double? claimedAt;

        public void Claimed(double now) { if (!claimedAt.HasValue) claimedAt = now; }

        public BedStep Next(double now, bool owned, bool mine, bool inBed)
        {
            if (owned && !mine) return BedStep.Taken;
            if (mine && inBed) return BedStep.Sleeping;
            if (claimedAt.HasValue && !mine)
                return now - claimedAt.Value < 3 ? BedStep.Wait : BedStep.TimedOut;
            return BedStep.Use;
        }
    }
}
