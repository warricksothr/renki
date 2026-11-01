using System;
using System.Collections.Generic;
using Renki.Movement;
using Renki.Tasks;
using Xunit;

namespace Renki.Tests
{
    public sealed class FleeBearingsTests
    {
        static RouteStatus Finish(FleeBearings search)
        {
            while (search.Status == RouteStatus.Searching) search.Advance(1);
            return search.Status;
        }

        [Fact]
        public void CapturedShorePositionChoosesCheckedAwayTravelAlongASyntheticDryShelf()
        {
            // Captured positions; this shelf is synthetic, not captured terrain.
            var start = new V3(-27.42, 30.85, -20.58);
            var threat = new V3(5.23, 33.09, -18.84);
            var threats = new[] { new HostileClearance.Threat(threat),
                new HostileClearance.Threat(new V3(-52.63, 35.77, -59.34), "Boar"),
                new HostileClearance.Threat(new V3(13.39, 33.03, 3.97)),
                new HostileClearance.Threat(new V3(24.22, 32.79, -7.18)) };
            bool Dry(V3 p) => p.X >= -27.5 - Math.Max(0, p.Z - start.Z) * .6;
            V3? Probe(V3 a, V3 b) => Dry(b) && HostileClearance.Clear(a, b, threats) ? b : (V3?)null;
            var search = new FleeBearings(start, threat, 35, Probe);
            Assert.Equal(RouteStatus.Found, Finish(search));
            Assert.NotEqual(0, search.Degrees);
            Assert.True(search.Goal.Z > start.Z);
            var direction = (search.Goal - start).Flat.Normalized;
            var previous = start;
            for (var i = 1; i <= FleeBearings.PrefixSteps; i++)
            {
                var at = start + direction * (i * .5);
                Assert.NotNull(Probe(previous, at));
                Assert.True(V3.FlatDistance(at, threat) > V3.FlatDistance(previous, threat));
                previous = at;
            }
            Assert.InRange(search.Samples, 1, FleeBearings.CandidateLimit * FleeBearings.PrefixSteps);
        }

        [Fact]
        public void PhysicalRefusalAcrossEveryBearingIsTerminalAndBudgeted()
        {
            var calls = 0;
            var search = new FleeBearings(V3.Zero, new V3(10, 0, 0), 35,
                (a, b) => { calls++; return null; });
            search.Advance(2);
            Assert.Equal(2, calls);
            Assert.Equal(RouteStatus.NoRoute, Finish(search));
            Assert.Equal(FleeBearings.CandidateLimit, calls);
            search.Advance(); search.Reject(V3.Zero, V3.Forward);
            Assert.Equal(FleeBearings.CandidateLimit, calls);
            Assert.Equal(RouteStatus.NoRoute, search.Status);
        }

        [Fact]
        public void AcceptedPrefixDoesNotAuthorizeTheRestAndRefusedGoalsConsumeTheirBearing()
        {
            var calls = 0;
            var search = new FleeBearings(V3.Zero, new V3(10, 0, 0), 35,
                (a, b) => { calls++; return b; });
            var seen = new HashSet<int>();
            for (var i = 0; i < FleeBearings.CandidateLimit; i++)
            {
                Assert.Equal(RouteStatus.Found, Finish(search));
                Assert.True(seen.Add(search.Degrees));
                Assert.True(V3.FlatDistance(V3.Zero, search.Goal) > FleeBearings.PrefixSteps * .5);
                // The normal walker rejects the remaining goal. New position
                // and moving threat must not replenish the bearing budget.
                search.Reject(new V3(0, 0, i), new V3(10 + i, 0, 0));
            }
            Assert.Equal(RouteStatus.NoRoute, search.Status);
            Assert.Equal(FleeBearings.CandidateLimit * FleeBearings.PrefixSteps, calls);
        }

        [Fact]
        public void ProjectionKeepsActualStepHeightsWithoutTrustingTheSpeculativeGoalHeight()
        {
            var search = new FleeBearings(V3.Zero, new V3(10, 0, 0), 35,
                (a, b) => new V3(b.X, a.Y + .2, b.Z));
            Assert.Equal(RouteStatus.Found, Finish(search));
            Assert.Equal(0, search.Goal.Y); // Guidance only; the walker probes its actual floor.
            Assert.Equal(FleeBearings.PrefixSteps, search.Samples);
        }

        [Fact]
        public void RunningAwayFromOneThreatCannotApproachAnotherLoadedThreat()
        {
            var threats = new[] { new HostileClearance.Threat(new V3(10, 0, 0)),
                new HostileClearance.Threat(new V3(-8, 0, -8)) };
            var search = new FleeBearings(V3.Zero, threats[0].Pos, 35,
                (a, b) => HostileClearance.Clear(a, b, threats) ? b : (V3?)null);
            Assert.Equal(RouteStatus.Found, Finish(search));
            Assert.Equal(-60, search.Degrees);
            Assert.True(search.Goal.Z > 0);
        }

        [Fact]
        public void CancellationCannotBeRevivedOrIssueAnotherProbe()
        {
            var calls = 0;
            var search = new FleeBearings(V3.Zero, V3.Forward, 35, (a, b) => { calls++; return b; });
            search.Advance(1); search.Cancel();
            search.Reject(V3.Zero, V3.Forward); search.Advance(100);
            Assert.Equal(1, calls);
            Assert.Equal(RouteStatus.NoRoute, search.Status);
        }

        [Fact]
        public void AlternativeBearingsDoNotResetTheEnclosingEncounterDeadline()
        {
            var encounter = new TravelRecovery();
            encounter.Decide(0, true, false);
            var search = new FleeBearings(V3.Zero, V3.Forward, 35, (a, b) => b);
            Finish(search); search.Reject(V3.Zero, V3.Forward); Finish(search);
            Assert.Equal(TravelMode.Escape, encounter.Decide(29, true, false));
            Assert.Equal(TravelMode.Refused, encounter.Decide(30, true, false));
            search.Cancel();
            Assert.Equal(1, encounter.Attempts);
        }

        [Fact]
        public void SelectedBearingCanReaimAwayFromAMovingThreatWithoutChangingItsBudget()
        {
            var start = new V3(5, 4, 3);
            foreach (var degrees in new[] { 0, 30, -30, 60, -60, 90, -90 })
            {
                var threat = new V3(15, 0, 2);
                var goal = FleeBearings.AtBearing(start, threat, 35, degrees);
                Assert.Equal(40, V3.FlatDistance(goal, threat), 6);
                Assert.Equal(start.Y, goal.Y);
                Assert.True(V3.Dot((start - threat).Flat, (goal - start).Flat) >= -1e-10);
            }
        }

        [Fact]
        public void BodyMovementCannotTurnATangentEscapeIntoCirclingAStaticThreat()
        {
            var threat = new V3(10, 0, 0);
            var search = new FleeBearings(V3.Zero, threat, 35, (a, b) => b);
            while (search.Degrees != 90)
            { Finish(search); search.Reject(V3.Zero, threat); }
            Finish(search);
            var original = search.Goal;
            Assert.Equal(original, search.Reaim(new V3(0, 0, -5), threat));
            Assert.Equal(original, search.Reaim(new V3(0, 0, -10), threat + new V3(2, 0, 0)));
            var changed = search.Reaim(new V3(0, 0, -10), threat + new V3(4, 0, 0));
            Assert.NotEqual(original, changed);
            Assert.Equal(90, search.Degrees);
            Assert.Equal(40, V3.FlatDistance(changed, threat + new V3(4, 0, 0)), 6);
            Assert.Equal(6 * FleeBearings.PrefixSteps, search.Samples);
        }
    }
}
