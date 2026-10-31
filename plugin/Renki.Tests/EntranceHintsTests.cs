using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public sealed class EntranceHintsTests
    {
        [Fact]
        public void RaisedHomeUsesObservedEntranceFloorInsteadOfTheGroundBeneathIt()
        {
            var hints = new EntranceHints();
            var from = new V3(-39.9, 32.2, 1.5);
            var goal = new V3(-35.9, 37.66, 6.53);
            var entrance = new V3(-40, 36.09, 11);
            var otherFloor = new V3(-36, 32, 6);
            Assert.Equal(entrance, hints.Select(from, goal, new[] { otherFloor, entrance }));
            Assert.Null(hints.Select(from, goal, new[] { entrance }));
            hints.Reset();
            Assert.Null(hints.Select(entrance, goal, new[] { entrance }));
            Assert.Null(hints.Select(new V3(100, 32, 0), goal, new[] { entrance }));
            Assert.Null(hints.Select(from, goal, new[] { new V3(-36, 36, 40) }));
        }

        [Fact]
        public void ChangedOrUnreachableEntrancesHaveABoundedFallback()
        {
            var hints = new EntranceHints();
            var floors = new[] { new V3(1, 3, 0), new V3(3, 3, 0), new V3(5, 3, 0), new V3(7, 3, 0) };
            for (var i = 0; i < EntranceHints.Limit; i++) Assert.NotNull(hints.Select(V3.Zero, new V3(0, 3, 0), floors));
            Assert.Null(hints.Select(V3.Zero, new V3(0, 3, 0), floors));
        }

        [Fact]
        public void AnEntranceFloorNeedsStricterArrivalThanAnEyeHeightHomeMarker()
        {
            var floor = new V3(0, 1.5, 0);
            var local = new LocalRoute(V3.Zero, floor, (_, at) => at, maxNodes: 16, arrivalLevel: .6);
            Assert.Equal(RouteStatus.NoRoute, local.Advance(32));
            var hint = new RouteHint(1, new[] { V3.Zero });
            var replay = new RouteValidation(V3.Zero, floor, hint, (_, at) => at, arrivalLevel: .6);
            Assert.Equal(RouteStatus.NoRoute, replay.Advance());
            Assert.True(Arrival.Reached(V3.Zero, floor, .5));
            Assert.False(Arrival.Reached(V3.Zero, floor, .5, .6));
        }

        [Fact]
        public void SprintPacingKeepsEscapeReserveAndRegeneratesBeforeResuming()
        {
            var pace = new SprintPacing();
            Assert.True(pace.Run(true, 60, 60));
            Assert.False(pace.Run(true, 12, 60));
            Assert.False(pace.Run(true, 25, 60));
            Assert.False(pace.Run(false, 60, 60));
            Assert.True(pace.Run(true, 40, 60));
            Assert.False(pace.Run(true, double.NaN, 60));
            Assert.False(pace.Run(true, 30, 0));
        }
    }
}
