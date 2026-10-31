using Renki.Tasks;
using Xunit;

namespace Renki.Tests
{
    public sealed class RecoveryReturnTests
    {
        static RecoveryReturn Guard() => new RecoveryReturn(new RecoveryReturnOrder
        { World = "one", Pos = new[] { 0d, 30, 0 }, Expires = 1200 });

        static RecoveryReturnAction Step(RecoveryReturn guard, double now = 0, double hp = 53,
            double max = 68, double stamina = 30, int foods = 3, double threat = 8,
            bool session = true, bool water = false, string world = "one", long utc = 1000,
            V3? pos = null, bool sheltered = false, bool settled = true, bool grounded = true) =>
            guard.Decide(world, utc, now, pos ?? new V3(40, 30, 0), hp, max, stamina,
                foods, grounded, water, sheltered, threat, session, settled);

        [Fact]
        public void ExpiringReturnKeepsBodyHeldUntilExactHomeShelterThenHandsToPassiveHealing()
        {
            var g = Guard();
            Assert.Equal(RecoveryReturnAction.Hold, Step(g, session: false, settled: false));
            Assert.Equal(RecoveryReturnAction.Move, Step(g, now: 2));
            Assert.Equal(RecoveryReturnAction.Move, Step(g, now: 10, pos: new V3(0, 26, 0)));
            Assert.Equal(RecoveryReturnAction.Arrived, Step(g, now: 20, pos: new V3(0, 28.5, 0), sheltered: true, threat: 20));
        }

        [Theory]
        [InlineData("critical")]
        [InlineData("fraction")]
        [InlineData("food")]
        [InlineData("stamina")]
        [InlineData("water")]
        [InlineData("threat")]
        [InlineData("ground")]
        [InlineData("expired")]
        [InlineData("world")]
        [InlineData("distance")]
        [InlineData("nonfinite")]
        public void UnsafeAdmissionNeverReleasesTheBody(string cause)
        {
            Assert.Equal(RecoveryReturnAction.Logout, Step(Guard(), hp: cause == "critical" ? 4 : 53,
                max: cause == "fraction" ? 100 : 68, foods: cause == "food" ? 1 : 3,
                stamina: cause == "stamina" ? 24 : 30, water: cause == "water",
                threat: cause == "threat" ? 5 : 8, grounded: cause != "ground",
                utc: cause == "expired" ? 1200 : 1000, world: cause == "world" ? "other" : "one",
                pos: cause == "distance" ? new V3(81, 30, 0) : cause == "nonfinite" ? new V3(double.NaN, 30, 0) : (V3?)null));
        }

        [Theory]
        [InlineData("injury")]
        [InlineData("session")]
        [InlineData("time")]
        [InlineData("stamina")]
        [InlineData("cancel")]
        public void EscapeStopsOnInjuryLostAuthorityAndBoundsWithoutRearming(string cause)
        {
            var g = Guard();
            Assert.Equal(RecoveryReturnAction.Move, Step(g));
            if (cause == "cancel") g.Cancel();
            Assert.Equal(RecoveryReturnAction.Logout, Step(g, now: cause == "time" ? 120 : 2,
                hp: cause == "injury" ? 52 : 53, session: cause != "session", stamina: cause == "stamina" ? 7 : 30));
            Assert.Equal(RecoveryReturnAction.Logout, Step(g, now: 3));
        }

        [Fact]
        public void FoodMaximumChangesDoNotInventInjuryButNewThreatAtHomePreventsArrival()
        {
            var g = Guard();
            Assert.Equal(RecoveryReturnAction.Move, Step(g));
            Assert.Equal(RecoveryReturnAction.Move, Step(g, now: 1, max: 70));
            Assert.Equal(RecoveryReturnAction.Move, Step(g, now: 2, hp: 52.9, max: 52.9));
            Assert.Equal(RecoveryReturnAction.Logout, Step(g, now: 3, hp: 52.9, max: 52.9,
                pos: new V3(0, 30, 0), sheltered: true, threat: 8));
        }

        [Fact]
        public void InvalidArmingAndInitializationTimeoutFailClosed()
        {
            Assert.False(new RecoveryReturnOrder { World = "one", Pos = new[] { 0d }, Expires = 1100 }.Valid("one", 1000));
            Assert.False(new RecoveryReturnOrder { World = "one", Pos = new[] { 0d, 30, 0 }, Expires = 1700 }.Valid("one", 1000));
            var g = Guard();
            Assert.Equal(RecoveryReturnAction.Hold, Step(g, session: false));
            Assert.Equal(RecoveryReturnAction.Logout, Step(g, now: 10, session: false));
        }
    }
}
