using Renki.Tasks;
using Xunit;

namespace Renki.Tests
{
    public sealed class ScoutFootingTests
    {
        [Fact]
        public void ShallowWalkingContactLossWaitsWithoutAuthorizingMovement()
        {
            var footing = new ScoutFooting();
            Assert.Equal(ScoutFootingAction.Ready, footing.Decide(0, true, true, false, double.NaN));
            Assert.Equal(ScoutFootingAction.Wait, footing.Decide(1, false, true, true, .12));
            Assert.Equal(ScoutFootingAction.Wait, footing.Decide(1.2, false, true, true, .1));
            Assert.Equal(ScoutFootingAction.Ready, footing.Decide(1.3, true, true, false, double.NaN));
            Assert.Equal(ScoutFootingAction.Wait, footing.Decide(3, false, true, true, .12));
        }

        [Fact]
        public void SustainedAirborneStateDoesNotResetItsBudgetOnAnotherTick()
        {
            var footing = new ScoutFooting();
            Assert.Equal(ScoutFootingAction.Wait, footing.Decide(1, false, true, true, .1));
            Assert.Equal(ScoutFootingAction.Wait, footing.Decide(1.49, false, true, true, .1));
            Assert.Equal(ScoutFootingAction.Abort, footing.Decide(1.5, false, true, true, .1));
        }

        [Theory]
        [InlineData(false, true, .1)] // Acquired target: no airborne engagement.
        [InlineData(true, false, .1)] // Missing or wet support.
        [InlineData(true, true, .401)]
        [InlineData(true, true, -.401)]
        [InlineData(true, true, double.NaN)]
        [InlineData(true, true, double.PositiveInfinity)]
        public void UnsupportedFallsAndCombatNeverGetTheScoutingGrace(bool scouting, bool dry, double gap)
        {
            Assert.Equal(ScoutFootingAction.Abort, new ScoutFooting().Decide(1, false, scouting, dry, gap));
        }

        [Fact]
        public void ChangingToCombatOrLosingSupportDuringSettlingRefusesImmediately()
        {
            var footing = new ScoutFooting();
            Assert.Equal(ScoutFootingAction.Wait, footing.Decide(1, false, true, true, .1));
            Assert.Equal(ScoutFootingAction.Abort, footing.Decide(1.01, false, false, true, .1));
            Assert.Equal(ScoutFootingAction.Abort, footing.Decide(1.01, false, true, false, .1));
            Assert.Equal(ScoutFootingAction.Abort, footing.Decide(.9, false, true, true, .1));
        }
    }
}
