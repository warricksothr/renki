using Renki.Tasks;
using Xunit;

namespace Renki.Tests
{
    public sealed class StartupRecoveryTests
    {
        [Fact]
        public void CriticalLoginHoldsThroughInitializationAndReleasesOnlyAfterHealing()
        {
            var guard = new StartupRecovery(true);
            Assert.Equal(RecoveryAction.Hold, guard.Decide(false, 0, 0, false, false, false, false, false, false));
            Assert.Equal(RecoveryAction.Hold, guard.Decide(true, 3.2, 32, false, false, false, false, false, false));
            Assert.Equal(RecoveryAction.Hold, guard.Decide(true, 20, 32, true, true, true, true, false, false));
            Assert.Equal(RecoveryAction.Continue, guard.Decide(true, 30.5, 32, true, true, true, true, false, false));
            // Once recovered, this opt-in startup guard does not govern ordinary combat.
            Assert.Equal(RecoveryAction.Continue, guard.Decide(true, 20, 32, true, false, true, false, true, true));
        }

        [Theory]
        [InlineData("hostile")]
        [InlineData("injury")]
        [InlineData("shelter")]
        [InlineData("ground")]
        [InlineData("food")]
        [InlineData("wet")]
        public void RecoveryDeclinesUnsafeConditionsInsteadOfWalkingOrFighting(string cause)
        {
            var guard = new StartupRecovery(true);
            guard.Decide(true, 5, 32, false, true, true, true, false, false);
            Assert.Equal(RecoveryAction.Logout, guard.Decide(true, cause == "injury" ? 4 : 5, 32, true,
                cause != "shelter", cause != "ground", cause != "food", cause == "wet", cause == "hostile"));
        }

        [Fact]
        public void DisabledGuardAndNaturalFoodMaximumDecayDoNotCauseLogout()
        {
            Assert.Equal(RecoveryAction.Continue, new StartupRecovery(false).Decide(true, 3, 32, true, false, false, false, true, true));
            var guard = new StartupRecovery(true);
            guard.Decide(true, 26, 32, true, true, true, true, false, false);
            Assert.Equal(RecoveryAction.Hold, guard.Decide(true, 24, 24, true, true, true, true, false, false));
            Assert.Equal(RecoveryAction.Continue, guard.Decide(true, 25, 25, true, true, true, false, false, false));
        }
    }
}
