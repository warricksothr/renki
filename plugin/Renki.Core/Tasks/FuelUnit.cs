using System;

namespace Renki.Tasks
{
    public enum FuelAction { Add, Wait, Ready, Failed }

    /// <summary>One ordinary fuel use. Acknowledgement is not consumption:
    /// both pack and synchronized fuel must agree before another use.</summary>
    public sealed class FuelUnit
    {
        readonly int packBefore;
        readonly double fuelBefore;
        bool requested, complete, failed;
        double requestedAt;
        public string Failure { get; private set; }
        public FuelUnit(int pack, double fuel) { packBefore = pack; fuelBefore = fuel; }
        FuelAction Fail(string why) { failed = true; Failure = why; return FuelAction.Failed; }
        public FuelAction Next(double elapsed, int pack, double fuel)
        {
            if (failed) return FuelAction.Failed;
            if (complete) return FuelAction.Ready;
            if (!Finite(elapsed) || elapsed < 0 || !Finite(fuelBefore) || fuelBefore < 0 || !Finite(fuel) || fuel < 0 || pack < 0)
                return Fail("invalid fuel observation; not adding again");
            if (!requested)
            {
                if (packBefore < 1 || pack != packBefore || fuel != fuelBefore)
                    return Fail("carried fuel or fire changed before use; not adding again");
                requested = true; requestedAt = elapsed; return FuelAction.Add;
            }
            if (elapsed < requestedAt || elapsed - requestedAt > 5 || pack < packBefore - 1 || pack > packBefore || fuel > fuelBefore + 1.05)
                return Fail("ambiguous or late pack/fuel receipt; not adding again");
            // Allow a little normal burning during the bounded remote receipt.
            if (pack == packBefore - 1 && fuel >= fuelBefore + .9)
            { complete = true; return FuelAction.Ready; }
            return FuelAction.Wait;
        }
        internal static bool Finite(double value) => !double.IsNaN(value) && !double.IsInfinity(value);
    }

    /// <summary>Bounded explicit count, serialized through one-unit receipts.</summary>
    public sealed class FireplaceFuel
    {
        readonly int wanted;
        readonly double capacity;
        FuelUnit unit;
        double nextAt;
        bool complete, failed;
        public int Added { get; private set; }
        public bool Pending => unit != null;
        public string Detail { get; private set; }
        public FireplaceFuel(int count, double maxFuel) { wanted = count; capacity = maxFuel; }
        FuelAction Fail(string why) { failed = true; Detail = why; return FuelAction.Failed; }
        public FuelAction Next(double elapsed, int pack, double fuel)
        {
            if (failed) return FuelAction.Failed;
            if (complete) return FuelAction.Ready;
            if (wanted < 1 || wanted > 16 || !FuelUnit.Finite(capacity) || capacity < 1 ||
                !FuelUnit.Finite(elapsed) || elapsed < 0 || !FuelUnit.Finite(fuel) || fuel < 0 || fuel > capacity || pack < 0)
                return Fail("invalid fireplace observation; not adding again");
            if (unit != null)
            {
                var step = unit.Next(elapsed, pack, fuel);
                if (step == FuelAction.Failed) return Fail(unit.Failure);
                if (step != FuelAction.Ready) return step;
                Added++; unit = null; nextAt = elapsed + .25;
            }
            if (Added == wanted || Math.Ceiling(fuel) >= capacity)
            { complete = true; Detail = Added == wanted ? "requested count verified" : "fire full"; return FuelAction.Ready; }
            if (elapsed < nextAt) return FuelAction.Wait;
            if (pack < 1) return Fail("no carried fuel remains");
            unit = new FuelUnit(pack, fuel);
            return unit.Next(elapsed, pack, fuel);
        }
    }
}
