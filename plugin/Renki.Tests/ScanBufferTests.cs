using Renki.Perception;
using Xunit;

namespace Renki.Tests
{
    public sealed class ScanBufferTests
    {
        [Fact]
        public void AProtectedObjectBeyondTheOldPrefixIsNotLost()
        {
            var buffer = new ScanBuffer<bool>(256, 2048);
            var calls = 0;
            Assert.True(buffer.TryFill(items =>
            {
                calls++;
                for (var i = 0; i < items.Length && i < 400; i++) items[i] = i == 350;
                return System.Math.Min(items.Length, 400);
            }));
            Assert.Equal(2, calls);
            Assert.Equal(400, buffer.Count);
            Assert.True(buffer.Items[350]);
        }

        [Fact]
        public void ExactCapacityIsRetriedBecauseCompletenessIsUnknown()
        {
            var buffer = new ScanBuffer<int>(4, 16);
            Assert.True(buffer.TryFill(items => System.Math.Min(items.Length, 4)));
            Assert.Equal(8, buffer.Items.Length);
            Assert.Equal(4, buffer.Count);
        }

        [Fact]
        public void ExpandedStorageIsReusedButOnlyFreshCountIsCurrent()
        {
            var buffer = new ScanBuffer<int>(4, 16);
            Assert.True(buffer.TryFill(items => System.Math.Min(items.Length, 6)));
            var storage = buffer.Items;
            Assert.True(buffer.TryFill(items => { items[0] = 99; return 1; }));
            Assert.Same(storage, buffer.Items);
            Assert.Equal(1, buffer.Count);
            Assert.Equal(99, buffer.Items[0]);
        }

        [Fact]
        public void SaturatedMaximumNeverBecomesACompleteObservation()
        {
            var buffer = new ScanBuffer<int>(256, 2048); var calls = 0;
            Assert.False(buffer.TryFill(items => { calls++; return items.Length; }));
            Assert.Equal(4, calls);
            Assert.Equal(2048, buffer.Items.Length);
            Assert.False(buffer.TryFill(items => items.Length));
            Assert.False(buffer.TryFill(_ => -1));
        }
    }
}
