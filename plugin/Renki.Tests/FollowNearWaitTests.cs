using Renki.Movement;
using Renki.Protocol;
using Xunit;

namespace Renki.Tests
{
    /// <summary>
    /// The follow seen on the VM on 2026-10-29 at 01:39: Nimah inside the
    /// house, the bot outside its east wall 3.56 m away, and every re-path
    /// a straight line through the wall. After nine recoveries the follow
    /// failed <c>stuck</c> and sent the bot home.
    /// </summary>
    public sealed class FollowNearWaitTests
    {
        static readonly V3 Bot = new V3(-28.1, 36.09, 11.55);
        static readonly V3 Nimah = new V3(-31.54, 36.09, 12.45);
        const double Distance = 3.5;

        [Fact]
        public void OutsideTheHouseWallTheFollowerWaitsInsteadOfFailingStuck()
        {
            // Walking up to the house and just beyond the distance, with no
            // known wall between them, the pace keeps walking, so the walker
            // spends its recoveries against the wall.
            var pace = new FollowPace();
            Assert.Equal(Gait.Walk, pace.Decide(6, Distance, RunMode.Auto, false));
            Assert.Equal(Gait.Walk, pace.Decide(V3.FlatDistance(Bot, Nimah), Distance, RunMode.Auto, false));

            var wait = new NearLeaderWait();
            Assert.True(wait.GiveUp(Bot, Nimah, Distance));
            Assert.True(wait.Waiting);
            Assert.True(wait.Hold(Bot, Nimah));
        }

        [Fact]
        public void TheFollowerResumesOnceTheLeaderMovesTwoMetres()
        {
            var wait = new NearLeaderWait();
            Assert.True(wait.GiveUp(Bot, Nimah, Distance));
            // Moving about the room does not end the wait.
            Assert.True(wait.Hold(Bot, Nimah + new V3(-1.2, 0, 1.2)));
            Assert.True(wait.Hold(Bot, Nimah + new V3(0, 0, NearLeaderWait.Resume)));
            Assert.True(wait.Waiting);
            // Two metres and a little from where the wait began, it walks again.
            Assert.False(wait.Hold(Bot, Nimah + new V3(-2.1, 0, 0)));
            Assert.False(wait.Waiting);
            // Climbing a storey counts as moving.
            Assert.True(wait.GiveUp(Bot, Nimah, Distance));
            Assert.False(wait.Hold(Bot, Nimah + new V3(0, 2.5, 0)));
        }

        [Fact]
        public void AFollowerPushedOutOfReachStopsWaiting()
        {
            var wait = new NearLeaderWait();
            Assert.True(wait.GiveUp(Bot, Nimah, Distance));
            Assert.False(wait.Hold(Bot + new V3(2, 0, 0), Nimah));
            Assert.False(wait.Waiting);
        }

        [Fact]
        public void ALeaderOnAnotherFloorIsNotWaitedFor()
        {
            // Upstairs is reached by finding the stairs, so the follow fails
            // stuck and a fresh attempt (escort retries in 2 s) may find them.
            var wait = new NearLeaderWait();
            Assert.False(wait.GiveUp(Bot, Nimah + new V3(0, 2.5, 0), Distance));
            Assert.False(wait.Waiting);
        }

        [Fact]
        public void AFollowerStuckFartherAwayStillFails()
        {
            var wait = new NearLeaderWait();
            Assert.True(wait.GiveUp(Nimah + new V3(Distance + NearLeaderWait.Margin, 0, 0), Nimah, Distance));
            Assert.False(wait.GiveUp(Nimah + new V3(Distance + NearLeaderWait.Margin + .1, 0, 0), Nimah, Distance));
            Assert.False(wait.Waiting);
            Assert.False(wait.Hold(Nimah + new V3(1, 0, 0), Nimah));
        }

        [Fact]
        public void EscortDistancesUseTheSameMargin()
        {
            // Escort follows at 2 to 8 m through the same follower.
            var wait = new NearLeaderWait();
            Assert.True(wait.GiveUp(new V3(9.5, 0, 0), V3.Zero, 8));
            Assert.False(wait.GiveUp(new V3(3.6, 0, 0), V3.Zero, 2));
        }
    }
}
