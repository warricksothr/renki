using System;
using System.Linq;
using Renki.Tasks;
using Xunit;

namespace Renki.Tests
{
    public sealed class CookingApproachesTests
    {
        static V3[] Candidates(double bearing) => Enumerable.Range(0, CookingApproaches.Limit)
            .Select(i => CookingApproaches.Offset(i, bearing)).ToArray();

        [Fact]
        public void NarrowStandingSideCannotDisappearWithTheArrivalBearing()
        {
            // The live failure approached the same station from a bearing
            // shifted about thirty degrees from the successful trial. A thin
            // opening between structure and fire can fall between spokes.
            bool Clear(V3 at) => at.X > 1.24 && at.X < 1.26 && Math.Abs(at.Z) < .03;
            foreach (var bearing in new[] { -2.148, -1.6, 0.12, 2.6 })
                Assert.Contains(Candidates(bearing), Clear);
            // Original twelve relative spokes alone miss this opening.
            Assert.DoesNotContain(Enumerable.Range(0, 12).Select(i => CookingApproaches.Offset(i, -2.148)), Clear);
        }

        [Theory]
        [InlineData(-Math.PI)]
        [InlineData(0)]
        [InlineData(Math.PI)]
        public void EveryCandidateIsBoundedAndCurrentSideKeepsPriority(double bearing)
        {
            var points = Candidates(bearing);
            Assert.Equal(144, points.Length);
            Assert.All(points, p => { Assert.Equal(0, p.Y); Assert.InRange(p.Length, 1.24999, 4.25001); });
            Assert.InRange((points[0] - new V3(Math.Cos(bearing) * 1.25, 0, Math.Sin(bearing) * 1.25)).Length, 0, 1e-9);
            foreach (var radius in new[] { 1.25, 2.25, 3.25, 4.25 })
                Assert.Contains(points, p => (p - new V3(radius, 0, 0)).Length < 1e-9);
        }

        [Fact]
        public void ExhaustionAndInvalidBearingsCannotProduceMoreCandidates()
        {
            Assert.Throws<ArgumentOutOfRangeException>(() => CookingApproaches.Offset(-1, 0));
            Assert.Throws<ArgumentOutOfRangeException>(() => CookingApproaches.Offset(CookingApproaches.Limit, 0));
            Assert.Throws<ArgumentOutOfRangeException>(() => CookingApproaches.Offset(0, double.NaN));
            Assert.Throws<ArgumentOutOfRangeException>(() => CookingApproaches.Offset(0, double.PositiveInfinity));
        }
    }
}
