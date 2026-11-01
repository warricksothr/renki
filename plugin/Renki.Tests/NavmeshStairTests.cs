using System;
using System.Linq;
using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public sealed class NavmeshStairTests
    {
        static readonly V3 Start = new V3(0, 3, 0), Goal = new V3(10, 0, 0);
        // A smooth navmesh starts descending before the actual stair treads.
        static readonly V3[] Hint = { new V3(4, 2, 0), new V3(8, 0, 0), Goal };
        static double Floor(double x) => x <= 4 ? 3 : x >= 8 ? 0 : 3 - (x - 4) * .75;

        static V3? Physical(V3 from, V3 desired)
        {
            var middle = (from + desired) * .5;
            var at = new V3(desired.X, Floor(desired.X), desired.Z);
            middle = new V3(middle.X, Floor(middle.X), middle.Z);
            return DryTravel.ContinuousSupport(from, middle, at) ? at : (V3?)null;
        }

        static RouteStatus Finish(RouteValidation route)
        {
            while (route.Status == RouteStatus.Searching) route.Advance(1);
            return route.Status;
        }

        [Fact]
        public void SmoothedNavmeshUsesOnlyLiveContinuousStairSupport()
        {
            var candidate = new NavmeshCandidates().Try(Start, Goal, Hint,
                (a, b) => DryTravel.Probe(a, b, Physical), .25, .1);
            Assert.Equal(RouteStatus.Found, Finish(candidate));
            Assert.True(candidate.MaxHintHeightOffset > .65);
            Assert.All(candidate.Path, p => Assert.Equal(Floor(p.X), p.Y, 6));
            Assert.True(Arrival.Reached(candidate.Path.Last(), Goal, .25, .1));
            var previous = Start;
            foreach (var point in candidate.Path)
            {
                Assert.NotNull(Physical(previous, point));
                Assert.True(V3.FlatDistance(previous, point) <= .5 + 1e-6);
                previous = point;
            }
        }

        [Fact]
        public void RecordedFloorAgreementRemainsRequiredAndReportsTheActualProjection()
        {
            var recorded = new RouteValidation(Start, Goal, new RouteHint(1, Hint), Physical);
            Assert.Equal(RouteStatus.NoRoute, Finish(recorded));
            Assert.Equal("recorded_floor_changed", recorded.Rejection);
            Assert.True(Math.Abs(recorded.Supported.Value.Y - recorded.Checking.Value.Y) > .65);
            Assert.Equal(3, recorded.Supported.Value.Y);
            Assert.Equal(3, recorded.CheckingFrom.Y);
        }

        [Theory]
        [InlineData("missing_floor")]
        [InlineData("body")]
        [InlineData("water")]
        [InlineData("one_way_drop")]
        public void ProjectingNavmeshNeverBypassesThePhysicalProbe(string obstruction)
        {
            V3? Probe(V3 from, V3 desired)
            {
                if ((obstruction == "body" || obstruction == "missing_floor") && desired.X >= 4) return null;
                if (obstruction == "one_way_drop")
                {
                    var at = new V3(desired.X, desired.X >= 4 ? 0 : 3, desired.Z);
                    return Math.Abs(at.Y - from.Y) <= .3 ? at : (V3?)null;
                }
                var supported = Physical(from, desired);
                return supported.HasValue && (obstruction != "water" || DryTravel.Clear(supported.Value, 1, DryTravel.TravelMargin))
                    ? supported : null;
            }
            var candidate = new NavmeshCandidates().Try(Start, Goal, Hint, Probe, .25);
            Assert.Equal(RouteStatus.NoRoute, Finish(candidate));
            Assert.Equal("physical_probe", candidate.Rejection);
            Assert.Null(candidate.Supported);
            Assert.False(Arrival.Reached(candidate.Path.Last(), Goal, .25));
        }

        [Fact]
        public void NavmeshCannotDeclareArrivalOnADifferentStorey()
        {
            var candidate = new NavmeshCandidates().Try(Start, Goal, new[] { Goal },
                (_, b) => new V3(b.X, 3, b.Z), .25);
            Assert.Equal(RouteStatus.NoRoute, Finish(candidate));
            Assert.Equal("actual_goal_not_reached", candidate.Rejection);
            Assert.All(candidate.Path, p => Assert.Equal(3, p.Y));
            Assert.False(Arrival.Reached(candidate.Path.Last(), Goal, .25));
        }

        [Fact]
        public void ProjectedSamplesKeepTheirOriginalHardBudget()
        {
            var goal = new V3(1000, 0, 0);
            var calls = 0;
            var candidate = new NavmeshCandidates().Try(Start, goal, new[] { goal }, (_, b) =>
            { calls++; return new V3(b.X, 3, b.Z); }, .25);
            Assert.Equal(RouteStatus.NoRoute, Finish(candidate));
            Assert.Equal("sample_budget", candidate.Rejection);
            Assert.Equal(RouteValidation.MaxSamples, calls);
        }
    }
}
