using System;
using System.Linq;
using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public sealed class HeightRouteTests
    {
        static RouteStatus Finish(LocalRoute route)
        {
            while (route.Status == RouteStatus.Searching) route.Advance(8);
            return route.Status;
        }

        [Fact]
        public void KnownRaisedGoalPrioritizesTheCheckedStairConnectionOverGroundBelowIt()
        {
            // Synthetic lower floor, separate upper floor, and a stair
            // connection away from the goal. Never snap between storeys.
            V3? Probe(V3 from, V3 to)
            {
                var y = 0d;
                var ramp = (to.X - 2) * .75;
                if (to.X >= 2 && to.X <= 6 && Math.Abs(to.Z - 4) < .01 && Math.Abs(ramp - from.Y) <= .6) y = ramp;
                else if (from.Y >= 2.4 && to.X >= 6 && to.X <= 10 && to.Z >= 0 && to.Z <= 4) y = 3;
                return Math.Abs(y - from.Y) <= .6 ? new V3(to.X, y, to.Z) : (V3?)null;
            }
            var goal = new V3(8, 3, 0);
            var flat = new LocalRoute(V3.Zero, goal, Probe, within: .25, arrivalLevel: .6);
            var height = new LocalRoute(V3.Zero, goal, Probe, within: .25, arrivalLevel: .6, heightAware: true);
            Assert.Equal(RouteStatus.Found, Finish(flat));
            Assert.Equal(RouteStatus.Found, Finish(height));
            Assert.True(height.Expanded < flat.Expanded, $"height={height.Expanded}, flat={flat.Expanded}");
            Assert.Equal(3, height.Path.Last().Y);
            Assert.Contains(height.Path, at => Math.Abs(at.Z - 4) < .01 && at.Y > 0 && at.Y < 3);
            for (var i = 1; i < height.Path.Count; i++) Assert.Equal(height.Path[i], Probe(height.Path[i - 1], height.Path[i]));
        }

        [Fact]
        public void AGoalHeightCannotCreateSupportOrRenewTheNodeBound()
        {
            var route = new LocalRoute(V3.Zero, new V3(3, 3, 0), (from, to) => new V3(to.X, 0, to.Z),
                maxNodes: 128, heightAware: true);
            Assert.Equal(RouteStatus.NoRoute, Finish(route));
            Assert.Null(route.Path);
            Assert.Equal(128, route.Expanded);
        }

        [Fact]
        public void DistantUnknownGoalHeightDoesNotDisplaceCheckedFrontierTravel()
        {
            var goal = new V3(100, 1000, 0);
            var flat = new LocalRoute(V3.Zero, goal, (from, to) => new V3(to.X, 0, to.Z));
            var height = new LocalRoute(V3.Zero, goal, (from, to) => new V3(to.X, 0, to.Z), heightAware: true);
            Assert.Equal(RouteStatus.Found, Finish(flat));
            Assert.Equal(RouteStatus.Found, Finish(height));
            Assert.True(height.ProgressLeg);
            Assert.Equal(flat.Path.ToArray(), height.Path.ToArray());
            Assert.Equal(flat.Expanded, height.Expanded);
        }
    }
}
