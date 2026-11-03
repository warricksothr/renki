using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public sealed class FleeProgressTests
    {
        [Fact]
        public void CapturedRaisedBasePositionFailsFastDespiteSyntheticReplansAndJitter()
        {
            // Captured start/goal; obstruction and jitter are synthetic.
            var start = new V3(-33.9, 36.09, 6.68);
            var goal = new V3(-20.85, 36.09, -2.57);
            var progress = new FleeProgress();
            Assert.True(V3.FlatDistance(start, goal) > 1);
            Assert.False(progress.Blocked(start, 5, 100));
            Assert.False(progress.Blocked(start + new V3(.1, .5, 0), 4, 103.99));
            Assert.True(progress.Blocked(start, 3, 104));
        }

        [Fact]
        public void RealMovementAndThreatClearanceResetTheWindow()
        {
            var progress = new FleeProgress();
            Assert.False(progress.Blocked(V3.Zero, 5, 0));
            Assert.False(progress.Blocked(new V3(1, 0, 0), 5, 3));
            Assert.False(progress.Blocked(new V3(1, 0, 0), 5, 6.99));
            Assert.True(progress.Blocked(new V3(1, 0, 0), 5, 7));
            Assert.False(progress.Blocked(new V3(1, 0, 0), 16, 8));
            Assert.False(progress.Blocked(new V3(1, 0, 0), 5, 20));
        }
    }
}
