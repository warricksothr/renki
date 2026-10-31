using System;
using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public sealed class DryTravelTests
    {
        [Fact]
        public void CapturedRainSurfaceAndBodyStepOffsetInvalidateAMarginalShorePath()
        {
            var support = new V3(-30.64, 30.8, -17.65);
            Assert.True(DryTravel.Clear(support, 30)); // Old static sea-level check.
            Assert.False(DryTravel.Clear(support, 30.46, DryTravel.TravelMargin));
            var higher = new V3(-30.64, 31.5, -17.65);
            Assert.True(DryTravel.Clear(higher, 30.46, DryTravel.TravelMargin));
            // Even if the body stands a full step below the sample, it stays
            // clear of that observed surface with the base dry margin.
            Assert.True(DryTravel.Clear(higher - new V3(0, .6, 0), 30.46));
            Assert.False(DryTravel.Clear(higher, 31, DryTravel.TravelMargin));
            Assert.True(DryTravel.Clear(new V3(0, 36, 0), 31, DryTravel.TravelMargin));
        }

        [Fact]
        public void MotionGuardStopsAtTheCheckedCornerWhileStrafingKeepsFullClearance()
        {
            var from = new V3(0, 31, 0);
            var corner = new V3(.2, 31, 0);
            // The safe path turns up Z at x=.2. The wall beyond the turn
            // rejects the former half-meter straight lookahead.
            bool Corridor(double lookahead) => lookahead <= .25;
            Assert.False(Corridor(.5));
            Assert.True(Corridor(DryTravel.ClearanceDistance(from, corner)));
            Assert.False(Corridor(DryTravel.ClearanceDistance(from, corner, strafing: true)));
            Assert.Equal(.5, DryTravel.ClearanceDistance(from, new V3(2, 31, 0)));
            Assert.Equal(.5, DryTravel.ClearanceDistance(from, null));
        }

        [Fact]
        public void CapturedWetReturnSupportIsRejectedButBridgeSupportIsDry()
        {
            Assert.False(DryTravel.Clear(new V3(-38.86, 28.1, -33.36), 30));
            Assert.False(DryTravel.Clear(new V3(0, 29.8, 0), 30));
            Assert.True(DryTravel.Clear(new V3(0, 30.2, 0), 30));
            Assert.True(DryTravel.Clear(new V3(0, 36, 0), 30));
            Assert.False(DryTravel.Clear(new V3(0, double.NaN, 0), 30));
            Assert.False(DryTravel.Clear(V3.Zero, double.NaN));
        }

        [Fact]
        public void LocalPlanningRoutesAroundAWetInletAndRechecksChangedWater()
        {
            var water = 30.0;
            V3? Probe(V3 from, V3 desired)
            {
                var wet = desired.X > 2 && desired.X < 6 && Math.Abs(desired.Z) < 2;
                var supported = new V3(desired.X, wet ? 29.8 : 30.4, desired.Z);
                return DryTravel.Clear(supported, water) ? supported : (V3?)null;
            }
            var start = new V3(0, 30.4, 0);
            var goal = new V3(8, 30.4, 0);
            Assert.Null(Probe(start, new V3(4, 30.4, 0)));
            var plan = new LocalRoute(start, goal, Probe);
            while (plan.Status == RouteStatus.Searching) plan.Advance();
            Assert.Equal(RouteStatus.Found, plan.Status);
            Assert.Contains(plan.Path, p => Math.Abs(p.Z) >= 2);
            Assert.All(plan.Path, p => Assert.True(DryTravel.Clear(p, water)));
            water = 31;
            Assert.Null(Probe(start, plan.Path[1]));
        }

        [Fact]
        public void DryShoreRecoveryKeepsWalkingPastShallowWater()
        {
            double Ground(double x, double z) => Math.Abs(x) >= 7 || Math.Abs(z) >= 7 ? 30.4 : 29.8;
            var shallow = DeepWater.Shore(new V3(0, 28, 0), Ground, 30);
            Assert.True(shallow.HasValue);
            Assert.False(DryTravel.Clear(shallow.Value, 30));
            var dry = DeepWater.Shore(new V3(0, 28, 0), Ground, 30, maxDepth: -DryTravel.Margin);
            Assert.True(dry.HasValue);
            Assert.True(DryTravel.Clear(dry.Value, 30));
        }
    }
}
