using System;
using System.Linq;
using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public sealed class NavigationCacheTests
    {
        static ObstacleShape Box(double x = 0, double z = 0, double yaw = 0) =>
            new ObstacleShape(new V3(x, 0, z), 1, 1, 0, 3, yaw);

        [Theory]
        [InlineData(0)]
        [InlineData(.7853981633974483)]
        [InlineData(-1.5707963267948966)]
        public void RotatedBoxesUseRoundedBodyClearanceInsteadOfFilledBounds(double yaw)
        {
            V3 World(double x, double z) => new V3(x * Math.Cos(yaw) - z * Math.Sin(yaw), 0,
                x * Math.Sin(yaw) + z * Math.Cos(yaw));
            var shape = Box(yaw: yaw);
            Assert.True(shape.Blocks(World(-2, 0), World(2, 0)));
            Assert.True(shape.Blocks(World(2, 1.39), World(.9, 1.39)));
            Assert.False(shape.Blocks(World(2, 1.41), World(-2, 1.41)));
            // Outside the rounded corner, inside its enclosing square.
            Assert.False(shape.Blocks(World(2, 1.3), World(1.3, 1.3)));
        }

        [Fact]
        public void CircularPrimitivesDoNotFillTheirBoundingSquare()
        {
            var shape = new ObstacleShape(V3.Zero, 1, 1, 0, 3, circle: true);
            Assert.True(shape.Blocks(new V3(-2, 0, 0), new V3(2, 0, 0)));
            Assert.True(shape.Blocks(new V3(2, 0, 1.39), new V3(0, 0, 1.39)));
            Assert.False(shape.Blocks(new V3(2, 0, 1.3), new V3(.99, 0, 1.3)));
        }

        [Fact]
        public void FloorsStepsCeilingsAndBackingOutRemainLiveDecisions()
        {
            var from = new V3(-2, 0, 0); var to = new V3(2, 0, 0);
            Assert.False(new ObstacleShape(V3.Zero, 1, 1, 2, 4).Blocks(from, to));
            Assert.False(new ObstacleShape(V3.Zero, 1, 1, -.2, .3).Blocks(from, to));
            Assert.False(Box().Blocks(from, new V3(2, .6, 0)));
            Assert.False(Box().Blocks(new V3(-1.2, 0, 0), from));
            Assert.True(Box().Blocks(from, to));
            Assert.False(Box().Blocks(new V3(-2, 4, 0), new V3(2, 4, 0)));
        }

        [Fact]
        public void SpatialIndexTracksMovesRemovalsAndNegativeCellBoundaries()
        {
            var index = new ObstacleIndex();
            var from = new V3(-6, 0, -4); var to = new V3(-2, 0, -4);
            index.Put(1, Box(-4, -4));
            Assert.True(index.Blocked(from, to));
            var revision = index.Revision;
            index.Put(1, Box(20, 20));
            Assert.True(index.Revision > revision);
            Assert.False(index.Blocked(from, to));
            Assert.True(index.Blocked(new V3(18, 0, 20), new V3(22, 0, 20)));
            index.Remove(1);
            Assert.False(index.Blocked(new V3(18, 0, 20), new V3(22, 0, 20)));
            Assert.Equal(0, index.Count);
            index.Put(2, Box()); index.Clear();
            Assert.False(index.Blocked(new V3(-2, 0, 0), new V3(2, 0, 0)));
        }

        [Fact]
        public void IndexVisitsANearbyShapeOnceAndSkipsDistantGeometry()
        {
            var index = new ObstacleIndex();
            for (var i = 0; i < 1000; i++) index.Put(i, Box(100 + i * 4, 100));
            index.Put(1000, new ObstacleShape(V3.Zero, 8, 8, 10, 12));
            Assert.False(index.Blocked(new V3(-6, 0, -6), new V3(6, 0, 6)));
            Assert.Equal(1L, index.Tested);
        }

        [Fact]
        public void StaleGeometryCannotRejectALegBeforeItsSweep()
        {
            var index = new ObstacleIndex();
            index.Put(1, Box());
            var from = new V3(-2, 0, 0); var to = new V3(2, 0, 0);
            Assert.False(index.Blocked(from, to, _ => false));
            Assert.Equal(0L, index.Rejected);
            index.Put(2, Box());
            Assert.True(index.Blocked(from, to, id => id == 2));
            Assert.Equal(1L, index.Rejected);
        }

        [Fact]
        public void RefreshPriorityProtectsNearObjectsThenTheTravelCorridor()
        {
            var bot = V3.Zero; var goal = new V3(40, 0, 0);
            double Priority(double x, double z) => ObstacleIndex.Priority(bot, goal, new V3(x, 0, z));
            Assert.True(Priority(-4, 0) < Priority(16, 0));
            Assert.True(Priority(16, 0) < Priority(16, 12));
            Assert.True(Priority(16, 0) < Priority(-16, 0));
            Assert.Equal(8, Priority(8, 0));
            Assert.True(ObstacleIndex.Priority(bot, new V3(-40, 0, 0), new V3(-16, 0, 0)) < Priority(16, 12));
        }

        [Fact]
        public void CachedBroadPhaseFindsTheSameDetourWithFewerLiveQueries()
        {
            var shape = new ObstacleShape(V3.Zero, .2, 3, 0, 3);
            var index = new ObstacleIndex(); index.Put(1, shape);
            var baselineCalls = 0; var cachedCalls = 0;
            V3? Live(V3 from, V3 to) { baselineCalls++; return shape.Blocks(from, to) ? (V3?)null : to; }
            V3? Cached(V3 from, V3 to)
            {
                if (index.Blocked(from, to)) return null;
                cachedCalls++;
                return shape.Blocks(from, to) ? (V3?)null : to;
            }
            LocalRoute Route(Func<V3, V3, V3?> probe)
            {
                var route = new LocalRoute(new V3(-3, 0, 0), new V3(3, 0, 0), probe);
                while (route.Status == RouteStatus.Searching) route.Advance();
                Assert.Equal(RouteStatus.Found, route.Status);
                return route;
            }
            var baseline = Route(Live); var cached = Route(Cached);
            Assert.Equal(baseline.Path.ToArray(), cached.Path.ToArray());
            Assert.True(cachedCalls < baselineCalls);
            Assert.True(index.Rejected > 0);
        }

        [Fact]
        public void SupportSamplesShareExactQueriesWithoutMergingHeightsOrFloors()
        {
            var calls = 0;
            var memo = new SupportMemo((at, y) => { calls++; return new V3(at.X, y, at.Z); });
            var at = new V3(1, 0, 2);
            Assert.Equal(new V3(1, 0, 2), memo.Sample(at, 0, 1, 0));
            Assert.Equal(new V3(1, 0, 2), memo.Sample(at, 0, 1.1, 0));
            Assert.Equal(1, calls);
            Assert.Equal(new V3(1, 3, 2), memo.Sample(at, 3, 1.1, 0));
            memo.Sample(at, .01, 1.1, 0);
            memo.Sample(new V3(1.001, 0, 2), 0, 1.1, 0);
            Assert.Equal(4, calls);
            Assert.Equal(1, memo.Hits);
            Assert.Equal(4, memo.Misses);
        }

        [Fact]
        public void SupportExpiryAndNextSliceSeeBuiltAndDemolishedFloors()
        {
            V3? floor = null;
            var memo = new SupportMemo((_, __) => floor);
            Assert.Null(memo.Sample(V3.Zero, 0, 1, 0));
            floor = V3.Zero;
            Assert.Null(memo.Sample(V3.Zero, 0, 1.1, 0));
            Assert.Equal(V3.Zero, memo.Sample(V3.Zero, 0, 1.1, 1));
            floor = null;
            Assert.Null(memo.Sample(V3.Zero, 0, 1.2, 2));
            floor = V3.Zero;
            Assert.Equal(V3.Zero, memo.Sample(V3.Zero, 0, 2.2, 2));
            floor = null;
            Assert.Null(memo.Sample(V3.Zero, 0, .5, 2));
        }

        [Fact]
        public void SupportMemoHasABoundEvenDuringALargeSearch()
        {
            var memo = new SupportMemo((at, _) => at);
            for (var i = 0; i < 20000; i++)
            {
                memo.Sample(new V3(i, 0, 0), 0, 1, 0);
                Assert.InRange(memo.Count, 1, 8192);
            }
        }
    }
}
