using System;

namespace Renki.Tasks
{
    public enum CookingFuelAction { Add, Wait, Ready, Failed }

    /// <summary>One wood for one empty supporting fire. A missing RPC receipt
    /// or a blocked fire must never trigger repeated fuel consumption.</summary>
    public sealed class CookingFuel
    {
        readonly int woodBefore;
        readonly double fuelBefore;
        bool requested, complete, failed;
        public string Failure { get; private set; }
        public CookingFuel(int wood, double fuel) { woodBefore = wood; fuelBefore = fuel; }
        CookingFuelAction Fail(string why) { failed = true; Failure = why; return CookingFuelAction.Failed; }
        public CookingFuelAction Next(double elapsed, int wood, double fuel, bool heated)
        {
            if (failed) return CookingFuelAction.Failed;
            if (complete) return CookingFuelAction.Ready;
            if (double.IsNaN(elapsed) || double.IsInfinity(elapsed) || elapsed < 0 ||
                double.IsNaN(fuel) || double.IsInfinity(fuel) || fuel < 0)
                return Fail("invalid fire observation");
            if (!requested)
            {
                if (woodBefore < 1 || wood != woodBefore || fuelBefore != 0 || fuel != 0)
                    return Fail("one carried wood and an empty supporting fire are required");
                requested = true; return CookingFuelAction.Add;
            }
            if (wood == woodBefore - 1 && fuel >= fuelBefore + .5 && heated)
            { complete = true; return CookingFuelAction.Ready; }
            if (elapsed > 5 || wood < woodBefore - 1)
                return Fail("wood/fuel/heat receipt unconfirmed or fire blocked; not adding again");
            return CookingFuelAction.Wait;
        }
    }
}
