using Renki.Protocol;
using Xunit;

namespace Renki.Tests
{
    public class WorldEventTests
    {
        [Theory]
        [InlineData(10, 30, 20, true)]
        [InlineData(105.99, -500, 20, true)]
        [InlineData(106, 30, 20, false)]
        [InlineData(10, 3000, 20, true)]
        [InlineData(10, 3000.01, 20, false)]
        public void AreaMembershipMatchesGame(double x, double y, double z, bool inside)
        {
            var e = new AreaEvent { Name = "army_greydwarf", Center = new[] { 10d, 30d, 20d }, Radius = 96 };
            Assert.Equal(inside, e.Contains(new[] { x, y, z }));
        }

        [Fact]
        public void MalformedAreasCannotBecomeSafeMembershipEvidence()
        {
            var e = new AreaEvent { Name = "raid", Radius = 96 };
            Assert.False(e.Contains(new[] { double.NaN, 30d, 20d }));
            e.Radius = double.PositiveInfinity;
            Assert.False(e.Valid());
            e.Radius = 96;
            e.Center = new[] { 1d, 2d };
            Assert.False(e.Valid());
        }
    }
}
