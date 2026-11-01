using System;
using System.Linq;
using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public sealed class UnderFloorExitTests
    {
        [Fact]
        public void CapturedStartCanLeaveASyntheticOverhangDespiteMovingAwayFromTheEntrance()
        {
            // Captured positions, explicitly synthetic rising ground and
            // overhead-floor boundary. Not a claim about native geometry.
            var start = new V3(-37.07, 34.1, 7.58);
            var goal = new V3(-39.77, 36.09, 11.2);
            V3? Probe(V3 from, V3 to)
            {
                var at = new V3(to.X, start.Y + .1 * (to.X - start.X), to.Z);
                return Math.Abs(at.Y - from.Y) <= .3 ? at : (V3?)null;
            }
            bool Outside(V3 at) => at.X > start.X + 4;
            var ordinary = new LocalRoute(start, goal, Probe, arrivalLevel: .6, heightAware: true, stagingEligible: Outside);
            var escape = new LocalRoute(start, goal, Probe, arrivalLevel: .6, heightAware: true,
                stagingEligible: Outside, underFloorExit: true);
            ordinary.Advance(500); escape.Advance(500);
            Assert.Null(ordinary.StagingPath);
            Assert.Equal(RouteStatus.Searching, escape.Status);
            Assert.Null(escape.Path);
            var path = escape.StagingPath;
            Assert.NotNull(path);
            Assert.True(Outside(path.Last()));
            Assert.True(V3.FlatDistance(path.Last(), goal) > V3.FlatDistance(start, goal));
            var cost = 0d;
            for (var i = 1; i < path.Count; i++)
            {
                Assert.Equal(path[i], Probe(path[i - 1], path[i]));
                cost += (path[i] - path[i - 1]).Length;
            }
            Assert.True(UnderFloorExit.Candidate(start, goal, path.Last(), cost));
            Assert.InRange(cost, DryStaging.MinimumDistance, DryStaging.MaximumCost);
        }

        [Fact]
        public void MissingSupportOrNoObservedExitCannotCreateAnEscape()
        {
            var goal = new V3(0, 3, 0);
            var missing = new LocalRoute(V3.Zero, goal, (_, __) => null, stagingEligible: _ => true, underFloorExit: true);
            missing.Advance(50);
            Assert.Null(missing.StagingPath);
            var covered = new LocalRoute(V3.Zero, goal, (from, at) => new V3(at.X, .1 * at.X, at.Z),
                stagingEligible: _ => false, underFloorExit: true);
            covered.Advance(300);
            Assert.Null(covered.StagingPath);
        }

        [Fact]
        public void EscapeNeedsHeightGainAndKeepsRetreatAndPathBounds()
        {
            var goal = new V3(0, 3, 0);
            Assert.True(UnderFloorExit.Candidate(V3.Zero, goal, new V3(3, .3, 0), 4));
            Assert.False(UnderFloorExit.Candidate(V3.Zero, V3.Zero, new V3(3, .3, 0), 4));
            Assert.False(UnderFloorExit.Candidate(V3.Zero, goal, new V3(3, 0, 0), 4));
            Assert.False(UnderFloorExit.Candidate(V3.Zero, goal, new V3(2, .3, 0), 4));
            Assert.False(UnderFloorExit.Candidate(V3.Zero, goal, new V3(9, .3, 0), 10));
            Assert.False(UnderFloorExit.Candidate(V3.Zero, goal, new V3(3, .3, 0), 13));
            var budget = new DryStaging();
            var path = new[] { V3.Zero, new V3(3, .3, 0) };
            Assert.True(budget.Take(2, true, path)); // shore or under-floor exit share this counter
            Assert.True(budget.Take(2, true, path));
            Assert.False(budget.Take(2, true, path));
        }
    }
}
