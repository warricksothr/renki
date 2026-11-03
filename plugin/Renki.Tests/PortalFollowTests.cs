using Renki.Movement;
using Renki.Protocol;
using Xunit;

namespace Renki.Tests
{
    public sealed class PortalFollowTests
    {
        static readonly FollowPortalLink Source = new FollowPortalLink("1:10", "1:20", V3.Zero);

        [Fact]
        public void PassivePickupCanReturnAFollowAfterTheLeaderDisappearedAtAPortal()
        {
            // The trail is observed independently of the active body task.
            var trail = new PortalLeaderTrail();
            trail.Observe(new V3(1, 0, 0), Source, 10);
            Assert.Same(Source, trail.Departure(12, false, V3.Zero));
            Assert.Null(trail.Departure(19, false, V3.Zero));
            Assert.Null(trail.Departure(9, false, V3.Zero));
        }

        [Fact]
        public void WalkingAwayFromAPortalOrDisappearingOnAnotherFloorDoesNotAuthorizeIt()
        {
            var trail = new PortalLeaderTrail();
            trail.Observe(V3.Zero, Source, 0);
            Assert.Null(trail.Departure(1, true, V3.Zero));
            trail.Observe(new V3(3, 0, 0), Source, 1);
            Assert.Null(trail.Departure(2, false, V3.Zero));
            trail.Observe(new V3(0, 3, 0), Source, 3);
            Assert.Null(trail.Departure(4, false, V3.Zero));
            trail.Observe(V3.Zero, null, 5);
            Assert.Null(trail.Departure(6, false, V3.Zero));
        }

        [Fact]
        public void RemotePositionJumpRetainsTheSourceEvenWhileThePlayerObjectStillExists()
        {
            var trail = new PortalLeaderTrail();
            var remote = new V3(1000, 0, 0);
            trail.Observe(V3.Zero, Source, 10);
            trail.Observe(remote, null, 10.1);
            trail.Observe(remote + new V3(1, 0, 0), null, 11);
            Assert.Same(Source, trail.Departure(12, true, remote));
            Assert.Same(Source, trail.Departure(12, false, V3.Zero));
            Assert.Null(trail.Departure(19, true, remote));
        }

        [Fact]
        public void OnlyTheObservedSourceAndUnchangedDestinationAreAdmitted()
        {
            var crossing = new PortalFollowCrossing(Source, 0);
            Assert.True(crossing.Allows("1:10", "1:20"));
            Assert.False(crossing.Allows("1:11", "1:20"));
            Assert.False(crossing.Allows("1:10", "1:21"));
            Assert.False(crossing.Allows("1:10", null));
            Assert.Equal(PortalFollowAction.Failed, crossing.Tick(1, false, false, false, true, false));
            Assert.Equal(Wire.ReasonPortalFailed, crossing.Failure.Reason);
            Assert.False(crossing.Allows("1:10", "1:20"));
        }

        [Fact]
        public void NativeCargoRefusalHasADistinctReasonAndRevokesAdmission()
        {
            var crossing = new PortalFollowCrossing(Source, 0);
            Assert.Equal(PortalFollowAction.Failed, crossing.Tick(1, false, false, true, false, false));
            Assert.Equal(Wire.ReasonPortalCargo, crossing.Failure.Reason);
            Assert.Contains("cargo", crossing.Failure.Detail);
            Assert.False(crossing.Allows("1:10", "1:20"));
        }

        [Fact]
        public void TeleportCanOutlastTheOrdinaryLoseTimerAndArrivalHasItsOwnSightWindow()
        {
            var crossing = new PortalFollowCrossing(Source, 0);
            Assert.Equal(PortalFollowAction.Walk, crossing.Tick(1, false, false, true, true, false));
            Assert.Equal(PortalFollowAction.Hold, crossing.Tick(2, false, false, true, true, true));
            Assert.Equal(PortalFollowAction.Hold, crossing.Tick(3, true, false, true, true, true));
            Assert.False(crossing.Allows("1:10", "1:20"));
            // Source unload and cargo checks are irrelevant while teleporting.
            Assert.Equal(PortalFollowAction.Hold, crossing.Tick(20, true, false, false, false, false));
            Assert.Equal(PortalFollowAction.Arrived, crossing.Tick(21, false, false, false, false, false));
            Assert.Equal(PortalFollowAction.Hold, crossing.Tick(30, false, false, false, false, false));
            Assert.Equal(PortalFollowAction.Resume, crossing.Tick(31, false, true, false, false, false));
            Assert.Null(crossing.Failure);
            Assert.False(crossing.Allows("1:10", "1:20"));
        }

        [Theory]
        [InlineData("approach")]
        [InlineData("trigger")]
        [InlineData("teleport")]
        [InlineData("arrival")]
        public void EveryPhaseIsBoundedAndFailureDoesNotBecomeTargetLost(string phase)
        {
            var crossing = new PortalFollowCrossing(Source, 0);
            PortalFollowAction ended;
            switch (phase)
            {
                case "trigger":
                    crossing.Tick(0, false, false, true, true, true);
                    ended = crossing.Tick(PortalFollowCrossing.TriggerSeconds + 1, false, false, true, true, true);
                    break;
                case "teleport":
                    crossing.Tick(0, true, false, true, true, true);
                    ended = crossing.Tick(PortalFollowCrossing.TeleportSeconds + 1, true, false, false, false, false);
                    break;
                case "arrival":
                    crossing.Tick(0, true, false, true, true, true);
                    crossing.Tick(1, false, false, false, false, false);
                    ended = crossing.Tick(PortalFollowCrossing.ArrivalSeconds + 2, false, false, false, false, false);
                    break;
                default:
                    ended = crossing.Tick(PortalFollowCrossing.ApproachSeconds + 1, false, false, true, true, false);
                    break;
            }
            Assert.Equal(PortalFollowAction.Failed, ended);
            Assert.Equal(Wire.ReasonPortalFailed, crossing.Failure.Reason);
            Assert.False(crossing.Allows("1:10", "1:20"));
        }

        [Fact]
        public void ATripAlreadyUnderwayPausesLossDetectionWithoutAdmittingATrigger()
        {
            var crossing = new PortalFollowCrossing(null, 0);
            Assert.False(crossing.Allows("1:10", "1:20"));
            Assert.Equal(PortalFollowAction.Hold, crossing.Tick(0, true, false, false, false, false));
            Assert.Equal(PortalFollowAction.Hold, crossing.Tick(10, true, false, false, false, false));
            Assert.Equal(PortalFollowAction.Arrived, crossing.Tick(11, false, false, false, false, false));
            Assert.Equal(PortalFollowAction.Resume, crossing.Tick(12, false, true, false, false, false));
            Assert.False(crossing.Allows("1:10", "1:20"));
        }

        [Fact]
        public void CancellationOrALeaderReturningRevokesAdmission()
        {
            var crossing = new PortalFollowCrossing(Source, 0);
            crossing.Cancel();
            Assert.False(crossing.Allows("1:10", "1:20"));
            crossing = new PortalFollowCrossing(Source, 0);
            Assert.Equal(PortalFollowAction.Resume, crossing.Tick(1, false, true, true, true, false));
            Assert.False(crossing.Allows("1:10", "1:20"));
        }

        [Fact]
        public void ReturnTripNeedsANewDepartureAtTheOtherPortal()
        {
            var trail = new PortalLeaderTrail();
            var exit = new FollowPortalLink("1:20", "1:10", new V3(1000, 0, 0));
            trail.Observe(exit.Entry, exit, 100);
            Assert.Same(exit, trail.Departure(101, false, V3.Zero));
            var crossing = new PortalFollowCrossing(exit, 101);
            Assert.True(crossing.Allows("1:20", "1:10"));
            Assert.False(crossing.Allows("1:10", "1:20"));
        }
    }
}
