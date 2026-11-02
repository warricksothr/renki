using System;
using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public sealed class DryTravelTests
    {
        [Fact]
        public void AShoreHintCannotSkipAnInterveningWaterHostileOrBodyRefusal()
        {
            var from = new V3(-41.36, 31.11, -43.38);
            var shore = new V3(-37.66, 31.43, -44.91);
            var calls = 0;
            var result = DryTravel.ShoreStep(from, shore, (a, b) =>
            {
                calls++;
                Assert.InRange(V3.FlatDistance(a, b), .499, .501);
                // A dry distant shore says nothing about the blocked step.
                return null;
            });
            Assert.Null(result);
            Assert.Equal(1, calls);
            var shortShore = from + new V3(.1, .05, 0);
            Assert.Equal(new V3(shortShore.X, from.Y, shortShore.Z), DryTravel.ShoreStep(from, shortShore, (_, at) => at));
            Assert.Null(DryTravel.ShoreStep(from, from, (_, at) => throw new Exception("no step needed")));
        }

        [Fact]
        public void FootingChoosesTheNearestSurfaceWithoutSnappingUpAFloorLip()
        {
            const double body = 33;
            Assert.True(DryTravel.PreferFooting(body, 32.85, null));
            Assert.True(DryTravel.PreferFooting(body, 33.08, 32.85));
            Assert.False(DryTravel.PreferFooting(body, 33.56, 33.08));
            Assert.False(DryTravel.PreferFooting(body, 32.3, null));
            Assert.False(DryTravel.PreferFooting(body, double.NaN, null));
            Assert.False(DryTravel.PreferFooting(body, double.PositiveInfinity, null));
            Assert.True(DryTravel.ContinuousSupport(new V3(0, 33, 0),
                new V3(0, 33.08, 0), new V3(.25, 33.2, 0), new V3(.5, 33.33, 0)));
        }

        [Fact]
        public void CapsuleOffsetDoesNotTurnSupportedDescentIntoAFloorLip()
        {
            var body = new V3(-29.9, 35.4, -2.01);
            var support = new V3(-29.9, 35.24, -2.01);
            var middle = new V3(-29.65, 35.08, -2.01);
            var to = new V3(-29.4, 34.9, -2.01);
            Assert.True(DryTravel.ContinuousSupport(body, middle, to));
            Assert.True(DryTravel.ContinuousSupport(body, support, middle, to));
            // A supported descent may now span 0.6m. Near that bound the
            // capsule offset must still be removed from the measured drop.
            var lower = to - new V3(0, .11, 0);
            Assert.False(DryTravel.ContinuousSupport(body, middle, lower));
            Assert.True(DryTravel.ContinuousSupport(body, support, middle, lower));
            // Captured stair-exit footing: the capsule remains 0.32m above
            // a gentle supported slope while the body's step settles.
            Assert.True(DryTravel.ContinuousSupport(new V3(-46.5, 34.07163, 10.99),
                new V3(-46.5, 33.75901, 10.99), new V3(-46.52, 33.72362, 10.74),
                new V3(-46.53, 33.69058, 10.49)));
            // Offset compensation cannot authorize an actual abrupt lip,
            // an overhead floor, or a body without nearby support.
            Assert.False(DryTravel.ContinuousSupport(body, support, support, to - new V3(0, .3, 0)));
            var overhead = body + new V3(0, .56, 0);
            Assert.False(DryTravel.ContinuousSupport(body, overhead, overhead, overhead));
            var below = body - new V3(0, .66, 0);
            Assert.False(DryTravel.ContinuousSupport(body, below, below, below));
        }

        [Fact]
        public void CapturedBedExitAllowsASmallSupportedDropWithoutAdmittingTheReverseClimb()
        {
            // task-9yw1yxhf exhausted a 54-node bed surface because its
            // 0.4m descent was treated like an upward floor lip.
            var body = new V3(-31.08, 36.55, 15.19);
            var support = new V3(-31.08, 36.490482330322266, 15.19);
            var middle = new V3(-31.33, 36.090484619140625, 15.19);
            var floor = new V3(-31.58, 36.090484619140625, 15.19);
            Assert.True(DryTravel.ContinuousSupport(body, support, middle, floor));
            Assert.False(DryTravel.ContinuousSupport(floor, middle, support));

            V3 Surface(V3 at) => new V3(at.X,
                V3.FlatDistance(at, support) < 1.4 ? support.Y : floor.Y, at.Z);
            V3? Probe(V3 from, V3 desired)
            {
                var to = Surface(desired);
                return DryTravel.ContinuousSupport(from, Surface(from), Surface((from + desired) * .5), to) ? to : (V3?)null;
            }
            var goal = floor + new V3(-3, 0, 0);
            var route = new LocalRoute(body, goal, Probe, arrivalLevel: .1);
            while (route.Status == RouteStatus.Searching) route.Advance();
            Assert.Equal(RouteStatus.Found, route.Status);
            Assert.InRange(route.Path[route.Path.Count - 1].Y, floor.Y - .01, floor.Y + .01);
        }

        [Fact]
        public void SupportedDescentCannotAccumulateTwoDropsOrHideAMidpointPit()
        {
            var from = V3.Zero;
            Assert.True(DryTravel.ContinuousSupport(from, new V3(.25, -.6, 0), new V3(.5, -.6, 0)));
            Assert.False(DryTravel.ContinuousSupport(from, new V3(.25, -.3, 0), new V3(.5, -.6001, 0)));
            Assert.False(DryTravel.ContinuousSupport(from, new V3(.25, -.6, 0), new V3(.5, -1.2, 0)));
            Assert.False(DryTravel.ContinuousSupport(from, new V3(.25, -.9, 0), new V3(.5, -.6, 0)));
            Assert.False(DryTravel.ContinuousSupport(from, new V3(.25, double.NaN, 0), new V3(.5, 0, 0)));
        }

        [Fact]
        public void ContinuousSupportFindsTheRampInsteadOfWalkingIntoAFloorLip()
        {
            double Height(V3 p) => p.X <= 0 ? .56 : p.X >= 2 || p.Z < 2 ? 0 : (2 - p.X) * .28;
            V3 Support(V3 p) => new V3(p.X, Height(p), p.Z);
            V3? Probe(V3 from, V3 desired)
            {
                var to = Support(desired);
                return DryTravel.ContinuousSupport(from, Support((from + desired) * .5), to) ? to : (V3?)null;
            }
            var start = new V3(2, 0, 0);
            var goal = new V3(-2, .56, 0);
            Assert.Null(Probe(new V3(.5, 0, 0), new V3(0, .56, 0)));
            var route = new LocalRoute(start, goal, Probe, arrivalLevel: .1);
            while (route.Status == RouteStatus.Searching) route.Advance();
            Assert.Equal(RouteStatus.Found, route.Status);
            Assert.Contains(route.Path, p => p.Z >= 2);
            Assert.InRange(route.Path[route.Path.Count - 1].Y, .55, .57);
        }

        [Fact]
        public void CapturedRaisedFloorLipCannotMasqueradeAsTwoStairTreads()
        {
            var from = new V3(-31.09, 35.53, -.5);
            var floor = new V3(-31.59, 36.09, 0);
            var middle = (from + floor) * .5;
            // Whichever side of the lip the midpoint samples, the entire
            // 0.56m rise falls into one half-edge and cannot be walked.
            Assert.False(DryTravel.ContinuousSupport(from, new V3(middle.X, from.Y, middle.Z), floor));
            Assert.False(DryTravel.ContinuousSupport(from, new V3(middle.X, floor.Y, middle.Z), floor));
            var stair = new V3(.5, 36.03, 0);
            var start = new V3(0, 35.53, 0);
            var tread = new V3(.25, 35.78, 0);
            Assert.True(DryTravel.ContinuousSupport(start, tread, stair));
            Assert.True(DryTravel.ContinuousSupport(stair, tread, start));
            Assert.True(DryTravel.ContinuousSupport(from, from + new V3(-.25, 0, 0), from + new V3(-.5, 0, 0)));
        }

        [Fact]
        public void DiagonalPlanningCannotSkipTheBodiesShorterLookahead()
        {
            var from = new V3(0, 35.14, 0); var to = new V3(.5, 35.48, -.5);
            var calls = 0;
            V3? Physical(V3 a, V3 b)
            {
                calls++;
                // Both endpoints of the full edge are supported; its first
                // half-metre lands on the higher corner of a log pile.
                return V3.FlatDistance(a, b) > .6 ? b : (V3?)null;
            }
            Assert.Null(DryTravel.Probe(from, to, Physical));
            Assert.Equal(2, calls);
            calls = 0;
            Assert.Null(DryTravel.Probe(from, new V3(.3, 35.14, 0), Physical));
            Assert.Equal(1, calls);
            Assert.Equal(to, DryTravel.Probe(from, to, (a, b) => b));
        }

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
        public void ThinCurrentBufferCanHaveAFullyCheckedUphillDeparture()
        {
            // Captured grounded/out-of-water position, synthetic bank and
            // water surface. This is policy evidence, not captured terrain.
            var start = new V3(-52.29, 30.75, 10.95);
            var goal = start + new V3(4, 1.6, 0);
            var water = 30.0;
            var blocked = false;
            V3 Surface(V3 at) => new V3(at.X, start.Y + (at.X - start.X) * .4, at.Z);
            V3? Physical(V3 from, V3 desired)
            {
                var middle = Surface((from + desired) * .5);
                var to = Surface(desired);
                return !blocked && DryTravel.ContinuousSupport(from, Surface(from), middle, to) &&
                    DryTravel.Clear(middle, water, DryTravel.TravelMargin) &&
                    DryTravel.Clear(to, water, DryTravel.TravelMargin) ? to : (V3?)null;
            }
            V3? Probe(V3 from, V3 desired) => DryTravel.Probe(from, desired, Physical);
            Assert.True(DryTravel.Clear(start, water));
            Assert.False(DryTravel.Clear(start, water, DryTravel.TravelMargin));
            var uphill = start + new V3(.5, .2, 0);
            Assert.Equal(uphill, Probe(start, uphill));
            Assert.Null(Probe(start, start + new V3(-.5, -.2, 0)));
            Assert.Null(Probe(start, start + new V3(0, 0, .5)));
            var route = new LocalRoute(start, goal, Probe, arrivalLevel: .1);
            while (route.Status == RouteStatus.Searching) route.Advance();
            Assert.Equal(RouteStatus.Found, route.Status);
            var previous = start;
            foreach (var point in route.Path)
            {
                if ((point - previous).Length < .01) continue;
                Assert.NotNull(Probe(previous, point));
                previous = point;
            }
            // The escape from a thin current buffer cannot weaken the next
            // step's normal water rule or bypass another physical refusal.
            water = 30.2;
            Assert.Null(Probe(start, uphill));
            water = 30;
            blocked = true;
            Assert.Null(Probe(start, uphill));
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
