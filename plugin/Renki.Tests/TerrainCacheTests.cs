using System;
using System.Linq;
using Renki.Perception;
using Xunit;

namespace Renki.Tests
{
    public sealed class TerrainCacheTests
    {
        static TerrainSample? Ground(double x, double z) => new TerrainSample(x + z, "Meadows");

        [Fact]
        public void NearestTileUsesFloorCoordinatesAndBoundsOneStepTo25Samples()
        {
            var c = new TerrainCache();
            var t = c.Step(-3, -4, 0, 0, 0, 30, Ground);
            Assert.Equal(-1, t.X); Assert.Equal(-1, t.Z);
            Assert.Equal(25L, c.Samples);
            Assert.Equal(-32, t.Heights[0]); Assert.Equal(0, t.Heights[24]);
        }

        [Fact]
        public void EqualDistanceTilesAheadAreScheduledFirst()
        {
            var east = new TerrainCache().Step(0, 0, 1, 0, 0, 30, Ground);
            var west = new TerrainCache().Step(0, 0, -1, 0, 0, 30, Ground);
            Assert.Equal(0, east.X); Assert.Equal(-1, west.X);
        }

        [Fact]
        public void StaticTerrainConfirmsFreshMeasurementsWithoutResamplingOnResend()
        {
            var c = new TerrainCache();
            // All nine radius-one tiles settle without another sample.
            for (var i = 0; i < 9; i++) Assert.NotNull(c.Step(8, 8, 0, 0, 0, 30, Ground, 1));
            var calls = c.Samples;
            Assert.Null(c.Step(8, 8, 0, 0, 1, 30, Ground, 1));
            Assert.Equal(calls, c.Samples);
            var confirmation = c.Step(8, 8, 0, 0, 31, 30, Ground, 1);
            Assert.NotNull(confirmation);
            Assert.All(confirmation.MeasuredAt, at => Assert.Equal(31, at));
            Assert.Equal(calls + 25, c.Samples);
            c.Resend();
            var resend = c.Step(8, 8, 0, 0, 31, 30, Ground, 1);
            Assert.Equal(confirmation.MeasuredAt, resend.MeasuredAt);
            Assert.Equal(calls + 25, c.Samples);
            c.Invalidate(1, 1, 2, 2);
            var changed = c.Step(8, 8, 0, 0, 32, 30, (x, z) => new TerrainSample(50, "Meadows"), 1);
            Assert.NotNull(changed); Assert.All(changed.Heights, h => Assert.Equal(50, h));
        }

        [Fact]
        public void UnknownTerrainStaysNullAndUnloadingDoesNotEraseKnowledge()
        {
            var c = new TerrainCache();
            var unknown = c.Step(8, 8, 0, 0, 0, 30, (_, _) => null, 1);
            Assert.All(unknown.Heights, h => Assert.Null(h));
            var loaded = c.Step(8, 8, 0, 0, 6, 30, Ground, 1);
            Assert.All(loaded.Heights, h => Assert.NotNull(h));
            c.Invalidate(1, 1, 2, 2);
            Assert.Null(c.Step(8, 8, 0, 0, 7, 30, (_, _) => null, 1));
            c.Resend();
            var remembered = c.Step(8, 8, 0, 0, 8, 30, (_, _) => null, 1);
            Assert.True(loaded.Heights.SequenceEqual(remembered.Heights));
            Assert.Equal(loaded.MeasuredAt, remembered.MeasuredAt);
        }

        [Fact]
        public void PartialRefreshPreservesOldCornerAgesAndOriginalSnapshots()
        {
            var c = new TerrainCache();
            var original = c.Step(8, 8, 0, 0, 10, 30, Ground, 1);
            c.Invalidate(1, 1, 2, 2);
            var partial = c.Step(8, 8, 0, 0, 12, 30,
                (x, z) => x == 0 && z == 0 ? new TerrainSample(80, "Meadows") : (TerrainSample?)null, 1);
            Assert.Equal(12, partial.MeasuredAt[0]);
            Assert.Equal(80, partial.Heights[0]);
            Assert.Equal(10, partial.MeasuredAt[1]);
            Assert.All(original.MeasuredAt, at => Assert.Equal(10, at));
            Assert.Equal(0, original.Heights[0]);
        }

        [Fact]
        public void LongTravelKeepsCacheBounded()
        {
            var c = new TerrainCache();
            for (var x = 0; x < TerrainCache.Capacity + 100; x++)
                c.Step(x * 64, 0, 1, 0, x, 30, Ground, 1);
            Assert.Equal(TerrainCache.Capacity, c.Count);
        }
    }
}
