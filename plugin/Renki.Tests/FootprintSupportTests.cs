using System;
using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public sealed class FootprintSupportTests
    {
        static readonly V3 Body = new V3(-33.53335952758789, 36.19356918334961, 11.111215591430664);

        [Fact]
        public void GroundedFireOpeningEdgeCanDepartOntoMeasuredFloor()
        {
            // Captured t-7: centre support had zero hits while grounded at
            // the fire opening. Adjacent floor geometry is synthetic.
            V3? Floor(V3 at) => at.X >= Body.X + .2 ? new V3(at.X, 36.09, at.Z) : (V3?)null;
            var contact = FootprintSupport.Find(Body, Body, true, "no_hits", at => (Floor(at), false), out var queries, out var full);
            Assert.NotNull(contact);
            Assert.Equal(8, queries);
            Assert.False(full);
            V3? Probe(V3 from, V3 desired)
            {
                var middle = Floor((from + desired) * .5); var to = Floor(desired);
                var start = Floor(from) ?? FootprintSupport.Find(from, Body, true, "no_hits", at => (Floor(at), false), out _, out _);
                return start.HasValue && middle.HasValue && to.HasValue &&
                    DryTravel.ContinuousSupport(from, start.Value, middle.Value, to.Value) ? to : (V3?)null;
            }
            var goal = Body + new V3(3, -.10356918334961, 0);
            var route = new LocalRoute(Body, goal, Probe, arrivalLevel: .1);
            while (route.Status == RouteStatus.Searching) route.Advance();
            Assert.Equal(RouteStatus.Found, route.Status);
            // No footprint-only walking across the opening or remote void.
            Assert.Null(Probe(Body, Body - new V3(.5, 0, 0)));
            Assert.Null(FootprintSupport.Find(Body + new V3(.5, 0, 0), Body, true, "no_hits",
                _ => throw new Exception("remote node must not probe"), out _, out _));
        }

        [Theory]
        [InlineData(false, "no_hits")]
        [InlineData(true, "slope")]
        [InlineData(true, "height_band")]
        [InlineData(true, "footing_band")]
        [InlineData(true, "saturated")]
        [InlineData(true, null)]
        public void ExistingRefusalsAndAirborneBodiesCannotUseFootprint(bool grounded, string refusal)
        {
            Assert.Null(FootprintSupport.Find(Body, Body, grounded, refusal,
                _ => throw new Exception("must retain original refusal"), out var queries, out _));
            Assert.Equal(0, queries);
        }

        [Fact]
        public void ContactRetainsHeightRadiusAndSaturationBounds()
        {
            foreach (var height in new[] { -.601, .301, double.NaN, double.PositiveInfinity })
                Assert.Null(FootprintSupport.Find(Body, Body, true, "no_hits",
                    at => (at + new V3(0, height, 0), false), out _, out _));
            Assert.Null(FootprintSupport.Find(Body, Body, true, "no_hits",
                at => (at + new V3(1, 0, 0), false), out _, out _));
            var count = 0;
            Assert.Null(FootprintSupport.Find(Body, Body, true, "no_hits", at =>
            {
                Assert.InRange(V3.FlatDistance(at, Body), .299999, .300001);
                return ((V3?)at, ++count == 2);
            }, out var queries, out var full));
            Assert.True(full);
            Assert.Equal(2, queries);
        }
    }
}
