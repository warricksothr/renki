using Renki.Perception;
using Xunit;

namespace Renki.Tests
{
    public sealed class ObservationScheduleTests
    {
        [Fact]
        public void LogoutPublishesUnloadingAndMenuWithoutClearingTheLogoutLatch()
        {
            var schedule = new ObservationSchedule(); var player = new object();
            Assert.True(schedule.Due(0, SessionPhase.Resolve(false, true, true, false), "A", player, 5, out _));
            Assert.Equal("logging_out", SessionPhase.Resolve(true, true, true, false));
            Assert.True(schedule.Due(.1, SessionPhase.Resolve(true, true, true, false), "A", player, 5, out _));
            Assert.Equal("logging_out", SessionPhase.Resolve(true, false, true, false));
            Assert.Equal("menu", SessionPhase.Resolve(true, false, false, false));
            Assert.True(schedule.Due(.2, SessionPhase.Resolve(true, false, false, false), "", null, 5, out _));
            // A lingering logout latch must keep even a newly present body
            // outside in_world until this game process is restarted.
            Assert.Equal("logging_out", SessionPhase.Resolve(true, true, true, false));
        }

        [Fact]
        public void EnteringAWorldBetweenPeriodicTicksForcesScopeBeforeInventory()
        {
            var schedule = new ObservationSchedule();
            Assert.True(schedule.Due(10, "connecting", "", null, .2, out _));
            Assert.False(schedule.Due(10.1, "connecting", "", null, .2, out _));
            var player = new object();
            // The captured failure: inventory at 10.1 preceded in_world at
            // 10.2 and was cleared by the controller's world transition.
            Assert.True(schedule.Due(10.1, "in_world", "Midgard", player, .2, out var changed));
            Assert.True(changed);
            Assert.False(schedule.Due(10.2, "in_world", "Midgard", player, .2, out _));
            Assert.True(schedule.Due(10.4, "in_world", "Midgard", player, .2, out changed));
            Assert.False(changed);
        }

        [Fact]
        public void ReconnectWorldChangeAndReplacementPlayerDoNotWaitForAnOldDeadline()
        {
            var schedule = new ObservationSchedule(); var player = new object();
            Assert.True(schedule.Due(0, "in_world", "A", player, 5, out _));
            schedule.Reset();
            Assert.True(schedule.Due(.1, "in_world", "A", player, 5, out var changed));
            Assert.False(changed);
            Assert.True(schedule.Due(.2, "in_world", "B", player, 5, out changed));
            Assert.True(changed);
            Assert.True(schedule.Due(.3, "in_world", "B", new object(), 5, out changed));
            Assert.True(changed);
            Assert.True(schedule.Due(.4, "menu", "", null, 5, out changed));
            Assert.True(changed);
        }
    }
}
