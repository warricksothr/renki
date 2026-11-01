using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public sealed class SupportSelectionTests
    {
        [Fact]
        public void WalkingSlopeRefusalIsDistinctFromMissingSupport()
        {
            var ray = new SupportSelection(31.37, .6, false, true);
            Assert.Equal("no_hits", ray.Refusal);
            ray.Consider(new V3(-50.79, 31.7, 10.12), .62);
            Assert.Null(ray.Point);
            Assert.Equal("slope", ray.Refusal);
            Assert.Equal(1, ray.Hits);
            Assert.Equal(0, ray.NormalHits);
            // Ordinary stair treads and another nearby floor do not make
            // that steep face walkable; the accepted tread is selected.
            var tread = new V3(-50.79, 31.6, 10.12);
            ray.Consider(tread, 1);
            Assert.Equal(tread, ray.Point);
            Assert.Null(ray.Refusal);
        }

        [Fact]
        public void HighestRayHitDoesNotSilentlyFallBackToAnotherFloor()
        {
            var ray = new SupportSelection(31, .6, false, true);
            ray.Consider(new V3(0, 31.2, 0), 1);
            ray.Consider(new V3(0, 31.64, 0), 1);
            Assert.Null(ray.Point);
            Assert.Equal("height_band", ray.Refusal);
            Assert.Equal(31.64, ray.Selected.Value.Y);
            // Reversing physics hit order keeps the same floor decision.
            var reversed = new SupportSelection(31, .6, false, true);
            reversed.Consider(new V3(0, 31.64, 0), 1);
            reversed.Consider(new V3(0, 31.2, 0), 1);
            Assert.Equal(ray.Selected, reversed.Selected);
        }

        [Fact]
        public void FootingRemainsSeparateFromFutureWalkingSupport()
        {
            var ray = new SupportSelection(31, .6, true, false);
            ray.Consider(new V3(0, 31.5, 0), 1);
            Assert.Equal("footing_band", ray.Refusal);
            ray.Consider(new V3(0, 30.9, 0), .62);
            Assert.Equal(30.9, ray.Point.Value.Y);
            ray.Consider(new V3(0, 31.2, 0), 1);
            Assert.Equal(30.9, ray.Point.Value.Y);
        }

        [Fact]
        public void MemoFailureEvidenceBelongsToTheExactQueryAndSlice()
        {
            var revision = "old";
            var memo = new SupportMemo((at, y) => ((V3?)null, (object)(revision + ":" + y)));
            memo.Sample(V3.Zero, 31, 0, 1);
            Assert.Equal("old:31", memo.LastEvidence);
            memo.Sample(V3.Forward, 32, 0, 1);
            Assert.Equal("old:32", memo.LastEvidence);
            revision = "new";
            memo.Sample(V3.Zero, 31, .1, 1);
            Assert.Equal("old:31", memo.LastEvidence);
            memo.Sample(V3.Zero, 31, .1, 2);
            Assert.Equal("new:31", memo.LastEvidence);
            var successful = new SupportMemo((at, y) => ((V3?)at, (object)null));
            Assert.Equal(V3.Zero, successful.Sample(V3.Zero, 0, 0, 1));
            Assert.Null(successful.LastEvidence);
        }
    }
}
