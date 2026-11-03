using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public sealed class DoorPassageTests
    {
        static readonly V3 North = new V3(0, 0, 1);

        [Theory]
        [InlineData(true, false, true, false, true)]
        [InlineData(true, true, true, false, false)]
        [InlineData(true, false, false, false, false)]
        [InlineData(false, false, true, false, false)]
        [InlineData(true, false, true, true, false)]
        public void RegisterOnlyOurInitialOpening(bool enabled, bool wasOpen, bool accepted, bool hold, bool tracked)
        {
            Assert.Equal(tracked, DoorPassage.TrackOpening(enabled, wasOpen, accepted, hold));
        }

        [Fact]
        public void OpeningFromTheDoubleDoorPlaneClosesAfterClearingEitherSide()
        {
            foreach (var direction in new[] { -1, 1 })
            {
                var hinge = new V3(2, 0, 0);
                var passage = new DoorPassage(V3.Zero, hinge, North, 0);
                Assert.Equal(DoorCleanup.Wait, passage.Observe(new V3(0, 0, direction * .5), hinge, North, true, true, 4));
                Assert.Equal(DoorCleanup.Close, passage.Observe(new V3(0, 0, direction * 1.6), hinge, North, true, true, 4));
            }
        }

        [Fact]
        public void BothGateHingesCloseAfterPassageEvenWhenTheNextTaskStopsNearby()
        {
            foreach (var hinge in new[] { new V3(-2, 1, 0), new V3(2, 1, 0) })
            {
                var passage = new DoorPassage(new V3(0, 0, -2), hinge, North, 0);
                // Neither leaf's hinge is within the old 1.5m doorway radius.
                Assert.False(DoorRules.In(V3.Zero, hinge));
                Assert.Equal(DoorCleanup.Wait, passage.Observe(V3.Zero, hinge, North, true, true, 1));
                Assert.Equal(DoorCleanup.Wait, passage.Observe(new V3(0, 0, .6), hinge, North, true, true, 3));
                Assert.Equal(DoorCleanup.Close, passage.Observe(new V3(0, 0, 1.6), hinge, North, true, true, 4));
                passage.Closing(4);
                Assert.Equal(DoorCleanup.Wait, passage.Observe(new V3(0, 0, 1.6), hinge, North, true, true, 4.5));
                Assert.Equal(DoorCleanup.Closed, passage.Observe(new V3(0, 0, 1.6), hinge, North, false, true, 5));
            }
        }

        [Fact]
        public void RotatedDoorUsesItsFixedDoorwayPlaneRatherThanWorldNorth()
        {
            var normal = new V3(1, 0, 0);
            var passage = new DoorPassage(new V3(-2, 0, 0), V3.Zero, normal, 0);
            Assert.Equal(DoorCleanup.Wait, passage.Observe(new V3(1, 0, 0), V3.Zero, normal, true, true, 4));
            Assert.Equal(DoorCleanup.Close, passage.Observe(new V3(1.6, 0, 0), V3.Zero, normal, true, true, 4));
        }

        [Fact]
        public void ClearingHingeDistanceOnTheOpeningSideDoesNotMeanCrossing()
        {
            var door = new V3(2, 0, 0);
            var passage = new DoorPassage(new V3(0, 0, -2), door, North, 0);
            Assert.Equal(DoorCleanup.Wait, passage.Observe(new V3(0, 0, -1.6), door, North, true, true, 4));
            // Backing away still tidies our own unused opening, as before.
            Assert.Equal(DoorCleanup.Close, passage.Observe(new V3(0, 0, -5), door, North, true, true, 5));
        }

        [Fact]
        public void NearbyOtherFloorCannotCloseTheDoorAndDistantCleanupIsAbandoned()
        {
            var passage = new DoorPassage(new V3(0, 0, -2), V3.Zero, North, 0);
            Assert.Equal(DoorCleanup.Wait, passage.Observe(new V3(0, 3, 2), V3.Zero, North, true, true, 4));
            Assert.Equal(DoorCleanup.Abandoned, passage.Observe(new V3(0, 0, 20), V3.Zero, North, true, true, 5));
        }

        [Fact]
        public void OpeningRefusalWaitsForReplicationAndObservedClosureNeedsNoToggle()
        {
            var passage = new DoorPassage(new V3(0, 0, -2), V3.Zero, North, 0);
            Assert.Equal(DoorCleanup.Wait, passage.Observe(V3.Zero, V3.Zero, North, false, true, 2.9));
            Assert.Equal(DoorCleanup.OpenRefused, passage.Observe(V3.Zero, V3.Zero, North, false, true, 3));
            var other = new DoorPassage(new V3(0, 0, -2), V3.Zero, North, 0);
            Assert.Equal(DoorCleanup.Wait, other.Observe(V3.Zero, V3.Zero, North, true, true, 1));
            Assert.Equal(DoorCleanup.Closed, other.Observe(V3.Zero, V3.Zero, North, false, true, 2));
        }

        [Fact]
        public void ClosingKeepsItsReceiptPendingAndAllowsOnlyOneSettledRetry()
        {
            var passage = new DoorPassage(new V3(0, 0, -2), V3.Zero, North, 0);
            var pos = new V3(0, 0, 2);
            Assert.Equal(DoorCleanup.Wait, passage.Observe(pos, V3.Zero, North, true, true, 2));
            Assert.Equal(DoorCleanup.Close, passage.Observe(pos, V3.Zero, North, true, true, 3));
            passage.Closing(3);
            Assert.Equal(DoorCleanup.Wait, passage.Observe(pos, V3.Zero, North, true, true, 5.9));
            Assert.Equal(DoorCleanup.Close, passage.Observe(pos, V3.Zero, North, true, true, 6));
            passage.Closing(6);
            Assert.Equal(DoorCleanup.Wait, passage.Observe(pos, V3.Zero, North, true, true, 8.9));
            Assert.Equal(DoorCleanup.CloseRefused, passage.Observe(pos, V3.Zero, North, true, true, 9));
        }

        [Fact]
        public void UnclosableDoorOrMovingOutOfReachCannotTriggerARetry()
        {
            var passage = new DoorPassage(new V3(0, 0, -2), V3.Zero, North, 0);
            var pos = new V3(0, 0, 2);
            Assert.Equal(DoorCleanup.Abandoned, passage.Observe(pos, V3.Zero, North, true, false, 4));
            passage.Closing(4);
            Assert.Equal(DoorCleanup.Abandoned, passage.Observe(new V3(0, 0, 9), V3.Zero, North, true, true, 7));
        }
    }
}
