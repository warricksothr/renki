using Renki.Tasks;
using Xunit;

namespace Renki.Tests
{
    public sealed class BedSleepTests
    {
        [Fact]
        public void ClaimWaitsForOwnershipThenRequiresActualBedAttachment()
        {
            var bed = new BedSleep();
            Assert.Equal(BedStep.Use, bed.Next(0, false, false, false));
            bed.Claimed(0);
            Assert.Equal(BedStep.Wait, bed.Next(1, false, false, false));
            Assert.Equal(BedStep.Use, bed.Next(1, true, true, false));
            Assert.Equal(BedStep.Sleeping, bed.Next(2, true, true, true));
        }

        [Fact]
        public void TakenOrUnconfirmedClaimsCannotBecomeSleepSuccess()
        {
            var bed = new BedSleep();
            Assert.Equal(BedStep.Taken, bed.Next(0, true, false, false));
            Assert.Equal(BedStep.Use, bed.Next(0, false, false, true));
            bed.Claimed(1);
            bed.Claimed(3); // Repeated observations never extend the deadline.
            Assert.Equal(BedStep.Wait, bed.Next(3, false, false, false));
            Assert.Equal(BedStep.TimedOut, bed.Next(4, false, false, false));
            Assert.Equal(BedStep.Taken, bed.Next(4, true, false, false));
        }
    }
}
