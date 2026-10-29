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
