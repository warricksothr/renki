using System;
using System.Linq;
using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public sealed class RouteMemoryTests
    {
        // Exit a raised floor via its entrance and ramp, rather than the
        // ground directly underneath. Returning can reverse the same route.
        static RouteTrail Entrance()
        {
            var trail = new RouteTrail();
            trail.Reset(new V3(0, 3, 0));
            foreach (var p in new[] { new V3(-4, 3, 0), new V3(-7, 0, 0), new V3(-10, 0, 0) }) trail.Record(p);
            return trail;
        }

        static V3? House(V3 from, V3 to)
        {
            var y = to.X >= -4 ? 3 : to.X <= -7 ? 0 : to.X + 7;
            // The entrance must be used; there is no unsupported shortcut
            // from the outside ground onto the upper floor.
            return Math.Abs(y - from.Y) <= .6 ? new V3(to.X, y, to.Z) : (V3?)null;
        }

        static RouteStatus Finish(RouteValidation validation)
        {
            for (var i = 0; i <= RouteValidation.MaxSamples && validation.Status == RouteStatus.Searching; i++)
                validation.Advance(1);
            return validation.Status;
        }

        [Fact]
        public void SuccessfulEntranceCanBeReversedAndJoinedToNearbyHomeChest()
        {
            var memory = new RouteMemory();
            memory.Remember(Entrance());
            var from = new V3(-10, 0, 0);
            var goal = new V3(1, 3.1, 1);
            var hint = memory.Find(from, goal);
            Assert.NotNull(hint);
            var replay = new RouteValidation(from, goal, hint, House);
            Assert.Equal(RouteStatus.Found, Finish(replay));
            Assert.Contains(replay.Path, p => p.X <= -4 && p.Y >= 2.9);
            Assert.True(Arrival.Reached(replay.Path.Last(), goal, .25));

            // A home marker above the floor is not a floating path point.
            goal = new V3(0, 4.5, 0);
            hint = memory.Find(from, goal);
            replay = new RouteValidation(from, goal, hint, House);
            Assert.Equal(RouteStatus.Found, Finish(replay));
            Assert.Equal(3, replay.Path.Last().Y);
        }

        [Fact]
        public void LowerFloorCannotJoinAnUpperFloorRouteAtTheSameHorizontalPoint()
        {
            var memory = new RouteMemory();
            var trail = new RouteTrail();
            trail.Reset(new V3(0, 3, 0)); trail.Record(new V3(5, 3, 0)); trail.Record(new V3(10, 3, 0));
            memory.Remember(trail);
            Assert.Null(memory.Find(V3.Zero, new V3(10, 3, 0)));
            Assert.Null(memory.Find(new V3(0, 3, 0), new V3(10, 0, 0)));
        }

        [Fact]
        public void EveryOldLegAndNewJoinIsCheckedAgainstChangedGeometry()
        {
            var memory = new RouteMemory(); memory.Remember(Entrance());
            var from = new V3(-10, 0, 0); var goal = new V3(0, 3, 0);
            var hint = memory.Find(from, goal);
            // A new obstacle in the doorway invalidates the route.
            var blocked = new RouteValidation(from, goal, hint, (a, b) => b.X >= -4.5 && b.X <= -3.5 ? null : House(a, b));
            Assert.Equal(RouteStatus.NoRoute, Finish(blocked));
            // Demolished/replaced floor must not snap down to terrain.
            var removedFloor = new RouteValidation(from, goal, hint, (a, b) => b.X > -3 ? new V3(b.X, 0, b.Z) : House(a, b));
            Assert.Equal(RouteStatus.NoRoute, Finish(removedFloor));
            // Joining from a nearby position also needs actual clearance.
            from = new V3(-10, 0, 1);
            hint = memory.Find(from, goal);
            var badJoin = new RouteValidation(from, goal, hint, (a, b) => a.Z > .1 ? null : House(a, b));
            Assert.Equal(RouteStatus.NoRoute, Finish(badJoin));
            memory.Remove(hint.ID);
            Assert.Equal(0, memory.Count);
        }

        [Fact]
        public void ReplayUsesBoundedSlicesAndRejectsLongUnsupportedLeaps()
        {
            var calls = 0;
            var hint = new RouteHint(1, new[] { new V3(20, 0, 0) });
            var replay = new RouteValidation(V3.Zero, new V3(20, 0, 0), hint, (a, b) => { calls++; Assert.True((a - b).Length <= .501); return b; });
            Assert.Equal(RouteStatus.Searching, replay.Advance(3));
            Assert.Equal(3, calls);
            Assert.Equal(RouteStatus.Found, Finish(replay));
            // An observed one-way fall is not assumed reversible as a climb.
            var climb = new RouteValidation(V3.Zero, new V3(0, 3, 0),
                new RouteHint(2, new[] { new V3(0, 3, 0) }), (a, b) => new V3(b.X, 0, b.Z));
            Assert.Equal(RouteStatus.NoRoute, Finish(climb));
            var tooLong = new RouteValidation(V3.Zero, new V3(1000, 0, 0),
                new RouteHint(3, new[] { new V3(1000, 0, 0) }), (_, b) => b);
            Assert.Equal(RouteStatus.NoRoute, Finish(tooLong));
            Assert.Equal(RouteValidation.MaxSamples, tooLong.Samples);
        }

        [Fact]
        public void ChestApproachStopsAtRequestedReachWithoutEnteringItsSolidCentre()
        {
            var goal = new V3(10, 0, 0);
            V3? Probe(V3 a, V3 b) => b.X >= 9 ? (V3?)null : b;
            var hint = new RouteHint(1, new[] { goal });
            var replay = new RouteValidation(V3.Zero, goal, hint, Probe, within: 1.5);
            Assert.Equal(RouteStatus.Found, Finish(replay));
            Assert.True(Arrival.Reached(replay.Path.Last(), goal, 1.5));
            Assert.True(replay.Path.Last().X < 9);
            var exact = new RouteValidation(V3.Zero, goal, hint, Probe);
            Assert.Equal(RouteStatus.NoRoute, Finish(exact));
        }

        [Fact]
        public void MemoryAndTrailsHaveHardBoundsAndClearWithTheirWorld()
        {
            var memory = new RouteMemory();
            for (var i = 0; i < RouteMemory.Capacity + 5; i++) memory.Remember(Entrance());
            Assert.Equal(RouteMemory.Capacity, memory.Count);
            var trail = new RouteTrail(); trail.Reset(V3.Zero);
            for (var i = 1; i <= RouteTrail.MaxPoints; i++) trail.Record(new V3(i, 0, 0));
            Assert.True(trail.Overflowed);
            memory.Remember(trail);
            Assert.Equal(RouteMemory.Capacity, memory.Count);
            memory.Clear(); Assert.Equal(0, memory.Count);
            Assert.Null(memory.Find(new V3(-10, 0, 0), new V3(0, 3, 0)));
            trail.Reset(V3.Zero); trail.Record(new V3(1, 0, 0));
            memory.Remember(trail); Assert.Equal(0, memory.Count);
        }

        [Fact]
        public void BacktrackingIsPrunedWithoutMergingDifferentFloors()
        {
            var trail = new RouteTrail(); trail.Reset(V3.Zero);
            trail.Record(new V3(1, 0, 0)); trail.Record(new V3(2, 0, 0)); trail.Record(new V3(1.1, 0, 0));
            Assert.DoesNotContain(trail.Points, p => p.X == 2);
            trail.Record(new V3(1.1, 3, 0));
            Assert.Contains(trail.Points, p => p.Y == 0);
            Assert.Contains(trail.Points, p => p.Y == 3);
        }
    }
}
