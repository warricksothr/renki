using System;
using System.Collections.Generic;
using Renki.Tasks;
using Xunit;

namespace Renki.Tests
{
    public sealed class ScoutingTests
    {
        static V3? Clear(V3 at) => at;
        static Scouting Search(double radius = 60) => new Scouting(V3.Zero, V3.Zero, V3.Zero, 0, radius);

        [Fact]
        public void OuterSearchProjectsShortLegsIntoFartherGroundAndReturnsToObservedOrigin()
        {
            var s = new Scouting(V3.Zero, V3.Zero, V3.Zero, 0, 60, 0, 60);
            var bot = V3.Zero; var farthest = 0.0; var ended = false;
            V3? Terrain(V3 at) => new V3(at.X, at.X * .02, at.Z);
            for (var i = 0; i < 120; i++)
            {
                var action = s.Decide(bot, 60, true, 2 + i * .2, Terrain);
                if (action == ScoutAction.Done) { ended = true; break; }
                Assert.Equal(ScoutAction.Walk, action);
                Assert.InRange(V3.FlatDistance(s.Goal, bot), 0, Scouting.LegLength + .001);
                Assert.InRange(s.Goal.Length, 0, Scouting.OuterReach + .25);
                Assert.Equal(s.Goal.X * .02, s.Goal.Y, 5);
                farthest = Math.Max(farthest, s.Goal.Length);
                bot = s.Goal;
            }
            Assert.True(ended);
            Assert.True(s.Returning);
            Assert.InRange(farthest, 110, Scouting.OuterReach + .25);
            Assert.InRange(bot.Length, 0, 1.5);
            Assert.InRange(s.Travel, 220, Scouting.TravelLimit);
        }

        [Fact]
        public void OuterTravelTriesOneBoundedAlternativePerDecisionAndKeepsProgress()
        {
            var s = new Scouting(V3.Zero, V3.Zero, V3.Zero, 0, 60, 0, 40);
            var probes = 0;
            V3? BlockDirect(V3 at) { probes++; return Math.Abs(at.Z) < .1 ? (V3?)null : at; }
            Assert.Equal(ScoutAction.Wait, s.Decide(V3.Zero, 60, true, 2, BlockDirect));
            Assert.Equal(1, probes);
            Assert.Equal(ScoutAction.Walk, s.Decide(V3.Zero, 60, true, 2.5, BlockDirect));
            Assert.Equal(2, probes);
            Assert.True(s.Goal.Z > 0);
            Assert.InRange(s.Goal.X, 0, Scouting.LegLength);
            Assert.True(V3.FlatDistance(s.Goal, new V3(60, 0, 0)) < 60);
            Assert.Equal(1, s.Attempts);
        }

        [Theory]
        [InlineData(false)]
        [InlineData(true)]
        public void OuterFanIsFiniteAndBackwardProjectionCannotEarnTravel(bool backward)
        {
            var s = new Scouting(V3.Zero, V3.Zero, V3.Zero, 0, 60, 0, 40);
            var probes = 0;
            V3? Route(V3 at) { probes++; return backward ? new V3(-10, 0, 0) : (V3?)null; }
            for (var i = 0; i < Scouting.LegAlternatives; i++)
                Assert.Equal(ScoutAction.Wait, s.Decide(V3.Zero, 60, true, 2 + i * .5, Route));
            Assert.Equal(ScoutAction.Done, s.Decide(V3.Zero, 60, true, 5, Route));
            Assert.Equal(Scouting.LegAlternatives, probes);
            Assert.Equal(0, s.CompletedLegs);
            Assert.Equal(1, s.Recoveries);
        }

        [Fact]
        public void OuterReturnNeverFansAroundAnUnsafeBreadcrumb()
        {
            var s = new Scouting(V3.Zero, V3.Zero, V3.Zero, 0, 60, 0, 60);
            s.Decide(V3.Zero, 60, true, 2, Clear);
            var probes = 0;
            Assert.Equal(ScoutAction.Abort, s.Decide(s.Goal, 29, true, 3, _ => { probes++; return null; }));
            Assert.Equal(1, probes);
            Assert.Contains("cannot return", s.Reason);
        }

        [Fact]
        public void OuterTravelReservesReturnTimeWithoutExtendingAnyBudget()
        {
            var s = new Scouting(V3.Zero, V3.Zero, V3.Zero, 0, 60, 0, 60);
            var bot = V3.Zero;
            for (var i = 0; i < 6; i++) { s.Decide(bot, 60, true, 2 + i, Clear); bot = s.Goal; }
            Assert.False(s.Returning);
            Assert.Equal(ScoutAction.Walk, s.Decide(bot, 60, true, 80, Clear));
            Assert.True(s.Returning);
            Assert.Equal(ScoutAction.Abort, s.Decide(bot, 60, true, Scouting.ReturnLimit, Clear));
            Assert.Contains("return time", s.Reason);
            Assert.Throws<ArgumentOutOfRangeException>(() => new Scouting(V3.Zero, V3.Zero, V3.Zero, 0, 60, null, 40));
            Assert.Throws<ArgumentOutOfRangeException>(() => new Scouting(V3.Zero, V3.Zero, V3.Zero, 0, 80, 0, 60));
        }

        [Theory]
        [InlineData(0)]
        [InlineData(1)]
        [InlineData(4)]
        [InlineData(7)]
        public void SectorSearchAdvancesAlongItsBearingAndReturnsOnCheckedLegs(int sector)
        {
            var origin = new V3(100, 30, -200);
            var s = new Scouting(origin, origin, origin, 0, 48, sector);
            var bearing = (Scouting.Candidate(origin, sector * 3, 48) - origin).Normalized;
            var bot = origin; var farthest = 0.0; var ended = false;
            for (var i = 0; i < 100; i++)
            {
                var action = s.Decide(bot, 60, true, 2 + i * .2, Clear);
                if (action == ScoutAction.Done) { ended = true; break; }
                Assert.Equal(ScoutAction.Walk, action);
                Assert.InRange((s.Goal - bot).Length, 0, Scouting.LegLength + 1e-6);
                Assert.InRange((s.Goal - origin).Length, 0, 48.001);
                Assert.InRange(V3.Dot(s.Goal - origin, bearing), -.001, 48.001);
                Assert.InRange(Scouting.RouteDistance(origin, origin + bearing * 48, s.Goal), 0, .001);
                if (!s.Returning) farthest = Math.Max(farthest, (s.Goal - origin).Length);
                bot = s.Goal;
            }
            Assert.True(ended);
            Assert.True(s.Returning);
            Assert.Equal(3, s.Attempts);
            Assert.True(s.CompletedLegs > 0);
            Assert.InRange(farthest, 47.999, 48.001);
            Assert.InRange((bot - origin).Length, 0, 1.5);
        }

        [Fact]
        public void BlockedSectorDoesNotWanderIntoAnotherBearingOrClaimCoverage()
        {
            var s = new Scouting(V3.Zero, V3.Zero, V3.Zero, 0, 60, 4);
            for (var i = 0; i < 3; i++)
                Assert.Equal(ScoutAction.Wait, s.Decide(V3.Zero, 60, true, 2 + i * .1, _ => null));
            Assert.Equal(ScoutAction.Done, s.Decide(V3.Zero, 60, true, 3, _ => throw new Exception("no more probes")));
            Assert.Equal(0, s.CompletedLegs);
            Assert.Equal(3, s.Attempts);
            Assert.Throws<ArgumentOutOfRangeException>(() => new Scouting(V3.Zero, V3.Zero, V3.Zero, 0, 60, 8));
        }

        [Fact]
        public void SearchProgressesOutwardWithoutReturningAfterEveryPoint()
        {
            var s = Search(); var bot = V3.Zero;
            Assert.Equal(ScoutAction.Wait, s.Decide(bot, 60, true, 1, _ => throw new Exception("too early")));
            for (var i = 0; i < 6; i++)
            {
                Assert.Equal(ScoutAction.Walk, s.Decide(bot, 60, true, 2 + i, Clear));
                Assert.False(s.Returning);
                Assert.InRange((s.Goal - bot).Length, 0, Scouting.LegLength);
                Assert.True(s.Goal.X > bot.X);
                bot = s.Goal;
            }
            Assert.Equal(60, bot.X);
            Assert.Equal(3, s.Attempts);
            Assert.True(s.CompletedLegs > 0);
        }

        [Fact]
        public void BlockedCandidatesAreFiniteAndDoNotCauseMovement()
        {
            var s = Search(); var probes = 0;
            for (var i = 0; i < Scouting.CandidateLimit; i++)
                Assert.Equal(ScoutAction.Wait, s.Decide(V3.Zero, 60, true, 2 + i * .1, _ => { probes++; return null; }));
            Assert.Equal(ScoutAction.Done, s.Decide(V3.Zero, 60, true, 5, _ => throw new Exception("already home")));
            Assert.Equal(Scouting.CandidateLimit, probes);
            Assert.Equal(0, s.CompletedLegs);
        }

        [Theory]
        [InlineData(false)]
        [InlineData(true)]
        public void TimeOrStaminaEndsSearchWithCheckedIntermediateReturnLegs(bool tired)
        {
            var s = Search(); var bot = V3.Zero;
            for (var i = 0; i < 4; i++) { s.Decide(bot, 60, true, 2 + i, Clear); bot = s.Goal; }
            Assert.Equal(40, bot.X);
            var now = tired ? 6 : Scouting.WorkLimit;
            while (bot.X > 1.5)
            {
                Assert.Equal(ScoutAction.Walk, s.Decide(bot, tired ? 29 : 60, true, now++, Clear));
                Assert.True(s.Returning);
                Assert.True(s.Goal.X < bot.X);
                Assert.InRange((s.Goal - bot).Length, 0, Scouting.LegLength);
                bot = s.Goal;
            }
            Assert.Equal(ScoutAction.Done, s.Decide(bot, 60, true, now, Clear));
        }

        [Fact]
        public void AcceptedRouteAloneDoesNotEstablishSearchCoverage()
        {
            var s = Search();
            Assert.Equal(ScoutAction.Walk, s.Decide(V3.Zero, 60, true, 2, Clear));
            Assert.Equal(0, s.CompletedLegs);
            Assert.Equal(ScoutAction.Walk, s.Decide(V3.Zero, 60, true, 3, Clear));
            Assert.Equal(0, s.CompletedLegs);
            Assert.Equal(ScoutAction.Walk, s.Decide(s.Goal, 60, true, 4, Clear));
            Assert.Equal(1, s.CompletedLegs);
        }

        [Fact]
        public void FailedReturnNeverFallsBackToUncheckedMovement()
        {
            var s = Search(); s.Decide(V3.Zero, 60, true, 2, Clear);
            Assert.Equal(ScoutAction.Abort, s.Decide(s.Goal, 29, true, 3, _ => null));
            Assert.Contains("cannot return", s.Reason);
        }

        [Fact]
        public void ChangedRouteAndNoProgressTryAnotherBearingWithoutResettingBudgets()
        {
            var s = Search(); s.Decide(V3.Zero, 60, true, 2, Clear);
            Assert.Equal(ScoutAction.Wait, s.Decide(V3.Zero, 60, true, 3, _ => null));
            Assert.Equal(1, s.Recoveries);
            Assert.Equal(ScoutAction.Walk, s.Decide(V3.Zero, 60, true, 3.5, Clear));
            Assert.True(s.Goal.Z > 0);
            Assert.Equal(4, s.Attempts);
            s = Search(); s.Decide(V3.Zero, 60, true, 2, Clear);
            Assert.Equal(ScoutAction.Wait, s.Decide(V3.Zero, 60, true, 6, Clear));
            Assert.Contains("no progress", s.Reason);
            Assert.Equal(ScoutAction.Abort, s.Decide(V3.Zero, 60, true, Scouting.ReturnLimit, Clear));
        }

        [Fact]
        public void RepeatedWalkerFailuresExhaustFiniteRecoveryAndNeverCountAsCoverage()
        {
            var s = Search();
            for (var i = 0; i < Scouting.RecoveryLimit; i++)
            {
                Assert.Equal(ScoutAction.Walk, s.Decide(V3.Zero, 60, true, 2 + i, Clear));
                Assert.Equal(ScoutAction.Wait, s.RejectRoute("walker refused"));
            }
            Assert.Equal(Scouting.RecoveryLimit, s.Recoveries);
            Assert.Equal(0, s.CompletedLegs);
            Assert.Equal(ScoutAction.Done, s.Decide(V3.Zero, 60, true, 10, _ => throw new Exception("already at return point")));
            Assert.True(s.Returning);
        }

        [Fact]
        public void ReturnRetracesObservedBendAndRevalidatesEveryConnector()
        {
            var s = Search();
            s.Decide(V3.Zero, 60, true, 2, Clear);
            var trail = new[] { new V3(8, 0, 0), new V3(16, 0, 0), new V3(16, 0, 8), new V3(16, 0, 16) };
            for (var i = 0; i < trail.Length - 1; i++)
                Assert.Equal(ScoutAction.Walk, s.Decide(trail[i], 60, true, 2.5 + i * .5, Clear));
            var bot = trail[trail.Length - 1];
            var now = 4.0;
            for (var i = trail.Length - 2; i >= -1; i--)
            {
                var expected = i >= 0 ? trail[i] : V3.Zero;
                Assert.Equal(ScoutAction.Walk, s.Decide(bot, 29, true, now++, at =>
                {
                    Assert.Equal(expected, at);
                    return at;
                }));
                Assert.Equal(expected, s.Goal);
                bot = s.Goal;
            }
            var outboundLegs = s.CompletedLegs;
            Assert.Equal(ScoutAction.Done, s.Decide(bot, 29, true, now, _ => throw new Exception("already home")));
            Assert.Equal(outboundLegs, s.CompletedLegs);
        }

        [Fact]
        public void WindingTrailReservesReturnTravelBeforeRadialEstimateWould()
        {
            var s = Search(); s.Decide(V3.Zero, 60, true, 2, Clear);
            var loop = new[] { new V3(16, 0, 0), new V3(32, 0, 0), new V3(32, 0, 16), new V3(16, 0, 16) };
            var bot = V3.Zero; var now = 2.0;
            for (var i = 0; i < 20 && !s.Returning; i++)
            {
                bot = loop[i % loop.Length];
                Assert.Equal(ScoutAction.Walk, s.Decide(bot, 60, true, now += .25, Clear));
            }
            Assert.True(s.Returning);
            Assert.True(s.Travel < Scouting.TravelLimit - 60 * 1.5);
            Assert.True(s.Travel + s.ReturnDistance <= Scouting.TravelLimit);
            while ((bot - V3.Zero).Length > 1.5)
            {
                bot = s.Goal;
                var action = s.Decide(bot, 60, true, now += .25, Clear);
                Assert.True(action == ScoutAction.Walk || action == ScoutAction.Done);
            }
            Assert.True(s.Travel <= Scouting.TravelLimit);
            Assert.InRange(s.TrailPoints, 1, Scouting.TrailLimit);
        }

        [Fact]
        public void ThreatOverridesRecoveryAndChangedReturnStillAborts()
        {
            var s = Search(); s.Decide(V3.Zero, 60, true, 2, Clear);
            s.RejectRoute("outbound route changed");
            Assert.Equal(ScoutAction.Abort, s.Decide(V3.Zero, 60, false, 3, _ => throw new Exception("unsafe")));
            s = Search(); s.Decide(V3.Zero, 60, true, 2, Clear);
            Assert.Equal(ScoutAction.Walk, s.Decide(s.Goal, 29, true, 3, Clear));
            Assert.Equal(ScoutAction.Abort, s.RejectRoute("return walker refused"));
            Assert.Equal(0, s.Recoveries);
        }

        [Theory]
        [InlineData("character")]
        [InlineData("leash")]
        [InlineData("return_time")]
        [InlineData("invalid")]
        public void ThreatAndHardBoundsOverrideScouting(string cause)
        {
            var s = Search();
            var bot = cause == "leash" ? new V3(61, 0, 0) : V3.Zero;
            Assert.Equal(ScoutAction.Abort, s.Decide(bot, cause == "invalid" ? double.NaN : 60,
                cause != "character", cause == "return_time" ? Scouting.ReturnLimit : 2, Clear));
        }

        [Fact]
        public void TotalTravelCannotBecomeAnUnboundedPatrol()
        {
            var s = Search(); ScoutAction action = ScoutAction.Wait;
            for (var i = 0; i < 8; i++)
                action = s.Decide(new V3(i % 2 == 0 ? 0 : 50, 0, 0), 60, true, 2 + i * .5, Clear);
            Assert.Equal(ScoutAction.Abort, action);
            Assert.True(s.Travel > Scouting.TravelLimit);
        }

        [Fact]
        public void CandidateScheduleCoversTheRequestedRadiusAndIsFinite()
        {
            var points = new HashSet<V3>(); var origin = new V3(100, 20, -70);
            for (var i = 0; i < Scouting.CandidateLimit; i++)
            {
                var at = Scouting.Candidate(origin, i, 80);
                Assert.True(points.Add(at));
                Assert.Equal(origin.Y, at.Y);
                Assert.InRange((at - origin).Length, 26.66, 80.001);
            }
            Assert.Equal(new V3(80, 0, 0), Scouting.Candidate(V3.Zero, 2, 80));
            Assert.Throws<ArgumentOutOfRangeException>(() => Scouting.Candidate(origin, -1));
            Assert.Throws<ArgumentOutOfRangeException>(() => Scouting.Candidate(origin, Scouting.CandidateLimit));
            Assert.Throws<ArgumentOutOfRangeException>(() => Search(81));
            var s = Search(12);
            Assert.Equal(ScoutAction.Wait, s.Decide(V3.Zero, 60, true, 2, _ => new V3(40, 0, 0)));
        }

        [Fact]
        public void CharacterClearanceIncludesTheMiddleOfAPathAndItsEndpoints()
        {
            Assert.Equal(2, Scouting.RouteDistance(V3.Zero, new V3(10, 0, 0), new V3(5, 0, 2)), 5);
            Assert.Equal(3, Scouting.RouteDistance(V3.Zero, new V3(10, 0, 0), new V3(13, 0, 0)), 5);
            Assert.Equal(3, Scouting.RouteDistance(V3.Zero, V3.Zero, new V3(0, 3, 0)), 5);
        }

        [Fact]
        public void TravelCanPassPlayersButCannotEndOnThemOrApproachCreatures()
        {
            Assert.True(Scouting.TravelSeparated(2.2, player: true, hostile: false, destination: false));
            Assert.True(Scouting.TravelSeparated(1, player: true, hostile: true, destination: false));
            Assert.False(Scouting.TravelSeparated(1, player: true, hostile: false, destination: true));
            foreach (var destination in new[] { false, true })
            {
                Assert.False(Scouting.TravelSeparated(14, player: false, hostile: true, destination: destination));
                Assert.False(Scouting.TravelSeparated(7, player: false, hostile: false, destination: destination));
                Assert.False(Scouting.TravelSeparated(double.NaN, player: true, hostile: false, destination: destination));
            }
        }
    }
}
