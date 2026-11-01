using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public sealed class SupportSelectionTests
    {
        [Fact]
        public void CapturedRotatedFloorNormalAllowsEntranceSupportWithoutAdmittingTheStairFace()
        {
            // Native audit Cb7pe0: the floor was refused as slope solely
            // because its float normal was slightly above one.
            var floor = new V3(-42.14382553100586, 36.090484619140625, 9.61785888671875);
            var ray = new SupportSelection(35.96063232421875, .6, false, true);
            ray.Consider(floor, 1.0000001192092896);
            Assert.Equal(floor, ray.Point);
            Assert.Null(ray.Refusal);
            Assert.Equal(1, ray.NormalHits);
            var hints = new EntranceHints();
            Assert.Equal(floor, hints.Select(new V3(-36.92, 34.11, 7.52),
                new V3(-35.9, 37.66, 6.53), new[] { ray.Point.Value }));

            var stair = new SupportSelection(35.14154815673828, .6, false, true);
            stair.Consider(new V3(-42.497379302978516, 35.032344818115234, 15.26430606842041), .44619783759117126);
            Assert.Null(stair.Point);
            Assert.Equal("slope", stair.Refusal);
        }

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
