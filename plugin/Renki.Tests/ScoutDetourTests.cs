using System;
using System.Collections.Generic;
using Renki.Movement;
using Renki.Tasks;
using Xunit;

namespace Renki.Tests
{
    public sealed class ScoutDetourTests
    {
        static RouteStatus Finish(ScoutDetour plan)
        {
            for (var i = 0; i < ScoutDetour.NodeLimit + 2 && plan.Status == RouteStatus.Searching; i++)
            {
                var expanded = plan.Expanded;
                plan.Advance(i * .0001, 3);
                Assert.InRange(plan.Expanded - expanded, 0, 3);
            }
            return plan.Status;
        }

        static GroundCorridor.Failure Inspect(V3 start, IReadOnlyList<V3> path, Func<V3, V3, V3?> probe)
        {
            var cursor = new GroundCorridor.Cursor();
            var previous = start;
            foreach (var point in path)
            {
                var failed = cursor.Inspect(previous, point, probe);
                if (failed != null) return failed;
                previous = point;
            }
            return null;
        }

        [Fact]
        public void CapturedOverGradeConnectorCanUseACheckedDifferentApproach()
        {
            // Rounded coordinates from t-6 / audit-window-nvwH5p. Only the
            // rejected segment is captured; this plane models its neighborhood
            // for the regression, not a replay of the live world's heightmap.
            var start = new V3(-27.61, 30.93, -19);
            var goal = new V3(-27.49, 31.18, -18.6);
            var delta = goal - start;
            var squared = delta.X * delta.X + delta.Z * delta.Z;
            V3? Terrain(V3 from, V3 at)
            {
                var y = start.Y + delta.Y * ((at.X - start.X) * delta.X + (at.Z - start.Z) * delta.Z) / squared;
                return Math.Abs(y - from.Y) <= .6 ? new V3(at.X, y, at.Z) : (V3?)null;
            }
            var direct = GroundCorridor.Inspect(start, goal, Terrain);
            Assert.Equal("grade", direct.Reason);
            Assert.True(direct.Angle > GroundCorridor.MaxIncline);
            var plan = new ScoutDetour(start, goal, Terrain, 0);
            Assert.Equal(RouteStatus.Found, Finish(plan));
            Assert.Null(Inspect(start, plan.Path, Terrain));
            Assert.True(Arrival.Reached(plan.Path[plan.Path.Count - 1], goal, .25));
            Assert.Contains(plan.Path, p => Math.Abs(p.X - start.X) < .01 && p.Z > start.Z + .45);
        }

        [Theory]
        [InlineData(false)]
        [InlineData(true)]
        public void CreatureBufferCanBeRoutedAroundAndTheReturnIsChecked(bool hostile)
        {
            var clearance = hostile ? 15 : 8;
            var start = new V3(-clearance - 1, 0, 0);
            var target = new V3(64, 0, 0);
            V3? Probe(V3 from, V3 at) => Scouting.TravelSeparated(Scouting.RouteDistance(from, at, V3.Zero),
                false, hostile, false) ? at : (V3?)null;
            Assert.Null(Probe(start, start + new V3(16, 0, 0)));
            var plan = new ScoutDetour(start, target, Probe, 0);
            Assert.Equal(RouteStatus.Found, Finish(plan));
            Assert.Null(Inspect(start, plan.Path, Probe));
            var end = plan.Path[plan.Path.Count - 1];
            Assert.True(V3.FlatDistance(start, target) - V3.FlatDistance(end, target) >= LocalRoute.FrontierProgress);
            var back = new ScoutDetour(end, start, Probe, 0);
            Assert.Equal(RouteStatus.Found, Finish(back));
            Assert.Null(Inspect(end, back.Path, Probe));
            Assert.True(Arrival.Reached(back.Path[back.Path.Count - 1], start, .25));
        }

        [Fact]
        public void NewObstacleInvalidatesAFoundRouteRatherThanAuthorizingReplay()
        {
            var blocked = false;
            V3? Probe(V3 from, V3 at) => blocked && at.X > 2 ? (V3?)null : at;
            var plan = new ScoutDetour(V3.Zero, new V3(5, 0, 0), Probe, 0);
            Assert.Equal(RouteStatus.Found, Finish(plan));
            blocked = true;
            Assert.NotNull(Inspect(V3.Zero, plan.Path, Probe));
        }

        [Fact]
        public void PlanningIsBoundedAndCancellationOrDeadlineNeverRestartsIt()
        {
            var probes = 0;
            V3? Flat(V3 from, V3 at) { probes++; return new V3(at.X, 0, at.Z); }
            var unreachableFloor = new ScoutDetour(V3.Zero, new V3(4, 10, 0), Flat, 0);
            Assert.Equal(RouteStatus.NoRoute, Finish(unreachableFloor));
            Assert.InRange(unreachableFloor.Expanded, 1, ScoutDetour.NodeLimit);
            Assert.InRange(unreachableFloor.Probes, 1, ScoutDetour.NodeLimit * 9);
            var before = probes;
            var timed = new ScoutDetour(V3.Zero, new V3(8, 0, 0), Flat, 0);
            Assert.Equal(RouteStatus.NoRoute, timed.Advance(ScoutDetour.TimeLimit));
            Assert.Equal(before, probes);
            var cancelled = new ScoutDetour(V3.Zero, new V3(8, 0, 0), Flat, 0);
            cancelled.Cancel();
            Assert.Equal(RouteStatus.NoRoute, cancelled.Advance(0));
            Assert.Equal(before, probes);
            Assert.Throws<ArgumentOutOfRangeException>(() => timed.Advance(0, ScoutDetour.SliceLimit + 1));
            Assert.Throws<ArgumentOutOfRangeException>(() => new ScoutDetour(new V3(double.NaN, 0, 0), V3.Zero, Flat, 0));
        }
    }
}
