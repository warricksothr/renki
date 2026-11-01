using System;
using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public sealed class BankExitTests
    {
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
            Assert.False(BankExit.Clear(body, body, new V3(.25, .56, 0), new V3(.5, .56, 0), 0, 0, 0));
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
