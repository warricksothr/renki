using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using Newtonsoft.Json;
using Renki.Movement;
using Renki.Protocol;
using Xunit;

namespace Renki.Tests
{
    public sealed class LocalRouteTests
    {
        static Wall Wall(double x1, double z1, double x2, double z2, double y = 0) =>
            new Wall { A = new[] { x1, y, z1 }, B = new[] { x2, y, z2 }, H = 2 };

        static RouteStatus Finish(LocalRoute route)
        {
            for (var i = 0; i <= LocalRoute.MaxNodes && route.Status == RouteStatus.Searching; i++) route.Advance(1);
            return route.Status;
        }

        [Fact]
        public void RealHouseRoutesThroughTheDoorInsteadOfTheEastWall()
        {
            var walls = JsonConvert.DeserializeObject<List<Wall>>(File.ReadAllText(Path.Combine(AppContext.BaseDirectory, "Fixtures", "house-walls.json")));
            var start = new V3(-36.79, 36.09, 7.62);
            var goal = new V3(-6.75, 36.09, 9.7);
            Assert.False(WallRoute.Clear(start, goal, walls));
            var route = new LocalRoute(start, goal, (from, to) => WallRoute.Clear(from, to, walls) ? to : (V3?)null);
            Assert.Equal(RouteStatus.Found, Finish(route));
            Assert.Contains(route.Path, p => p.X < -39);
            Assert.True(WallRoute.Clear(start, route.Path, walls));
            Assert.True(route.Expanded < LocalRoute.MaxNodes);
        }

        [Fact]
        public void ReturnHomeUsesSafeOpenGroundBeforeSearchingForTheRaisedFloor()
        {
            var start = new V3(-.93, 33.25, 9.02);
            var goal = new V3(-35.9, 37.66, 6.53);
            var end = new V3(-37.47, 34.14, 6.13);
            var points = new[] { start, new V3(-20, 33, 10), end };
            var prefix = WallRoute.Prefix(start, goal, points, Array.Empty<Wall>());
            Assert.Equal(points, prefix);
            Assert.False(Arrival.Reached(end, goal, 2));
            var follower = new PathFollower();
            follower.Set(prefix);
            follower.Next(start);
            follower.Next(points[1]);
            Assert.Null(follower.Next(end));

            var wall = new[] { Wall(-25, 0, -25, 20, 33) };
            var truncated = WallRoute.Prefix(start, goal, points, wall);
            Assert.Equal(points.Take(2), truncated);
            Assert.True(WallRoute.Clear(start, truncated, wall));
            Assert.Empty(WallRoute.Prefix(new V3(-36, 34, 6), goal, new[] { end }, Array.Empty<Wall>()));
        }

        [Theory]
        [InlineData(false)]
        [InlineData(true)]
        public void QuarterMetreStairsConnectFloorsInBothDirectionsWithinPhysicalStepLimit(bool descending)
        {
            var start = new V3(0, descending ? 4 : 0, 0);
            var goal = new V3(4, descending ? 0 : 4, 0);
            V3? Probe(V3 from, V3 to)
            {
                // A narrow supported stair flight; stepping off it or
                // skipping too many treads must still refuse.
                if (to.X < 0 || to.X > 4 || Math.Abs(to.Z) > .3) return null;
                var height = Math.Floor(to.X * 4 + 1e-6) * .25;
                if (descending) height = 4 - height;
                return Math.Abs(height - from.Y) <= .6 ? new V3(to.X, height, to.Z) : (V3?)null;
            }
            // The former stride cannot enter the first three-tread sample.
            Assert.Null(Probe(start, new V3(.75, start.Y, 0)));
            var route = new LocalRoute(start, goal, Probe);
            Assert.Equal(RouteStatus.Found, Finish(route));
            Assert.True(Arrival.Reached(route.Path.Last(), goal, .25));
            for (var i = 1; i < route.Path.Count; i++)
                Assert.Equal(route.Path[i], Probe(route.Path[i - 1], route.Path[i]));
        }

        [Fact]
        public void WallsRespectFloorHeightAndBodyClearance()
        {
            var wall = new[] { Wall(0, -2, 0, 2, 2) };
            Assert.True(WallRoute.Clear(new V3(-2, 0, 0), new V3(2, 0, 0), wall));
            Assert.False(WallRoute.Clear(new V3(-2, 2, 0), new V3(2, 2, 0), wall));
            Assert.False(WallRoute.Clear(new V3(-2, 2, 2.3), new V3(2, 2, 2.3), wall));
            Assert.True(WallRoute.Clear(new V3(-2, 2, 2.8), new V3(2, 2, 2.8), wall));
            Assert.True(WallRoute.Clear(new V3(-.4, 2, 0), new V3(-1.2, 2, 0), wall));
            Assert.False(WallRoute.Clear(new V3(-.4, 2, 0), new V3(1.2, 2, 0), wall));
        }

        [Fact]
        public void ClosedRoomHasNoRouteAndSearchHasAHardBound()
        {
            var walls = new[] { Wall(-2, -2, 2, -2), Wall(2, -2, 2, 2), Wall(2, 2, -2, 2), Wall(-2, 2, -2, -2) };
            var enclosed = new LocalRoute(V3.Zero, new V3(8, 0, 0), (a, b) => WallRoute.Clear(a, b, walls) ? b : (V3?)null);
            Assert.Equal(RouteStatus.NoRoute, Finish(enclosed));
            var wrongFloor = new LocalRoute(V3.Zero, new V3(8, 10, 0), (_, b) => b);
            Assert.Equal(RouteStatus.NoRoute, Finish(wrongFloor));
            Assert.Equal(LocalRoute.MaxNodes, wrongFloor.Expanded);
            Assert.Null(wrongFloor.Path);
        }

        [Fact]
        public void DistantHomeAdvancesInCheckedLegsFollowingSlopingSupport()
        {
            var origin = new V3(189.51, 82.18, -289.97);
            var goal = new V3(-35.9, 37.66, 6.53);
            var distance = V3.FlatDistance(origin, goal);
            var direction = (goal - origin).Flat.Normalized;
            V3? Probe(V3 from, V3 to)
            {
                var y = origin.Y + V3.Dot((to - origin).Flat, direction) / distance * (goal.Y - origin.Y);
                return Math.Abs(y - from.Y) <= .6 ? new V3(to.X, y, to.Z) : (V3?)null;
            }
            var here = origin;
            var legs = 0;
            while (!Arrival.Reached(here, goal, .5) && legs < 32)
            {
                var route = new LocalRoute(here, goal, Probe, within: .5);
                Assert.Equal(RouteStatus.Found, Finish(route));
                Assert.True(route.Probes > 0);
                Assert.InRange(route.Expanded, 1, LocalRoute.MaxNodes);
                Assert.All(route.Path, at => Assert.True(Math.Abs(at.X - here.X) <= LocalRoute.Range &&
                    Math.Abs(at.Z - here.Z) <= LocalRoute.Range));
                for (var i = 1; i < route.Path.Count; i++)
                    Assert.Equal(route.Path[i], Probe(route.Path[i - 1], route.Path[i]));
                var end = route.Path.Last();
                if (route.ProgressLeg)
                {
                    Assert.False(Arrival.Reached(end, goal, .5));
                    Assert.True(V3.FlatDistance(here, end) >= LocalRoute.FrontierDistance);
                    Assert.True(V3.FlatDistance(here, goal) - V3.FlatDistance(end, goal) >= LocalRoute.FrontierProgress);
                    Assert.InRange(route.Expanded, 1, 512);
                }
                here = end;
                legs++;
            }
            Assert.InRange(legs, 20, 32);
            Assert.True(Arrival.Reached(here, goal, .5));
        }

        [Fact]
        public void NearbyRaisedHomeStillUsesACheckedApproachBeforeItsEntranceSearch()
        {
            var start = new V3(0, 0, -23);
            var goal = new V3(0, 4, 0);
            // The lower bank is supported but does not reach the home floor.
            // Searching it for an entrance must not exhaust the whole bound
            // before walking the safe approach that is already available.
            var route = new LocalRoute(start, goal, (_, to) => to, within: 2, maxNodes: 512,
                progressHorizon: LocalRoute.FrontierDistance + 2);
            Assert.Equal(RouteStatus.Found, Finish(route));
            Assert.True(route.ProgressLeg);
            Assert.InRange(route.Expanded, 1, 512);
            var end = route.Path.Last();
            Assert.True(V3.FlatDistance(start, end) >= LocalRoute.FrontierDistance);
            Assert.True(V3.FlatDistance(start, goal) - V3.FlatDistance(end, goal) >= LocalRoute.FrontierProgress);
            Assert.All(route.Path, at => Assert.Equal(start.Y, at.Y));
            Assert.False(Arrival.Reached(end, goal, 2));
            Assert.False(new LocalRoute(end, goal, (_, to) => to, within: 2,
                progressHorizon: LocalRoute.FrontierDistance + 2).ProgressLeg);
        }

        [Fact]
        public void DistantProgressRoutesAroundMeasuredWallWithoutUsingTheGoalsFloor()
        {
            var walls = new[] { Wall(3, -4, 3, 4) };
            var goal = new V3(375, 50, 0);
            var route = new LocalRoute(V3.Zero, goal, (a, b) => WallRoute.Clear(a, b, walls) ? b : (V3?)null);
            Assert.Equal(RouteStatus.Found, Finish(route));
            Assert.True(route.ProgressLeg);
            Assert.True(WallRoute.Clear(V3.Zero, route.Path, walls));
            Assert.Contains(route.Path, at => Math.Abs(at.Z) > 4);
            Assert.All(route.Path, at => Assert.Equal(0, at.Y));
            Assert.False(Arrival.Reached(route.Path.Last(), goal, 2));
        }

        [Fact]
        public void DistantSearchCannotEscapeEnclosureOrWalkIntoUnprobedGround()
        {
            var goal = new V3(375, 0, 0);
            var walls = new[] { Wall(-2, -2, 2, -2), Wall(2, -2, 2, 2), Wall(2, 2, -2, 2), Wall(-2, 2, -2, -2) };
            var enclosed = new LocalRoute(V3.Zero, goal, (a, b) => WallRoute.Clear(a, b, walls) ? b : (V3?)null);
            Assert.Equal(RouteStatus.NoRoute, Finish(enclosed));
            Assert.True(enclosed.Probes > 0);
            Assert.Null(enclosed.Path);
            // The same refusal is used for absent support and unsafe water.
            var unloaded = new LocalRoute(V3.Zero, goal, (a, b) => b.Flat.Length < 5 ? b : (V3?)null);
            Assert.Equal(RouteStatus.NoRoute, Finish(unloaded));
            Assert.Null(unloaded.Path);
            Assert.True(unloaded.Expanded < LocalRoute.MaxNodes);
        }

        [Fact]
        public void ACellOnTheUpperFloorIsNotTheCellBeneathIt()
        {
            V3? Probe(V3 from, V3 to)
            {
                if (to.X < -.1 || to.X > 4.6 || to.Z < -.1 || to.Z > .8) return null;
                var y = from.Y >= 2.9 || to.Z > .5 ? 3 : to.X * 2 / 3;
                return Math.Abs(y - from.Y) <= .6 ? new V3(to.X, y, to.Z) : (V3?)null;
            }
            var route = new LocalRoute(V3.Zero, new V3(0, 3, 0), Probe);
            Assert.Equal(RouteStatus.Found, Finish(route));
            Assert.Equal(3, route.Path.Last().Y);
            Assert.Contains(route.Path, p => p.X >= 3.7);
        }

        [Fact]
        public void ApproachUsesTheTasksStoppingDistanceInsteadOfTheSolidGoalCentre()
        {
            var goal = new V3(10, 0, 0);
            V3? Probe(V3 a, V3 b) => b.X >= 9 ? (V3?)null : b;
            var route = new LocalRoute(V3.Zero, goal, Probe, within: 1.5);
            Assert.Equal(RouteStatus.Found, Finish(route));
            Assert.True(Arrival.Reached(route.Path.Last(), goal, 1.5));
            Assert.True(route.Path.Last().X < 9);
            var wrongFloor = new LocalRoute(V3.Zero, new V3(10, 3, 0), Probe, within: 1.5);
            Assert.Equal(RouteStatus.NoRoute, Finish(wrongFloor));
        }

        [Fact]
        public void FollowingWaypointsKeepsTheFloorHeightUntilActuallyReached()
        {
            var above = new V3(1, 3, 1);
            var next = new V3(3, 3, 1);
            var below = new V3(1, 0, 1);
            var path = new PathFollower();
            path.Set(new[] { above, next });
            Assert.Equal(above, path.Next(below));
            Assert.Equal(above, path.Next(below));
            Assert.Equal(next, path.Next(above));
            // The last waypoint also cannot be consumed from below it.
            Assert.Equal(next, path.Next(new V3(3, 0, 1)));
            Assert.Null(path.Next(next));

            // Small differences on sloping terrain remain walkable.
            path.Set(new[] { above, next });
            Assert.Equal(next, path.Next(new V3(1, 2.5, 1)));
        }

        [Fact]
        public void APartialPathEndsWithoutInventingAWaypointToTheGoal()
        {
            var path = new PathFollower();
            path.Set(new[] { V3.Zero, new V3(2, 0, 0) });
            Assert.Equal(new V3(2, 0, 0), path.Next(V3.Zero));
            Assert.Null(path.Next(new V3(2, 0, 0)));
            Assert.Null(path.Next(new V3(2, 0, 0)));
            Assert.Equal(V3.Zero, path.Direction(new V3(2, 0, 0)));
        }

        [Fact]
        public void FollowingNearALeaderBehindAWallWaitsUntilTheyMoveOn()
        {
            var pace = new FollowPace();
            Assert.Equal(Gait.Run, pace.Decide(12, 3.5, RunMode.Auto, false));
            Assert.Equal(Gait.Stop, pace.Decide(4, 3.5, RunMode.Auto, false, blocked: true));
            Assert.Equal(Gait.Stop, pace.Decide(4.2, 3.5, RunMode.Auto, false));
            Assert.Equal(Gait.Walk, pace.Decide(5, 3.5, RunMode.Auto, false, blocked: true));
        }
    }
}
