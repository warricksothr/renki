using System;
using Renki.Tasks;
using Xunit;

namespace Renki.Tests
{
    public sealed class SearchAreaTravelTests
    {
        [Fact]
        public void RegionalLocalSearchReturnsToArrivalInsteadOfTheDistantDeparture()
        {
            var requested = new V3(80, 38, 0);
            var arrival = new V3(80, 31, 0);
            var origin = SearchAreaTravel.LocalOrigin(requested, arrival);
            var scout = new Scouting(origin, arrival, arrival, 0, 24);
            V3? Route(V3 p) => new V3(p.X, 31, p.Z);
            Assert.Equal(ScoutAction.Walk, scout.Decide(arrival, 60, true, 3, Route));
            Assert.True(V3.FlatDistance(scout.Destination, requested) <= 24);
            var bot = scout.Goal;
            // Exhaust outbound work; the first return is toward the actual
            // region arrival, never the departure eighty meters away.
            Assert.Equal(ScoutAction.Walk, scout.Decide(bot, 60, true, 91, Route));
            Assert.True(scout.Returning);
            Assert.True(V3.FlatDistance(scout.Goal, arrival) < 1);
            Assert.Equal(ScoutAction.Done, scout.Decide(arrival, 60, true, 92, Route));
        }

        [Fact]
        public void EntryAlternativesStayInsideTheRequestedAreaAndBudgetsAreFinite()
        {
            var center = new V3(80, 31, 0);
            Assert.Equal(center, SearchAreaTravel.Candidate(center, 0));
            for (var i = 1; i < SearchAreaTravel.Attempts; i++)
                Assert.InRange(V3.FlatDistance(center, SearchAreaTravel.Candidate(center, i)), 7.999, 8.001);
            Assert.Throws<ArgumentOutOfRangeException>(() => SearchAreaTravel.Candidate(center, SearchAreaTravel.Attempts));
            Assert.True(SearchAreaTravel.WithinBudget(V3.Zero, center, 60, 100));
            Assert.False(SearchAreaTravel.WithinBudget(V3.Zero, center, 90, 100));
            Assert.False(SearchAreaTravel.WithinBudget(V3.Zero, center, 60, 201));
            Assert.False(SearchAreaTravel.WithinBudget(V3.Zero, new V3(149, 31, 0), 60, 150));
            Assert.False(SearchAreaTravel.WithinBudget(V3.Zero, center, double.NaN, 100));
        }

        [Fact]
        public void WindingMigrationCannotSpendTheReservedCombatEscapeScope()
        {
            var anchor = new V3(0, 30, 0);
            // A leg can stay near its departure and intended endpoint but
            // wander outside the durable goal's absolute search envelope.
            var departure = new V3(80, 30, 0);
            var outside = new V3(150, 30, 0);
            Assert.True(SearchAreaTravel.WithinBudget(departure, outside, 60, 80));
            Assert.False(SearchAreaTravel.WithinScope(anchor, 148, outside));
            Assert.True(SearchAreaTravel.WithinScope(anchor, 148, new V3(148, 30, 0)));
            Assert.False(SearchAreaTravel.WithinScope(anchor, 148, new V3(148, 40, 0)));
        }
    }
}
