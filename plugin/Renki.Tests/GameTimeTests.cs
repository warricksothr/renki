using Renki.Protocol;
using Xunit;
namespace Renki.Tests
{
    public sealed class GameTimeTests
    {
        [Theory]
        [InlineData(385199, 213, 0.999444444444)]
        [InlineData(385201, 214, 0.000555555555)]
        public void CalendarStaysCoherentWhileVisualLightingLags(double seconds, int day, double raw)
        {
            var clock = GameTime.FromClock(seconds, 1800, .998, true);
            Assert.Equal(day, clock.Day);
            Assert.Equal(raw, clock.CalendarFraction.Value, 9);
            Assert.Equal(.998, clock.DayFraction);
        }
        [Fact]
        public void InvalidClockIsUnknown()
        {
            Assert.Null(GameTime.FromClock(double.NaN, 1800, .5, false));
            Assert.Null(GameTime.FromClock(1, 0, .5, false));
            Assert.Null(GameTime.FromClock(-1, 1800, .5, false));
        }
    }
}
