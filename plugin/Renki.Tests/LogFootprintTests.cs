using System;
using System.Linq;
using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public sealed class LogFootprintTests
    {
        static LogFootprint Log(double x = 0, double z = 0, double minY = 0, double maxY = .6, double yaw = 0) =>
            new LogFootprint(new V3(x, (minY + maxY) * .5, z), 4, .35, minY, maxY, yaw);

        [Theory]
        [InlineData(0)]
        [InlineData(.7853981633974483)]
        [InlineData(-1.5707963267948966)]
        public void LowLogsForbidCrossingButLeaveClearGroundAndAttackStances(double yaw)
        {
            V3 Rotate(double x, double z) => new V3(x * Math.Cos(yaw) - z * Math.Sin(yaw), 0, x * Math.Sin(yaw) + z * Math.Cos(yaw));
            var log = Log(yaw: yaw);
            Assert.True(log.Blocks(Rotate(0, -2), Rotate(0, 2)));
            Assert.True(log.Blocks(Rotate(0, -.9), Rotate(0, -.5)));
            Assert.False(log.Blocks(Rotate(-5, -.76), Rotate(5, -.76)));
            Assert.False(log.Blocks(Rotate(0, -2), Rotate(0, -1.4)));
        }

        [Fact]
        public void EgressCannotBeUsedToEnterCrossOrTravelAlongALog()
        {
            var log = Log();
            Assert.False(log.Blocks(new V3(0, 0, -.5), new V3(0, 0, -.6)));
            Assert.True(log.Blocks(new V3(0, 0, -.5), new V3(0, 0, 2)));
            Assert.False(log.Blocks(V3.Zero, new V3(0, 0, -.5)));
            Assert.True(log.Blocks(V3.Zero, new V3(5, 0, 0)));
            Assert.True(log.Blocks(new V3(0, 0, -.5), new V3(.1, 0, -.5)));
        }

        [Fact]
        public void HeightAllowsPassageUnderRaisedLogsAndTravelAboveThem()
        {
            var a = new V3(0, 0, -2); var b = new V3(0, 0, 2);
            Assert.False(Log(minY: 2, maxY: 2.8).Blocks(a, b));
            Assert.True(Log(minY: 1.5, maxY: 2.3).Blocks(a, b));
            Assert.False(Log().Blocks(a + new V3(0, 1, 0), b + new V3(0, 1, 0)));
        }

        [Fact]
        public void MovingRollingAndSplittingReplaceOldFootprints()
        {
            var valid = true;
            var index = new LogFootprintIndex(_ => valid);
            var a = new V3(0, 0, -2); var b = new V3(0, 0, 2);
            index.Put(1, Log()); Assert.True(index.Blocked(a, b));
            valid = false; Assert.False(index.Blocked(a, b));
            valid = true; index.Put(1, Log(x: 12)); Assert.False(index.Blocked(a, b));
            index.Remove(1); index.Put(2, Log(z: 6)); index.Put(3, Log(z: -6));
            Assert.Equal(2, index.Count); Assert.False(index.Blocked(a, b));
            Assert.True(index.Blocked(new V3(0, 0, 4), new V3(0, 0, 8)));
            index.Clear(); Assert.Equal(0, index.Count);
        }

        [Fact]
        public void MeasuredRotatedEnvelopeContainsEveryTiltedColliderCorner()
        {
            var corners = Enumerable.Range(0, 8).Select(i =>
            {
                var x = (i & 1) == 0 ? -4 : 4;
                var y = (i & 2) == 0 ? -.3 : .3;
                var z = (i & 4) == 0 ? -.4 : .4;
                return new V3(10 + x * .8 - z * .6, 3 + y + x * .1, 20 + x * .6 + z * .8);
            }).ToArray();
            var measured = LogFootprint.Measure(corners, new V3(.8, 0, .6));
            Assert.True(measured.HasValue);
            var f = measured.Value;
            Assert.Equal(4, f.HalfX, 8); Assert.Equal(.4, f.HalfZ, 8);
            Assert.Equal(2.3, f.MinY, 8); Assert.Equal(3.7, f.MaxY, 8);
            Assert.Null(LogFootprint.Measure(new[] { new V3(double.NaN, 0, 0) }, V3.Zero));
            Assert.Null(LogFootprint.Measure(new[] { V3.Zero, new V3(1000, 1, 1000) }, V3.Zero));
        }

        [Fact]
        public void PlannerFindsADetourAroundLowLogInsteadOfClimbingItsLip()
        {
            var index = new LogFootprintIndex(); index.Put(1, Log());
            var route = new LocalRoute(new V3(0, 0, -2), new V3(0, 0, 2),
                (from, to) => index.Blocked(from, to) ? (V3?)null : to);
            while (route.Status == RouteStatus.Searching) route.Advance();
            Assert.Equal(RouteStatus.Found, route.Status);
            Assert.Contains(route.Path, p => Math.Abs(p.X) > 4);
        }
    }
}
