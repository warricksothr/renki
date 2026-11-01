using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using Newtonsoft.Json;
using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public sealed class ThreatGuidanceTests
    {
        static readonly HostileClearance.Threat[] Threats = { new HostileClearance.Threat(V3.Zero) };
        static bool Dry(V3 at) => at.X >= -35 || Math.Abs(at.Z) >= 15;
        static V3? Probe(V3 from, V3 at) => Dry(at) && HostileClearance.Clear(from, at, Threats) ? at : (V3?)null;
        static RouteStatus Finish(LocalRoute route)
        {
            while (route.Status == RouteStatus.Searching)
            {
                var before = route.Expanded;
                route.Advance(8);
                Assert.InRange(route.Expanded - before, 0, 8);
            }
            return route.Status;
        }
        static RouteStatus Finish(RouteSearch route)
        {
            while (route.Status == RouteStatus.Searching)
            {
                var before = route.Expanded;
                route.Advance(8);
                Assert.InRange(route.Expanded - before, 0, 8);
            }
            return route.Status;
        }

        [Fact]
        public void CoastDetourCanInitiallyMoveAwayAndStillReachTheRealGoalInCheckedLegs()
        {
            var start = new V3(-45, 0, -20);
            var goal = new V3(-45, 0, 20);
            var here = start;
            var travelled = 0d;
            var legs = 0;
            var prefer = false;
            while (!Arrival.Reached(here, goal, .5) && legs < 24)
            {
                var route = new RouteSearch(hints => new LocalRoute(here, goal, Probe, within: .5,
                    guidance: hints != null ? hints.Estimate : (Func<V3, double>)null), () => new ThreatGuidance(here, goal, Threats, Dry), prefer);
                Assert.Equal(RouteStatus.Found, Finish(route));
                var hints = route.Guidance;
                prefer |= hints?.Active == true;
                Assert.InRange(route.Expanded, 1, LocalRoute.MaxNodes * 2);
                var end = route.Path.Last();
                if (legs == 0)
                {
                    Assert.True(hints.Active);
                    Assert.True(hints.RejectedHints > 0);
                    Assert.True(route.ProgressLeg);
                    Assert.True(V3.FlatDistance(end, goal) > V3.FlatDistance(start, goal));
                }
                if (route.ProgressLeg)
                {
                    Assert.True(V3.FlatDistance(here, end) >= LocalRoute.FrontierDistance);
                    Func<V3, double> remaining = hints?.Active == true
                        ? hints.Estimate : at => V3.FlatDistance(at, goal);
                    Assert.True(remaining(here) - remaining(end) >= LocalRoute.FrontierProgress);
                    Assert.False(Arrival.Reached(end, goal, .5));
                }
                var previous = here;
                foreach (var at in route.Path)
                {
                    Assert.Equal(at, Probe(previous, at));
                    travelled += (at - previous).Length;
                    previous = at;
                }
                here = end;
                legs++;
            }
            Assert.InRange(legs, 5, 24);
            Assert.True(Arrival.Reached(here, goal, .5));
            Assert.InRange(travelled, 150, 300);
        }

        [Fact]
        public void GuidanceDoesNotAuthorizeWaterUnloadedGroundOrTheWrongFloor()
        {
            var start = new V3(-45, 0, -20);
            var goal = new V3(-45, 3, 20);
            var hints = new ThreatGuidance(start, goal, Threats, Dry);
            var blocked = new LocalRoute(start, goal, (_, __) => null, guidance: hints.Estimate);
            Assert.Equal(RouteStatus.NoRoute, Finish(blocked));
            Assert.Null(blocked.Path);
            // A nearby floor still requires real height, even with an arbitrary hint.
            var floor = new LocalRoute(V3.Zero, new V3(1, 3, 0), (_, at) => at,
                maxNodes: 32, guidance: at => V3.FlatDistance(at, new V3(1, 3, 0)));
            Assert.Equal(RouteStatus.NoRoute, Finish(floor));
            Assert.True(floor.NodeLimitReached);
            Assert.Null(floor.Path);
        }

        [Fact]
        public void ARealBridgeOverKnownWaterRemainsAvailableToThePhysicalProbe()
        {
            var start = new V3(-45, 0, -20);
            var goal = new V3(-45, 0, 20);
            var here = start;
            var travelled = 0d;
            var hintCalls = 0;
            for (var leg = 0; leg < 4 && !Arrival.Reached(here, goal, .5); leg++)
            {
                var route = new RouteSearch(hints => new LocalRoute(here, goal, (from, at) =>
                    HostileClearance.Clear(from, at, Threats) ? at : (V3?)null,
                    within: .5, guidance: hints != null ? hints.Estimate : (Func<V3, double>)null), () =>
                    { hintCalls++; return new ThreatGuidance(here, goal, Threats, Dry); });
                Assert.Equal(RouteStatus.Found, Finish(route));
                for (var i = 1; i < route.Path.Count; i++) travelled += (route.Path[i] - route.Path[i - 1]).Length;
                here = route.Path.Last();
            }
            Assert.True(Arrival.Reached(here, goal, .5));
            Assert.InRange(travelled, 39, 60);
            Assert.Equal(0, hintCalls);
        }

        [Theory]
        [InlineData(false)]
        [InlineData(true)]
        public void UnchangedFailedSearchHasAtMostOneGuidedAttempt(bool prefer)
        {
            var calls = 0;
            var start = new V3(-45, 0, -20); var goal = new V3(-45, 0, 20);
            var route = new RouteSearch(hints => new LocalRoute(start, goal, (_, __) => null,
                guidance: hints != null ? hints.Estimate : (Func<V3, double>)null), () => { calls++; return new ThreatGuidance(start, goal, Threats, Dry); }, prefer);
            Assert.Equal(RouteStatus.NoRoute, Finish(route));
            Assert.Equal(1, calls);
            Assert.Null(route.Path);
            Assert.Equal(RouteStatus.NoRoute, route.Advance(8));
            Assert.Equal(1, calls);
        }

        [Fact]
        public void InactiveGuidanceRetainsItsDiagnosticsWithoutAnotherSearch()
        {
            var calls = 0;
            var start = new V3(-45, 0, -20); var goal = new V3(-45, 0, 20);
            var route = new RouteSearch(_ => new LocalRoute(start, goal, (_, __) => null), () =>
            {
                calls++;
                return new ThreatGuidance(start, goal, Threats, _ => false);
            });
            Assert.Equal(RouteStatus.NoRoute, Finish(route));
            Assert.NotNull(route.Guidance);
            Assert.False(route.Guidance.Active);
            Assert.True(route.Guidance.RejectedHints > 0);
            Assert.True(route.Guidance.HintSamples > 0);
            Assert.Equal(0, route.Phase);
            Assert.Equal(1, calls);
            Assert.Equal(RouteStatus.NoRoute, route.Advance(8));
            Assert.Equal(1, calls);
        }

        [Fact]
        public void HintGraphIsBoundedAndCannotReplayChangingThreatsAsPermission()
        {
            var sample = Enumerable.Range(0, 16).Select(i => new HostileClearance.Threat(new V3(i * 3, 0, 0))).ToArray();
            var calls = 0;
            var hints = new ThreatGuidance(new V3(-45, 0, -20), new V3(-45, 0, 20), sample,
                _ => { calls++; return true; });
            Assert.InRange(hints.Waypoints, 0, ThreatGuidance.ThreatLimit * ThreatGuidance.Spokes);
            Assert.InRange(calls, 0, ThreatGuidance.HintLimit);
            var changed = new[] { new HostileClearance.Threat(new V3(-45, 0, -20)) };
            var route = new LocalRoute(new V3(-45, 0, -20), new V3(-45, 0, 20),
                (from, to) => HostileClearance.Clear(from, to, changed) && Dry(to) ? to : (V3?)null,
                maxNodes: 64, guidance: hints.Estimate);
            Finish(route);
            if (route.Path != null)
                for (var i = 1; i < route.Path.Count; i++) Assert.True(HostileClearance.Clear(route.Path[i - 1], route.Path[i], changed));
            Assert.Throws<ArgumentException>(() => new ThreatGuidance(V3.Zero, V3.Zero,
                Enumerable.Repeat(Threats[0], ThreatGuidance.SampleLimit + 1).ToArray()));
            Assert.Throws<ArgumentException>(() => new ThreatGuidance(new V3(double.NaN, 0, 0), V3.Zero, Threats));
            Assert.Throws<ArgumentException>(() => new LocalRoute(V3.Zero, V3.Forward, (_, at) => at, guidance: _ => double.NaN));
        }

        sealed class Captured
        {
            public double[] Start { get; set; }
            public double[] Home { get; set; }
            public double[][] Threats { get; set; }
            public double[][] Terrain { get; set; }
        }

        [Fact]
        public void CapturedReturnHasAWetShortSideAndALongerGeometricDetour()
        {
            var data = JsonConvert.DeserializeObject<Captured>(File.ReadAllText(Path.Combine(AppContext.BaseDirectory, "Fixtures", "hostile-return.json")));
            V3 Pos(double[] p) => new V3(p[0], p[1], p[2]);
            var start = Pos(data.Start); var home = Pos(data.Home);
            var threats = data.Threats.Select(t => new HostileClearance.Threat(Pos(t), t[3] == 24 ? "Boar" : "Greyling")).ToArray();
            var terrain = data.Terrain.ToDictionary(p => (p[0], p[1]), p => p[2]);
            bool Hint(V3 at)
            {
                var x = Math.Floor(at.X / 4) * 4; var z = Math.Floor(at.Z / 4) * 4;
                if (!terrain.TryGetValue((x, z), out var a) || !terrain.TryGetValue((x + 4, z), out var b) ||
                    !terrain.TryGetValue((x, z + 4), out var c) || !terrain.TryGetValue((x + 4, z + 4), out var d)) return true;
                var dx = (at.X - x) / 4; var dz = (at.Z - z) / 4;
                return (a * (1 - dx) + b * dx) * (1 - dz) + (c * (1 - dx) + d * dx) * dz >= 30 + DryTravel.Margin;
            }
            var hints = new ThreatGuidance(start, home, threats, Hint);
            Assert.True(hints.Active, $"estimate={hints.InitialEstimate} rejected={hints.RejectedHints} waypoints={hints.Waypoints}");
            Assert.True(hints.HintSamples > 0);
            Assert.True(hints.InitialEstimate > V3.FlatDistance(start, home) + 80, $"estimate={hints.InitialEstimate}");
            // This proves the retained geometric bottleneck, not body/support
            // clearance along the possible detour. Live probes must establish that.
        }
    }
}
