using System;
using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public class ApproachRecoveryTests
    {
        readonly V3 goal = new V3(0, 3.8, 0);

        [Fact]
        public void UnderARaisedFloorWalksAwayThenRetriesFromEitherSide()
        {
            var recovery = new ApproachRecovery();
            var pos = V3.Zero;
            var targets = new V3[3];
            for (var i = 0; i < 3; i++)
            {
                var time = i * 4;
                Assert.Equal(ApproachStep.Detour, recovery.Update(pos, goal, true, time, p => p));
                targets[i] = recovery.Target.Value;
                Assert.True(V3.FlatDistance(targets[i], goal) >= 6);
                Assert.Equal(ApproachStep.Retry, recovery.Update(targets[i], goal, false, time + 1, p => p));
                Assert.Null(recovery.Target);
                // Simulate the retried partial path returning under the floor.
                Assert.Equal(ApproachStep.None, recovery.Update(pos, goal, true, time + 2, p => p));
            }
            Assert.True(targets[0].Z > 0);
            Assert.True(targets[1].X < 0);
            Assert.True(targets[2].X > 0);
            Assert.Equal(ApproachStep.GiveUp, recovery.Update(pos, goal, true, 12, p => p));
        }

        [Fact]
        public void BlockedDetourRetriesAfterADeadlineAndDoesNotLoopForever()
        {
            var recovery = new ApproachRecovery();
            Assert.Equal(ApproachStep.Detour, recovery.Update(V3.Zero, goal, true, 0, p => p));
            Assert.Equal(ApproachStep.None, recovery.Update(V3.Zero, goal, true, 5, p => p));
            Assert.Equal(ApproachStep.Retry, recovery.Update(V3.Zero, goal, true, 6, p => p));
            // Even moving without reaching the original endpoint has a bound.
            Assert.Equal(ApproachStep.GiveUp, recovery.Update(new V3(3, 0, 0), goal, false, 30, p => p));
        }

        [Fact]
        public void UnsafeCandidatesAreNotWalkedAndRetriesAreSpacedOut()
        {
            var recovery = new ApproachRecovery();
            var probes = 0;
            V3? Reject(V3 p) { probes++; return null; }
            for (var i = 0; i < 3; i++)
            {
                Assert.Equal(ApproachStep.Retry, recovery.Update(V3.Zero, goal, true, i * 2, Reject));
                Assert.Null(recovery.Target);
                Assert.Equal(ApproachStep.None, recovery.Update(V3.Zero, goal, true, i * 2 + 1, Reject));
            }
            Assert.Equal(6, probes);
            Assert.Equal(ApproachStep.GiveUp, recovery.Update(V3.Zero, goal, true, 6, Reject));
        }

        [Fact]
        public void ACompleteRepathOrNewGoalClearsTheRecovery()
        {
            var recovery = new ApproachRecovery();
            Assert.Equal(ApproachStep.None, recovery.Update(V3.Zero, goal, false, 0, p => p));
            Assert.Equal(ApproachStep.Detour, recovery.Update(V3.Zero, goal, true, 1, p => p));
            recovery.Reset();
            Assert.Equal(ApproachStep.None, recovery.Update(V3.Zero, goal, false, 100, p => p));
            Assert.Null(recovery.Target);
            Assert.Equal(0, recovery.Attempts);
        }

        [Fact]
        public void PartialOffLevelAndWaterCrossingDetourPathsAreRejected()
        {
            var target = new V3(6, 0, 0);
            bool Dry(V3 a, V3 b) => false;
            Assert.False(ApproachRecovery.SafePath(V3.Zero, target, Array.Empty<V3>(), Dry));
            Assert.False(ApproachRecovery.SafePath(V3.Zero, target, new[] { new V3(2, 0, 0) }, Dry));
            Assert.False(ApproachRecovery.SafePath(V3.Zero, target, new[] { new V3(6, -3, 0) }, Dry));
            var path = new[] { new V3(3, 0, 0), target };
            Assert.False(ApproachRecovery.SafePath(V3.Zero, target, path, (a, b) => b.X == 3));
            Assert.False(ApproachRecovery.SafePath(V3.Zero, target, path, (a, b) => a.X == 3));
            Assert.True(ApproachRecovery.SafePath(V3.Zero, target, path, Dry));
        }
    }
}
