using Renki.Movement;
using Renki.Protocol;
using Newtonsoft.Json.Linq;
using Xunit;

namespace Renki.Tests
{
    public sealed class HostileClearanceTests
    {
        [Fact]
        public void SafeEndpointsCannotHideAThreatInTheMiddleOfALeg()
        {
            var threats = new[] { V3.Zero };
            Assert.False(HostileClearance.Clear(new V3(-30, 0, 0), new V3(30, 0, 0), threats));
            Assert.False(HostileClearance.Clear(new V3(-45, 0, 25), new V3(45, 0, 25), threats));
            Assert.True(HostileClearance.Clear(new V3(-45, 0, 41), new V3(45, 0, 41), threats));
            Assert.True(HostileClearance.Clear(new V3(-45, 0, 0), new V3(-40, 0, 0), threats));
        }
        [Fact]
        public void StartingNearOneThreatStillAllowsAnEscapeButNeverThroughAnother()
        {
            var threats = new[] { V3.Zero };
            Assert.True(HostileClearance.Clear(new V3(10, 0, 0), new V3(11, 0, 0), threats));
            Assert.True(HostileClearance.Clear(new V3(10, 0, 0), new V3(10, 0, 0), threats));
            Assert.False(HostileClearance.Clear(new V3(10, 0, 0), new V3(9, 0, 0), threats));
            Assert.False(HostileClearance.Clear(new V3(10, 0, 0), new V3(-30, 0, 0), threats));
            Assert.False(HostileClearance.Clear(new V3(10, 0, 0), new V3(11, 0, 0), new[] { V3.Zero, new V3(20, 0, 0) }));
            Assert.True(HostileClearance.Clear(V3.Zero, new V3(1, 0, 0), threats));
        }
        [Fact]
        public void LocalPlanningTakesADetourInsteadOfEnteringTheThreatArea()
        {
            var threats = new[] { V3.Zero };
            var start = new V3(-45, 0, 0);
            V3? Probe(V3 from, V3 to) => HostileClearance.Clear(from, to, threats) ? to : (V3?)null;
            var route = new LocalRoute(start, new V3(45, 0, 0), Probe);
            while (route.Status == RouteStatus.Searching) route.Advance();
            Assert.Equal(RouteStatus.Found, route.Status);
            Assert.Contains(route.Path, p => System.Math.Abs(p.Z) > 10);
            var previous = start;
            foreach (var point in route.Path) { Assert.True(HostileClearance.Clear(previous, point, threats)); previous = point; }
        }

        [Fact]
        public void OrdinaryShoreCreaturesDoNotAcquireTheGreylingsWholeExclusionArea()
        {
            var start = new V3(9.78, 31.29, 52.14);
            var end = new V3(50, 34, 27.5);
            var threats = new[]
            {
                new HostileClearance.Threat(new V3(19.63, 30.8, 63.58), "Neck"),
                new HostileClearance.Threat(new V3(-1.38, 31.35, 71.72), "Boar"),
                new HostileClearance.Threat(new V3(23.05, 32.26, 88.98), "Neck"),
                new HostileClearance.Threat(new V3(.12, 32.34, 90.93), "Boar"),
                new HostileClearance.Threat(new V3(-35.97, 29.56, 65.09), "Neck"),
                new HostileClearance.Threat(new V3(44.66, 33.27, 84.87), "Greyling"),
            };
            // The first straight step still approaches the nearby Neck.
            // A checked northern detour must be possible instead.
            Assert.False(HostileClearance.Clear(start, end, threats));
            Assert.False(HostileClearance.Clear(start, new V3(19.63, 31, 63.58), threats));
            Assert.False(HostileClearance.Clear(start, new V3(44.66, 33, 50), threats));
            Assert.False(HostileClearance.Clear(new V3(-45, 0, 25), new V3(45, 0, 25),
                new[] { new HostileClearance.Threat(V3.Zero, "Neck", 2) }));
            V3? Probe(V3 from, V3 to) => HostileClearance.Clear(from, to, threats) ? to : (V3?)null;
            var route = new LocalRoute(start, end, Probe);
            while (route.Status == RouteStatus.Searching) route.Advance();
            Assert.Equal(RouteStatus.Found, route.Status);
            var previous = start;
            foreach (var point in route.Path) { Assert.True(HostileClearance.Clear(previous, point, threats)); previous = point; }
        }
        [Fact]
        public void NativeProtocolKeepsAvoidanceOptIn()
        {
            var args = Assert.IsType<MoveToArgs>(TaskArgs.Parse(new TaskRequest { Task = "move_to", Args = JObject.Parse("{\"to\":[1,2,3],\"dry\":true,\"avoidHostiles\":true}") }));
            Assert.True(args.AvoidHostiles);
            Assert.Throws<BadArgsException>(() => TaskArgs.Parse(new TaskRequest { Task = "move_to", Args = JObject.Parse("{\"to\":[1,2,3],\"avoidHostiles\":true}") }));
            Assert.False(Assert.IsType<MoveToArgs>(TaskArgs.Parse(new TaskRequest { Task = "move_to", Args = JObject.Parse("{\"to\":[1,2,3]}") })).AvoidHostiles);
        }
    }
}
