using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public class PickupTrackingTests
    {
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
