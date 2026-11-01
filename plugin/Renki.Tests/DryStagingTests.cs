using System.Linq;
using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public sealed class DryStagingTests
    {
        [Fact]
        public void CapturedBankCanStageOnExplicitlySyntheticCheckedHigherGroundBeforeRaisedGoalIsSolved()
        {
            // Positions come from the retained return; the support below is
            // synthetic and does not assert native bank traversability.
            var start = new V3(-51.11, 31.35, 8.77);
            var goal = new V3(-35.9, 37.66, 6.53);
            V3? Probe(V3 from, V3 desired)
            {
                var x = desired.X - start.X; var z = desired.Z - start.Z;
                if (x > .5 && System.Math.Abs(z) < 2) return null; // steep direct face
                var at = new V3(desired.X, start.Y + .12 * x + .08 * z, desired.Z);
                return DryTravel.Clear(at, 30, DryTravel.TravelMargin) && System.Math.Abs(at.Y - from.Y) <= .3 ? at : (V3?)null;
            }
            var route = new LocalRoute(start, goal, Probe, stagingEligible: at => DryTravel.Clear(at, 30, DryStaging.WaterMargin));
            route.Advance(300);
            Assert.Equal(RouteStatus.Searching, route.Status);
            Assert.Null(route.Path);
            var stage = route.StagingPath;
            Assert.NotNull(stage);
            Assert.Equal(start, stage[0]);
            Assert.True(stage.Any(p => System.Math.Abs(p.Z - start.Z) >= 2));
            Assert.True(DryTravel.Clear(stage.Last(), 30, DryStaging.WaterMargin));
            Assert.True(V3.FlatDistance(start, goal) - V3.FlatDistance(stage.Last(), goal) >= DryStaging.MinimumProgress);
            var length = 0d;
            for (var i = 1; i < stage.Count; i++)
            {
                Assert.Equal(stage[i], Probe(stage[i - 1], stage[i]));
                length += (stage[i] - stage[i - 1]).Length;
            }
            Assert.InRange(length, DryStaging.MinimumDistance, DryStaging.MaximumCost);
        }

        [Fact]
        public void EligibilityAndGoalHintsCannotInventAPathThroughMissingSupport()
        {
            var route = new LocalRoute(V3.Zero, new V3(10, 3, 0), (_, __) => null, stagingEligible: _ => true);
            route.Advance(10);
            Assert.Equal(RouteStatus.NoRoute, route.Status);
            Assert.Null(route.StagingPath);
        }

        [Fact]
        public void ReplansCannotRenewTheSameGoalsTwoLegBudget()
        {
            var budget = new DryStaging();
            var path = new[] { V3.Zero, new V3(3, 1.6, 0) };
            Assert.False(budget.Take(1.9, true, path));
            Assert.False(budget.Take(2, false, path));
            Assert.False(budget.Take(2, true, null));
            Assert.False(budget.Take(double.NaN, true, path));
            Assert.True(budget.Take(2, true, path));
            Assert.True(budget.Take(10, true, path));
            Assert.False(budget.Available);
            Assert.False(budget.Take(100, true, path));
            Assert.Equal(2, budget.Used);
        }

        [Fact]
        public void OrdinarySearchDoesNotAcquireStagingWithoutOptIn()
        {
            var route = new LocalRoute(V3.Zero, new V3(10, 20, 0), (from, to) => new V3(to.X, 0, to.Z));
            route.Advance(300);
            Assert.Null(route.StagingPath);
            Assert.Equal(RouteStatus.Searching, route.Status);
        }
    }
}
