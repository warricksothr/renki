using System;
using System.Collections.Generic;
using Renki.Tasks;
using Xunit;

namespace Renki.Tests
{
    public sealed class ScoutingTests
    {
        static V3? Clear(V3 at) => at;
        static Scouting Search() => new Scouting(V3.Zero, V3.Zero, V3.Zero, 0);

        [Fact]
        public void SearchWaitsBrieflyThenMakesCheckedOutAndBackExcursions()
        {
            var s = Search();
            Assert.Equal(ScoutAction.Wait, s.Decide(V3.Zero, 60, true, 1, _ => throw new Exception("too early")));
            Assert.Equal(ScoutAction.Walk, s.Decide(V3.Zero, 60, true, 2, Clear));
            var first = s.Goal;
            Assert.Equal(new V3(6, 0, 0), first);
            Assert.False(s.Returning);
            Assert.Equal(ScoutAction.Walk, s.Decide(first, 60, true, 4, Clear));
            Assert.True(s.Returning);
            Assert.Equal(V3.Zero, s.Goal);
            Assert.Equal(ScoutAction.Walk, s.Decide(V3.Zero, 60, true, 6, Clear));
            Assert.False(s.Returning);
            Assert.NotEqual(first, s.Goal);
            Assert.Equal(12, s.Travel, 5);
        }

        [Fact]
        public void BlockedCandidatesAreFiniteAndDoNotCauseMovement()
        {
            var s = Search(); var probes = 0;
            for (var i = 0; i < Scouting.CandidateLimit; i++)
                Assert.Equal(ScoutAction.Wait, s.Decide(V3.Zero, 60, true, 2 + i * .1, _ => { probes++; return null; }));
            Assert.Equal(ScoutAction.Done, s.Decide(V3.Zero, 60, true, 4, _ => throw new Exception("already home")));
            Assert.Equal(Scouting.CandidateLimit, probes);
        }

        [Theory]
        [InlineData(false)]
        [InlineData(true)]
        public void TimeOrStaminaEndsTheSearchByReturningToCheckedGround(bool tired)
        {
            var s = Search();
            s.Decide(V3.Zero, 60, true, 2, Clear);
            var outside = s.Goal;
            var now = tired ? 3 : Scouting.WorkLimit;
            Assert.Equal(ScoutAction.Walk, s.Decide(outside, tired ? 29 : 60, true, now, Clear));
            Assert.True(s.Returning);
            Assert.Equal(V3.Zero, s.Goal);
            Assert.Equal(ScoutAction.Done, s.Decide(V3.Zero, 60, true, now + 2, Clear));
        }

        [Fact]
        public void FailedReturnIsExplicitAndNeverFallsBackToUncheckedMovement()
        {
            var s = Search(); s.Decide(V3.Zero, 60, true, 2, Clear);
            Assert.Equal(ScoutAction.Abort, s.Decide(s.Goal, 29, true, 3, _ => null));
            Assert.Contains("route changed", s.Reason);
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
            var bot = cause == "leash" ? new V3(13, 0, 0) : V3.Zero;
            Assert.Equal(ScoutAction.Abort, s.Decide(bot, cause == "invalid" ? double.NaN : 60,
                cause != "character", cause == "return_time" ? Scouting.ReturnLimit : 2, Clear));
        }

        [Fact]
        public void TotalTravelCannotBecomeAnUnboundedPatrol()
        {
            var s = Search();
            ScoutAction action = ScoutAction.Wait;
            for (var i = 0; i < 8; i++)
                action = s.Decide(new V3(i % 2 == 0 ? 0 : 10, 0, 0), 60, true, 2 + i * .5, Clear);
            Assert.Equal(ScoutAction.Abort, action);
            Assert.True(s.Travel > Scouting.TravelLimit);
        }

        [Fact]
        public void CandidateScheduleIsUniqueLocalAndFinite()
        {
            var points = new HashSet<V3>(); var area = new V3(100, 20, -70);
            for (var i = 0; i < Scouting.CandidateLimit; i++)
            {
                var at = Scouting.Candidate(area, i);
                Assert.True(points.Add(at));
                Assert.Equal(area.Y, at.Y);
                Assert.InRange((at - area).Length, 5.99, 10.01);
            }
            Assert.Throws<ArgumentOutOfRangeException>(() => Scouting.Candidate(area, -1));
            Assert.Throws<ArgumentOutOfRangeException>(() => Scouting.Candidate(area, Scouting.CandidateLimit));
            var s = Search();
            Assert.Equal(ScoutAction.Abort, s.Decide(V3.Zero, 60, true, 2, _ => new V3(40, 0, 0)));
        }

        [Fact]
        public void CharacterClearanceIncludesTheMiddleOfAPathAndItsEndpoints()
        {
            Assert.Equal(2, Scouting.RouteDistance(V3.Zero, new V3(10, 0, 0), new V3(5, 0, 2)), 5);
            Assert.Equal(3, Scouting.RouteDistance(V3.Zero, new V3(10, 0, 0), new V3(13, 0, 0)), 5);
            Assert.Equal(3, Scouting.RouteDistance(V3.Zero, V3.Zero, new V3(0, 3, 0)), 5);
        }
    }
}
