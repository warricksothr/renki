using System;
using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public sealed class BankExitTests
    {
        [Fact]
        public void CompletionClearsThePlanningBoundaryAndTheSettledBody()
        {
            // Captured recovery alternated at this rounded support height
            // without walking, until its original eight seconds expired.
            Assert.False(BankExit.Recovered(new V3(0, 30.86, 0), new V3(0, 30.8, 0), 30));
            Assert.False(BankExit.Recovered(new V3(0, 30.88, 0), new V3(0, 30.98, 0), 30));
            Assert.False(BankExit.Recovered(new V3(0, 31.0, 0), new V3(0, 30.88, 0), 30));
            Assert.True(BankExit.Recovered(new V3(0, 30.96, 0), new V3(0, 30.95, 0), 30));
            Assert.False(BankExit.Recovered(new V3(0, 30.96, 0), new V3(0, 30.95, 0), 30.1));
            Assert.False(BankExit.Recovered(new V3(0, 31.26, 0), new V3(0, 31.2, 0), 30, .62));
            Assert.True(BankExit.Recovered(new V3(0, 31.26, 0), new V3(0, 31.2, 0), 30, .9));
        }

        [Theory]
        [InlineData(.62, false)]
        [InlineData(.79, false)]
        [InlineData(.8, true)]
        [InlineData(.9, true)]
        [InlineData(1, true)]
        [InlineData(1.1, false)]
        [InlineData(double.NaN, false)]
        public void WalkingSupportRejectsSlidingSlopesWithoutRejectingFlatStairTreads(double normalY, bool walkable)
        {
            Assert.Equal(walkable, DryTravel.WalkableNormal(normalY));
        }

        [Fact]
        public void CapturedDryShoreCanLeaveASyntheticBankWithoutAdmittingWadingOrDownhillTravel()
        {
            var body = new V3(-52.47, 30.71, 10.94);
            var water = 30.15;
            var middle = body + new V3(.25, .15, 0);
            var end = body + new V3(.5, .3, 0);
            Assert.False(DryTravel.Clear(body, water, DryTravel.TravelMargin));
            Assert.True(BankExit.Clear(body, body, middle, end, water, water, water));
            Assert.False(BankExit.Clear(body, body, body - new V3(.25, .15, 0), body - new V3(.5, .3, 0), water, water, water));
            Assert.False(BankExit.Clear(body, body, middle, end, 30.5, 30.5, 30.5));
            Assert.False(BankExit.Clear(body, body, middle, end, water, 30.4, 30.5));
            Assert.False(BankExit.Clear(body, body + new V3(0, 2, 0), middle, end, water, water, water));
            Assert.False(BankExit.Clear(body, body, middle, end + new V3(.5, .3, 0), water, water, water));
        }

        [Fact]
        public void UphillPhysicalHitOffsetFitsInsideTheOriginalShortStepBound()
        {
            var body = new V3(0, .56, 0); var exit = new BankExit();
            V3? Probe(V3 from, V3 desired)
            {
                // Captured uphill casts shifted support hits by up to2cm.
                var to = desired + (desired - from).Flat.Normalized * .02 + new V3(0, .06, 0);
                var middle = (from + to) * .5;
                return BankExit.Clear(from, from, middle, to, 0, 0, 0) ? to : (V3?)null;
            }
            var step = exit.Next(body, V3.Forward, 0, Probe);
            Assert.True(step.HasValue);
            Assert.InRange(V3.FlatDistance(body, step.Value), .45, .5);
            Assert.False(BankExit.Clear(body, body, new V3(.251, .59, 0), new V3(.502, .62, 0), 0, 0, 0));
        }

        [Fact]
        public void RetainedAscendingSupportHeightsPassTheCorePolicyOnAHalfMetreStep()
        {
            Assert.True(BankExit.Clear(new V3(0, 30.7088, 0), new V3(0, 30.6685, 0),
                new V3(.25, 30.7098, 0), new V3(.5, 30.7463, 0), 30, 30, 30));
        }

        [Fact]
        public void GradualUphillGainUsesTheSameSettlingAllowanceAtBothEnds()
        {
            var body = new V3(0, .56, 0);
            Assert.True(BankExit.Clear(body, body, new V3(.25, .59, 0), new V3(.5, .62, 0), 0, 0, 0));
            // Level contour travel can reach a gentler bank, without losing
            // dry clearance; no downhill or water margin waiver is admitted.
            Assert.True(BankExit.Clear(body, body, new V3(.25, .56, 0), new V3(.5, .56, 0), 0, 0, 0));
            Assert.False(BankExit.Clear(body, body, new V3(.25, .55, 0), new V3(.5, .55, 0), 0, 0, 0));
        }

        [Fact]
        public void UphillSurfaceCannotConcealABodySettlingBelowTheRequiredDryMargin()
        {
            var body = new V3(0, .35, 0);
            var footing = new V3(0, .6, 0);
            Assert.False(BankExit.Clear(body, footing, new V3(.25, .62, 0), new V3(.5, .7, 0), 0, 0, 0));
            Assert.False(BankExit.Clear(body, footing, new V3(.25, double.NaN, 0), new V3(.5, .8, 0), 0, 0, 0));
        }

        [Fact]
        public void RecoverySlicesProbesAndCannotReviveAfterExhaustion()
        {
            var exit = new BankExit(); var calls = 0;
            V3? Refuse(V3 a, V3 b) { calls++; return null; }
            Assert.Null(exit.Next(V3.Zero, V3.Forward, 0, Refuse));
            Assert.Equal(4, calls);
            for (var t = 1; t < 4; t++) exit.Next(V3.Zero, V3.Forward, t, Refuse);
            Assert.True(exit.Refused); Assert.Equal(16, calls);
            exit.Next(V3.Zero, V3.Forward, 5, (a, b) => throw new Exception("terminal"));
        }

        [Fact]
        public void PhysicallyStalledStepTriesAnotherBearingWithoutRenewingTheBudget()
        {
            var exit = new BankExit();
            var first = exit.Next(V3.Zero, V3.Forward, 0, (a, b) => b);
            Assert.Equal(first, exit.Next(V3.Zero, V3.Forward, .9, (a, b) => b));
            // Centimetre bobbing is not forward progress.
            var next = exit.Next(new V3(0, 0, .01), V3.Forward, 1, (a, b) => b);
            Assert.NotEqual(first, next);
            Assert.Equal(1, exit.Stalls);
            Assert.Equal(2, exit.Samples);
            Assert.Null(exit.Next(V3.Zero, V3.Forward, BankExit.Seconds, (a, b) => b));
            Assert.True(exit.Refused);
        }

        [Fact]
        public void RealProgressCanFinishOneStepBeforeStartingTheNext()
        {
            var exit = new BankExit();
            var first = exit.Next(V3.Zero, V3.Forward, 0, (a, b) => b);
            Assert.Equal(first, exit.Next(new V3(0, 0, .1), V3.Forward, .9, (a, b) => b));
            Assert.Equal(first, exit.Next(new V3(0, 0, .2), V3.Forward, 1.8, (a, b) => b));
            var next = exit.Next(new V3(0, 0, .4), V3.Forward, 2.7, (a, b) => b);
            Assert.NotEqual(first, next);
            Assert.Equal(0, exit.Stalls);
            Assert.Equal(2, exit.Samples);
        }

        [Fact]
        public void SelectedShortStepRequiresLiveClearanceAndOriginalTimeAndDistanceBudgets()
        {
            var exit = new BankExit(); var permitted = true;
            V3? Probe(V3 a, V3 b) => permitted ? b : (V3?)null;
            Assert.NotNull(exit.Next(V3.Zero, V3.Forward, 0, Probe));
            permitted = false;
            for (var n = 0; n < 4; n++) Assert.Null(exit.Next(V3.Zero, V3.Forward, 1, Probe));
            Assert.True(exit.Refused);
            exit = new BankExit(); permitted = true;
            exit.Next(V3.Zero, V3.Forward, 0, Probe);
            Assert.Null(exit.Next(V3.Zero, V3.Forward, BankExit.Seconds, Probe));
            Assert.True(exit.Refused);
            exit = new BankExit(); exit.Next(V3.Zero, V3.Forward, 0, Probe);
            Assert.Null(exit.Next(new V3(0, 0, BankExit.Distance), V3.Forward, 1, Probe));
            Assert.True(exit.Refused);
        }
    }
}
