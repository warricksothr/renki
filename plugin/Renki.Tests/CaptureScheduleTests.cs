using Renki.Perception;
using Xunit;

namespace Renki.Tests
{
    public sealed class CaptureScheduleTests
    {
        [Fact]
        public void HardCeilingSurvivesConfigurationChangesAndSlowCaptures()
        {
            var s = new CaptureSchedule();
            Assert.True(s.Begin(0, 5, true));
            Assert.False(s.Begin(10, 1, true)); // Still in flight.
            s.Complete();
            Assert.True(s.Begin(10, 1, true)); s.Complete();
            Assert.False(s.Begin(10.5, 0, true)); // Malformed config cannot exceed 1 Hz.
            Assert.False(s.Begin(11, 1, false));
            Assert.True(s.Begin(11, 1, true)); s.Complete();
            Assert.False(s.Begin(15, 5, true));
            Assert.True(s.Begin(16, 5, true));
        }
    }
}
