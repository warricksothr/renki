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
        public void BlockedWalkingRouteAndNoProgressBothTerminate()
        {
            var s = Search(); s.Decide(V3.Zero, 60, true, 2, Clear);
            Assert.Equal(ScoutAction.Abort, s.Decide(V3.Zero, 60, true, 3, _ => null));
            s = Search(); s.Decide(V3.Zero, 60, true, 2, Clear);
            Assert.Equal(ScoutAction.Abort, s.Decide(V3.Zero, 60, true, 6, Clear));
            Assert.Contains("no progress", s.Reason);
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
