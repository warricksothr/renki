using System;
using Newtonsoft.Json.Linq;
using Renki.Movement;
using Renki.Protocol;
using Xunit;

namespace Renki.Tests
{
    public sealed class WadingTests
    {
        [Theory]
        [InlineData(2, .8)]
        [InlineData(1.4, .7)]
        [InlineData(.7, 0)]
        [InlineData(-1, 0)]
        public void ReservesDepthBelowNativeSwimming(double nativeDepth, double limit) =>
            Assert.Equal(limit, Wading.DepthLimit(nativeDepth), 6);

        [Fact]
        public void UnknownThresholdNeverGrantsWading()
        {
            Assert.Equal(0, Wading.DepthLimit(double.NaN));
            Assert.Equal(0, Wading.DepthLimit(double.PositiveInfinity));
            Assert.False(Wading.Clear(V3.Zero, 0, 0));
            Assert.False(Wading.Clear(V3.Zero, double.NaN, .8));
        }

        [Fact]
        public void CapturedBankCanContinueWithoutArtificialUphillDeparture()
        {
            // Captured body and reference water; synthetic supported next step.
            var body = new V3(-47.33, 30.95, -117.88);
            var footing = new V3(-47.33, 30.9, -117.88);
            var next = footing + new V3(0, -.1, -.5);
            Assert.False(BankExit.Recovered(body, footing, 30));
            Assert.True(Wading.Supported(body, footing, 30, .8, true, false));
            Assert.True(Wading.Edge(body, footing, (footing + next) * .5, next, 30, 30, 30, .8));
        }

        [Fact]
        public void ShallowSupportedWaterIsWalkingButDeepEdgesAndWavesRefuse()
        {
            var a = new V3(0, 29.5, 0); var b = new V3(.25, 29.5, 0); var c = new V3(.5, 29.5, 0);
            Assert.False(DryTravel.Clear(a, 30, DryTravel.TravelMargin));
            Assert.True(Wading.Edge(a, a, b, c, 30, 30, 30, .8));
            Assert.False(Wading.Edge(a, a, b, c, 30, 30.3, 30, .8));
            Assert.False(Wading.Edge(a, a, b, new V3(.5, 29.3, 0), 30, 30, 30, .8));
            Assert.False(Wading.Edge(a, new V3(0, 29.1, 0), b, c, 30, 30, 30, .8));
            Assert.False(Wading.Edge(a, a, b, c, 30.31, 30, 30, .8));
        }

        [Fact]
        public void ActualSwimmingMissingSupportSlopesAndFootingOffsetsCannotMove()
        {
            var a = new V3(0, 29.5, 0);
            Assert.True(Wading.Supported(a, a, 30, .8, true, false));
            Assert.False(Wading.Supported(a, a, 30, .8, true, true));
            Assert.False(Wading.Supported(a, a, 30, .8, false, false));
            Assert.False(Wading.Supported(a, null, 30, .8, true, false));
            var slope = new SupportSelection(a.Y, .6, true, true);
            slope.Consider(a, .79);
            Assert.False(Wading.Supported(a, slope.Point, 30, .8, true, false));
            Assert.False(Wading.Supported(a, a + new V3(0, -.7, 0), 30, .8, true, false));
            Assert.False(Wading.Edge(a, a, a + new V3(.25, .4, 0), a + new V3(.5, .5, 0), 30, 30, 30, .8));
        }

        [Fact]
        public void CapturedWaveMaySpendSettlingReserveButNextStepMustRecoverIt()
        {
            var body = new V3(-188.95, 29.49, -412.48);
            var floor = new V3(-188.95, 29.45, -412.48);
            const double water = 30.100942611694336;
            Assert.False(Wading.Clear(floor, water, .8));
            Assert.True(Wading.Supported(body, floor, water, .8, true, false));
            var middle = floor + new V3(.25, .08, 0);
            var to = floor + new V3(.5, .16, 0);
            Assert.True(Wading.Edge(body, floor, middle, to, water, water, water, .8));
            Assert.False(Wading.Edge(body, floor, floor + new V3(.25, 0, 0), floor + new V3(.5, 0, 0), water, water, water, .8));
            Assert.False(Wading.Supported(body, floor, 30.26, .8, true, false));
        }

        [Fact]
        public void LiveFootingUsesMeasuredNormalRatherThanClearedContactAccumulator()
        {
            // Native t-1 body/footing. The installed Character API returns
            // m_groundContactNormal, which UpdateGroundContact clears to zero.
            var body = new V3(-52.96, 30.64, -130.23);
            var ray = new SupportSelection(body.Y, .6, true, true);
            ray.Consider(new V3(body.X, 30.62, body.Z), 1);
            Assert.False(DryTravel.WalkableNormal(0));
            Assert.True(Wading.Supported(body, ray.Point, 30, .8, true, false));
            var steep = new SupportSelection(body.Y, .6, true, true);
            steep.Consider(new V3(body.X, 30.62, body.Z), .62);
            Assert.False(Wading.Supported(body, steep.Point, 30, .8, true, false));
        }

        [Theory]
        [InlineData("{\"to\":[1,2,3],\"wade\":true}")]
        [InlineData("{\"to\":[1,2,3],\"dry\":true,\"wade\":true}")]
        [InlineData("{\"to\":\"1:2\",\"dry\":true,\"avoidHostiles\":true,\"wade\":true}")]
        [InlineData("{\"to\":[1,2,3],\"dry\":true,\"avoidHostiles\":true,\"wade\":true,\"recoverThreats\":true}")]
        [InlineData("{\"to\":[1,2,3],\"dry\":true,\"avoidHostiles\":true,\"wade\":\"yes\"}")]
        public void UnsupportedArgumentsRefuse(string json) =>
            Assert.Throws<BadArgsException>(() => TaskArgs.Parse(new TaskRequest { Task = "move_to", Args = JObject.Parse(json) }));

        [Fact]
        public void WadingIsExplicitAndStrictDryDefaultRemains()
        {
            var req = new TaskRequest { Task = "move_to", Args = JObject.Parse("{\"to\":[1,2,3],\"dry\":true,\"avoidHostiles\":true}") };
            Assert.False(((MoveToArgs)TaskArgs.Parse(req)).Wade);
            req.Args["wade"] = true;
            Assert.True(((MoveToArgs)TaskArgs.Parse(req)).Wade);
        }
    }
}
