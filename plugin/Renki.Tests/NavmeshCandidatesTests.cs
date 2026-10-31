using System;
using System.Linq;
using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public sealed class NavmeshCandidatesTests
    {
        static RouteStatus Finish(RouteValidation route)
        {
            while (route.Status == RouteStatus.Searching) route.Advance();
            return route.Status;
        }

        [Fact]
        public void WetNavmeshShortcutIsRejectedAndLocalDryDetourStillWorks()
        {
            var start = new V3(0, 30.4, 0); var goal = new V3(8, 30.4, 0);
            V3? Probe(V3 from, V3 to)
            {
                var y = to.X > 2 && to.X < 6 && Math.Abs(to.Z) < 2 ? 29.8 : 30.4;
                var support = new V3(to.X, y, to.Z);
                return DryTravel.Clear(support, 30) ? support : (V3?)null;
            }
            var candidates = new NavmeshCandidates();
            var shortcut = new[] { goal };
            var candidate = candidates.Try(start, goal, shortcut, Probe, .5);
            Assert.Equal(RouteStatus.NoRoute, Finish(candidate));
            Assert.Null(candidates.Try(start, goal, shortcut, Probe, .5));
            var local = new LocalRoute(start, goal, Probe, .5);
            while (local.Status == RouteStatus.Searching) local.Advance();
            Assert.Equal(RouteStatus.Found, local.Status);
            Assert.Contains(local.Path, p => Math.Abs(p.Z) >= 2);
            Assert.All(local.Path, p => Assert.True(DryTravel.Clear(p, 30)));
        }

        [Fact]
        public void RaisedEntranceGuidesValidationWithoutSnappingToAnotherFloor()
        {
            var start = new V3(-10, 30.4, 0); var goal = new V3(0, 34.9, 0);
            V3? Probe(V3 from, V3 to)
            {
                var y = to.X <= -7 ? 30.4 : to.X >= -4 ? 33.4 : 30.4 + to.X + 7;
                return Math.Abs(y - from.Y) <= .6 ? new V3(to.X, y, to.Z) : (V3?)null;
            }
            var candidates = new NavmeshCandidates();
            var candidate = candidates.Try(start, goal,
                new[] { new V3(-7, 30.4, 0), new V3(-4, 33.4, 0), new V3(0, 33.4, 0) }, Probe, .5);
            Assert.NotNull(candidate);
            Assert.Equal(RouteStatus.Searching, candidate.Advance(1));
            Assert.Equal(1, candidate.Samples);
            Assert.Equal(RouteStatus.Found, Finish(candidate));
            Assert.True(Arrival.Reached(candidate.Path.Last(), goal, .5));
            // Replacing the entrance with a gap invalidates the same hint.
            candidates.Reset();
            var removed = candidates.Try(start, goal,
                new[] { new V3(-7, 30.4, 0), new V3(-4, 33.4, 0), new V3(0, 33.4, 0) },
                (a, b) => b.X > -6 && b.X < -4 ? null : Probe(a, b), .5);
            Assert.Equal(RouteStatus.NoRoute, Finish(removed));
            Assert.Null(candidates.Try(start, goal, new[] { new V3(0, 30.4, 0) }, Probe, .5));
        }

        [Fact]
        public void ChangedTilesCanOfferANewCandidateButStationaryAttemptsAreBounded()
        {
            var candidates = new NavmeshCandidates();
            var goal = new V3(10, 0, 0);
            for (var i = 0; i < NavmeshCandidates.MaxAttempts; i++)
            {
                var route = new[] { new V3(5, 0, i), goal };
                var candidate = candidates.Try(V3.Zero, goal, route, (_, b) => b, .25);
                Assert.NotNull(candidate);
                Assert.Null(candidates.Try(new V3(.01, 0, 0), goal,
                    new[] { new V3(5.01, 0, i), goal }, (_, b) => b, .25));
            }
            Assert.Null(candidates.Try(V3.Zero, goal, new[] { goal }, (_, b) => b, .25));
            Assert.NotNull(candidates.Try(new V3(2, 0, 0), goal, new[] { goal }, (_, b) => b, .25));
            candidates.Reset();
            Assert.NotNull(candidates.Try(V3.Zero, goal, new[] { goal }, (_, b) => b, .25));
        }

        [Fact]
        public void SpeculationRejectsPartialRoutesAndCannotExceedTheSampleBudget()
        {
            var goal = new V3(1000, 0, 0);
            var candidate = new NavmeshCandidates().Try(V3.Zero, goal, new[] { goal }, (_, b) => b, .25);
            Assert.Equal(RouteStatus.NoRoute, Finish(candidate));
            Assert.Equal(RouteValidation.MaxSamples, candidate.Samples);
            var calls = 0;
            Assert.Null(new NavmeshCandidates().Try(V3.Zero, goal, new[] { new V3(5, 0, 0) },
                (_, b) => { calls++; return b; }, .25));
            Assert.Equal(0, calls);
        }
    }
}
