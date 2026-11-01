using System;
using Renki.Movement;
using Renki.Tasks;
using Xunit;

namespace Renki.Tests
{
    public class PursuitGroupTests
    {
        static PursuitObservation Target() => new PursuitObservation { Id = "target", Pos = new V3(6, 0, 0), TargetKnown = true, Targeting = true };
        static PursuitObservation Other(double x = 30) => new PursuitObservation { Id = "other", Pos = new V3(x, 0, 0), TargetKnown = true };
        static IsolationEvidence Stable(PursuitGroup group, PursuitObservation other = null)
        {
            IsolationEvidence evidence = null;
            for (var i = 0; i <= 3; i++)
            {
                group.Sample(other == null ? new[] { Target() } : new[] { Target(), other }, V3.Zero, i);
                evidence = group.Assess(V3.Zero, i);
            }
            return evidence;
        }

        [Fact]
        public void OneFollowerSeparatingRequiresAContinuousQuietInterval()
        {
            var group = new PursuitGroup("target");
            group.Sample(new[] { Target(), Other() }, V3.Zero, 0);
            Assert.False(group.Assess(V3.Zero, 0).Isolated);
            for (var i = 1; i <= 3; i++)
            {
                group.Sample(new[] { Target(), Other() }, V3.Zero, i);
                Assert.Equal(i == 3, group.Assess(V3.Zero, i).Isolated);
            }
        }

        [Fact]
        public void WholePackPersistentHuntAndNearbyCompetitorNeverCertifyIsolation()
        {
            foreach (var other in new[] { Other(20), Other(), Other() })
            {
                if (other.Pos.X == 30) other.HuntingPlayers = true;
                var group = new PursuitGroup("target");
                Assert.False(Stable(group, other).Isolated);
                Assert.False(PursuitGroup.WithinBudget(60, 0));
            }
        }

        [Fact]
        public void LatePursuerNewArrivalAndOwnershipLossInvalidateAnEarnedIsolation()
        {
            foreach (var change in new[] { "pursuer", "arrival", "remote", "missing" })
            {
                var group = new PursuitGroup("target");
                Assert.True(Stable(group, Other()).Isolated);
                var other = Other();
                if (change == "pursuer") other.Targeting = true;
                if (change == "remote") other.TargetKnown = false;
                group.Sample(change == "arrival" ? new[] { Target(), other, new PursuitObservation { Id = "new", Pos = new V3(50, 0, 0), TargetKnown = true } } :
                    change == "missing" ? new[] { Target() } : new[] { Target(), other }, V3.Zero, 3.5);
                var evidence = group.Assess(V3.Zero, 3.5);
                Assert.False(evidence.Isolated);
                if (change == "remote" || change == "missing") Assert.Contains("other", evidence.Unknown);
            }
        }

        [Fact]
        public void MovementOverridesAnOwnedNonTargetAndStaleFramesCannotKeepQuiet()
        {
            var group = new PursuitGroup("target");
            Assert.True(Stable(group, Other(35)).Isolated);
            group.Sample(new[] { Target(), Other(34) }, V3.Zero, 3.5);
            Assert.Contains("other", group.Assess(V3.Zero, 3.5).Pursuers);
            Assert.False(group.Assess(V3.Zero, 3.5).Isolated);
            Assert.False(group.Assess(V3.Zero, 5).Isolated);
        }

        [Fact]
        public void UnloadedIsNotDeadAndObservedDeathAloneRemovesARisk()
        {
            foreach (var loaded in new[] { false, true })
            {
                var group = new PursuitGroup("target");
                var other = Other(); other.Loaded = loaded; other.Dead = true;
                Assert.Equal(loaded, Stable(group, other).Isolated);
            }
        }

        [Fact]
        public void InvalidDuplicateAndOverflowObservationsFailClosed()
        {
            var group = new PursuitGroup("target");
            group.Sample(new[] { Target(), Target() }, V3.Zero, 0);
            Assert.False(group.Assess(V3.Zero, 0).Isolated);
            var crowded = new PursuitGroup("target");
            var observations = new PursuitObservation[PursuitGroup.Maximum + 1];
            for (var i = 0; i < observations.Length; i++) observations[i] = new PursuitObservation { Id = "enemy" + i, TargetKnown = true };
            crowded.Sample(observations, V3.Zero, 0);
            Assert.Contains("invalid", crowded.Assess(V3.Zero, 0).Reason);
            Assert.False(PursuitGroup.WithinBudget(double.NaN, 0));
            Assert.False(PursuitGroup.WithinBudget(1, 41));
            Assert.True(PursuitGroup.WithinBudget(59, 40));
            Assert.False(PursuitGroup.Prepared(80, 100, 40, true, 2, .24, 1));
            Assert.False(PursuitGroup.Prepared(80, 100, 39, true, 2, 1, 1));
            Assert.False(PursuitGroup.Prepared(80, 100, 40, false, 2, 1, 1));
            Assert.False(PursuitGroup.Prepared(80, 100, 40, true, 0, 1, 1));
            Assert.True(PursuitGroup.Prepared(80, 100, 40, true, 2, .25, .25));
        }

        [Fact]
        public void APlayerHuntFlagAloneDoesNotIdentifyTheSelectedPlayersTarget()
        {
            var group = new PursuitGroup("target");
            var target = Target(); target.Targeting = false; target.HuntingPlayers = true;
            for (var i = 0; i <= 3; i++)
            {
                group.Sample(new[] { target }, V3.Zero, i);
                Assert.False(group.Assess(V3.Zero, i).Isolated);
            }
        }

        [Fact]
        public void SelectionDoesNotPreferAnEnemyBehindAnother()
        {
            Assert.True(double.IsPositiveInfinity(PursuitGroup.EdgeScore(V3.Zero, new V3(20, 0, 0), new[] { new V3(10, 0, 0) })));
            Assert.True(double.IsFinite(PursuitGroup.EdgeScore(V3.Zero, new V3(10, 0, 0), new[] { new V3(20, 0, 0) })));
        }
    }
}
