using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public sealed class RouteRefusalsTests
    {
        [Fact]
        public void ActualRefusalChangesTheNextPlanAndStillLetsTheBodyBackOut()
        {
            var memory = new RouteRefusals();
            var start = new V3(0, 34.5, 0);
            memory.Remember(start, new V3(.5, 34.4, 0), 0, () => true);
            Assert.True(memory.Blocks(start, new V3(.5, 34.4, 0), 1));
            Assert.False(memory.Blocks(start, new V3(-.5, 34.4, 0), 1));
            Assert.False(memory.Blocks(start, new V3(0, 34.5, .5), 1));
            V3? Probe(V3 from, V3 to) => memory.Blocks(from, to, 1) ? (V3?)null : to;
            var route = new LocalRoute(start, new V3(2, 34.5, 0), Probe);
            while (route.Status == RouteStatus.Searching) route.Advance();
            Assert.Equal(RouteStatus.Found, route.Status);
            Assert.Contains(route.Path, p => System.Math.Abs(p.Z) >= .5);
        }
        [Fact]
        public void ChangedGeometryExpiryAndAnotherStoreyDoNotInheritARefusal()
        {
            var memory = new RouteRefusals(); var current = true;
            var from = new V3(0, 34, 0); var to = new V3(.5, 34, 0);
            memory.Remember(from, to, 0, () => current);
            Assert.False(memory.Blocks(from + new V3(0, 2, 0), to + new V3(0, 2, 0), 1));
            current = false;
            Assert.False(memory.Blocks(from, to, 1));
            memory.Remember(from, to, 2, () => true);
            Assert.False(memory.Blocks(from, to, 47));
            Assert.Equal(0, memory.Count);
        }
        [Fact]
        public void EvidenceIsBoundedAndClearedForANewGoal()
        {
            var memory = new RouteRefusals();
            for (var i = 0; i < 100; i++) memory.Remember(new V3(i, 0, 0), new V3(i + .5, 0, 0), 0, () => true);
            Assert.Equal(RouteRefusals.Limit, memory.Count);
            memory.Clear(); Assert.Equal(0, memory.Count);
        }
    }
}
