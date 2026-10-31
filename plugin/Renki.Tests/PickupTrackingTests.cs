using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public class PickupTrackingTests
    {
        [Fact]
        public void CookingApproachAcceptsCheckedGroundWithinTheActualPickupRadius()
        {
            var feet = new V3(-33.11, 36.15, 9.73);
            var settled = new V3(-34.45, 35.62, 10.29);
            var endpoint = new V3(-33.6, 36.21, 10.53);
            V3? Ground(V3 from, V3 to) => new V3(to.X, 36.15, to.Z);
            Assert.False(PickupTracking.Reached(feet, settled, PickupTracking.PickupDistance));
            Assert.True(PickupTracking.Reached(feet, settled, 3.5));
            Assert.False(PickupTracking.Reached(feet, new V3(settled.X, 39, settled.Z), 3.5));
            // The old half-metre route radius discarded this usable endpoint
            // and tried to plan into the station collider for fifteen seconds.
            Assert.Null(new NavmeshCandidates().Try(feet, settled, new[] { endpoint }, Ground, .5));
            var route = new NavmeshCandidates().Try(feet, settled, new[] { endpoint }, Ground, PickupTracking.PickupDistance);
            Assert.NotNull(route);
            while (route.Status == RouteStatus.Searching) route.Advance();
            Assert.Equal(RouteStatus.Found, route.Status);
            Assert.True(PickupTracking.Reached(route.Path[route.Path.Count - 1], settled, PickupTracking.PickupDistance));
            Assert.Null(new NavmeshCandidates().Try(feet, new V3(settled.X, 39, settled.Z), new[] { endpoint }, Ground, PickupTracking.PickupDistance));
        }

        [Fact]
        public void ASettledCookingDropReplacesItsNearbySpawnGoal()
        {
            var spawn = new V3(-34.13, 35.66, 11.34);
            var settled = new V3(-33.86, 35.13, 10.74);
            var feet = new V3(-33.1, 36.15, 9.73);
            Assert.InRange((settled - spawn).Length, .25, 1);
            Assert.True(PickupTracking.Retarget(spawn, settled));
            Assert.False(PickupTracking.Reached(feet, spawn, 1));
            // A short approach to the settled item suffices; the old spawn
            // endpoint can remain blocked beside the station collider.
            var approach = new V3(-33.1, 36.15, 10.1);
            Assert.True(PickupTracking.Reached(approach, settled, 1));
            Assert.False(PickupTracking.Reached(approach, spawn, 1));
        }

        [Fact]
        public void SmallJitterDoesNotReplanAndAnotherFloorIsNotArrival()
        {
            Assert.False(PickupTracking.Retarget(V3.Zero, new V3(.01, .02, .03)));
            Assert.False(PickupTracking.Reached(new V3(0, 3, 0), V3.Zero, 1));
            Assert.True(PickupTracking.Reached(new V3(.8, 0, 0), V3.Zero, 1));
        }
    }
}
