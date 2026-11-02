namespace Renki.Movement
{
    /// <summary>Geometric eligibility only. The caller must observe a
    /// building overhead at the source, none at the destination, and a
    /// complete physically checked path. Uses the shared staging budget.</summary>
    public static class UnderFloorExit
    {
        public const double MinimumRise = .25, MaximumRetreat = 8;
        public static bool Candidate(V3 start, V3 goal, V3 at, double cost) =>
            goal.Y - start.Y > Arrival.Level &&
            cost <= DryStaging.MaximumCost &&
            V3.FlatDistance(start, at) >= DryStaging.MinimumDistance &&
            at.Y >= start.Y + MinimumRise &&
            V3.FlatDistance(at, goal) <= V3.FlatDistance(start, goal) + MaximumRetreat;
    }
}
