using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public sealed class RouteProgressTests
    {
        [Fact]
        public void CapturedStairOscillationCannotBuyAnotherReplanForever()
        {
            var progress = new RouteProgress();
            var low = new V3(-47.98, 32.47, 7.77);
            var high = new V3(-47.96, 33.23, 8.65);
            var next = new V3(-47.62, 33.47, 9.13);
            var stalled = false;
            for (var i = 0; i < 100 && !stalled; i++)
            {
                // Each failed plan walks up the same short prefix and slides
                // back. Replacing its waypoint list is not new body progress.
                var pos = i % 20 < 10 ? low : high;
                stalled = progress.Stalled(pos, next, i * .1);
            }
            Assert.True(stalled);
            var memory = new RouteRefusals();
            memory.Remember(low, next, 10, () => true);
            Assert.True(memory.Blocks(low, next, 11));
            Assert.False(memory.Blocks(low, low + new V3(0, 0, -1), 11));
        }

        [Fact]
        public void PlanningAndDoorWaitsPauseExecutionTimeAndRealTravelResetsIt()
        {
            var progress = new RouteProgress();
            for (var i = 0; i < 100; i++) Assert.False(progress.Stalled(V3.Zero, null, i));
            for (var i = 0; i < 30; i++) Assert.False(progress.Stalled(V3.Zero, new V3(1, 0, 0), 100 + i * .1));
            for (var i = 0; i < 100; i++) Assert.False(progress.Stalled(V3.Zero, null, 103 + i));
            // Walking a new part of the route is genuine progress.
            for (var i = 0; i < 100; i++)
            {
                var pos = new V3(i * .1, 0, 0);
                Assert.False(progress.Stalled(pos, pos + new V3(1, 0, 0), 203 + i * .1));
            }
        }

        [Fact]
        public void StalledPrefixChangesTheNextPhysicallyCheckedPlan()
        {
            var memory = new RouteRefusals();
            var start = new V3(-48, 32.5, 7.8);
            var blocked = start + new V3(0, .3, .5);
            memory.Remember(start, blocked, 0, () => true);
            V3? Probe(V3 from, V3 to) => memory.Blocks(from, to, 1) ? (V3?)null : new V3(to.X, 32.5, to.Z);
            var route = new LocalRoute(start, start + new V3(0, 0, 3), Probe, .25);
            while (route.Status == RouteStatus.Searching) route.Advance();
            Assert.Equal(RouteStatus.Found, route.Status);
            Assert.Contains(route.Path, p => System.Math.Abs(p.X - start.X) >= .5);
            Assert.All(route.Path, p => Assert.Equal(32.5, p.Y));
            Assert.False(memory.Blocks(start + new V3(0, 2, 0), blocked + new V3(0, 2, 0), 1));
        }
    }
}
