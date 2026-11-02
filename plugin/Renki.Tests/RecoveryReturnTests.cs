using Renki.Tasks;
using Xunit;

namespace Renki.Tests
{
    public sealed class RecoveryReturnTests
    {
        [Fact]
        public void NearbyDurationUsesTwoRequiredMealsEvenWithAnExpiringThird()
        {
            var order = new RecoveryReturnOrder { Profile = "nearby" };
            // Captured native admission: blueberries109s, yellow mushroom406s,
            // mushroom893s. Two lasting meals satisfy the120s return reserve.
            Assert.Equal(406, order.FoodReserveSeconds(109, 406, 893));
            Assert.Equal(406, order.FoodReserveSeconds(893, 109, 406));
            Assert.Equal(120, order.FoodReserveSeconds(120, 600));
            Assert.Equal(0, order.FoodReserveSeconds(600));
            Assert.Equal(0, order.FoodReserveSeconds(600, double.NaN, 600));
            Assert.Equal(0, order.FoodReserveSeconds(600, double.PositiveInfinity, 600));
            Assert.Equal(0, order.FoodReserveSeconds(-1, 600, 600));
            var start = new V3(20, 30, 0);
            Assert.Equal(RecoveryReturnAction.Move, Step(Guard("nearby"), hp: 50.1, max: 53.7, pos: start,
                foodSeconds: order.FoodReserveSeconds(109, 406, 893)));
            Assert.Equal(RecoveryReturnAction.Logout, Step(Guard("nearby"), hp: 50.1, max: 53.7, pos: start,
                foodSeconds: order.FoodReserveSeconds(109, 119, 893)));
            order.Profile = "work_trip";
            Assert.Equal(109, order.FoodReserveSeconds(109, 406, 893));
            Assert.Equal(0, order.FoodReserveSeconds(406, 893));
        }

        static RecoveryReturn Guard(string profile = null) => new RecoveryReturn(new RecoveryReturnOrder
        { World = "one", Pos = new[] { 0d, 30, 0 }, Expires = 1200, Profile = profile });

        static RecoveryReturnAction Step(RecoveryReturn guard, double now = 0, double hp = 53,
            double max = 68, double stamina = 30, int foods = 3, double threat = 8,
            bool session = true, bool water = false, string world = "one", long utc = 1000,
            V3? pos = null, bool sheltered = false, bool settled = true, bool grounded = true, double foodSeconds = 600) =>
            guard.Decide(world, utc, now, pos ?? new V3(40, 30, 0), hp, max, stamina,
                foods, grounded, water, sheltered, threat, session, settled, foodSeconds);

        [Fact]
        public void ExplicitNearbyReturnHasItsOwnTighterScopeAndReserves()
        {
            var start = new V3(20, 30, 0);
            Assert.Equal(RecoveryReturnAction.Logout, Step(Guard(), hp: 44, max: 53.5, pos: start));
            var g = Guard("nearby");
            Assert.Equal(RecoveryReturnAction.Move, Step(g, hp: 44, max: 53.5, pos: start, foodSeconds: 120));
            Assert.Equal(RecoveryReturnAction.Arrived, Step(g, now: 20, hp: 44, max: 53.5, pos: new V3(0, 28.5, 0), sheltered: true, threat: 20));
            Assert.Equal(RecoveryReturnAction.Logout, Step(Guard("nearby"), hp: 39.9, max: 45, pos: start));
            Assert.Equal(RecoveryReturnAction.Logout, Step(Guard("nearby"), hp: 44, max: 56, pos: start));
            Assert.Equal(RecoveryReturnAction.Logout, Step(Guard("nearby"), hp: 44, max: 53.5, pos: new V3(30.1, 30, 0)));
            Assert.Equal(RecoveryReturnAction.Logout, Step(Guard("nearby"), hp: 44, max: 53.5, pos: start, threat: 7.9));
            Assert.Equal(RecoveryReturnAction.Logout, Step(Guard("nearby"), hp: 44, max: 53.5, pos: start, foods: 1));
            Assert.Equal(RecoveryReturnAction.Logout, Step(Guard("nearby"), hp: 44, max: 53.5, pos: start, foodSeconds: 119));
            Assert.Equal(RecoveryReturnAction.Logout, Step(Guard("nearby"), hp: 44, max: 53.5, pos: start, foodSeconds: double.NaN));
        }

        [Theory]
        [InlineData("injury")]
        [InlineData("time")]
        [InlineData("travel")]
        [InlineData("threat")]
        [InlineData("water")]
        [InlineData("session")]
        public void NearbyRecoveryNeverRenewsItsReturnBudget(string cause)
        {
            var g = Guard("nearby"); var start = new V3(20, 30, 0);
            Assert.Equal(RecoveryReturnAction.Move, Step(g, hp: 44, max: 53.5, pos: start));
            if (cause == "travel")
                Assert.Equal(RecoveryReturnAction.Move, Step(g, now: 1, hp: 44, max: 53.5, pos: new V3(-20, 30, 0)));
            Assert.Equal(RecoveryReturnAction.Logout, Step(g, now: cause == "time" ? 90 : 2,
                hp: cause == "injury" ? 43 : 44, max: 53.5, pos: start, threat: cause == "threat" ? 7 : 8,
                water: cause == "water", session: cause != "session"));
            Assert.Equal(RecoveryReturnAction.Logout, Step(g, now: 3, hp: 44, max: 53.5, pos: start));
        }

        [Fact]
        public void ExplicitWorkTripCanReturnFromOutsideTheUnchangedLocalRadius()
        {
            var start = new V3(96, 30, 0);
            Assert.Equal(RecoveryReturnAction.Logout, Step(Guard(), hp: 68, stamina: 35, pos: start));
            var g = Guard("work_trip");
            Assert.Equal(RecoveryReturnAction.Move, Step(g, hp: 68, stamina: 35, pos: start, foodSeconds: 270));
            Assert.Equal(RecoveryReturnAction.Move, Step(g, now: 100, hp: 67, max: 67,
                foods: 2, stamina: 20, pos: new V3(0, 26, 0)));
            Assert.Equal(RecoveryReturnAction.Arrived, Step(g, now: 200, hp: 67, max: 67,
                pos: new V3(0, 28.5, 0), sheltered: true, threat: 20));
        }

        [Theory]
        [InlineData("health")]
        [InlineData("food_count")]
        [InlineData("food_duration")]
        [InlineData("unknown_duration")]
        [InlineData("invalid_duration")]
        [InlineData("infinite_duration")]
        [InlineData("stamina")]
        public void WorkTripRequiresStrongerAdmissionReserves(string cause)
        {
            Assert.Equal(RecoveryReturnAction.Logout, Step(Guard("work_trip"),
                hp: cause == "health" ? 60 : 68, foods: cause == "food_count" ? 2 : 3,
                stamina: cause == "stamina" ? 34 : 35, foodSeconds: cause == "food_duration" ? 269 :
                    cause == "unknown_duration" ? 0 : cause == "invalid_duration" ? double.NaN :
                    cause == "infinite_duration" ? double.PositiveInfinity : 300));
        }

        [Theory]
        [InlineData("time")]
        [InlineData("radius")]
        [InlineData("food")]
        [InlineData("injury")]
        [InlineData("authority")]
        [InlineData("water")]
        [InlineData("threat")]
        public void WorkTripPreservesLiveGuardsAndHasAnIndependentTimeAndRadiusBound(string cause)
        {
            var g = Guard("work_trip");
            Assert.Equal(RecoveryReturnAction.Move, Step(g, hp: 68, stamina: 35, pos: new V3(200, 30, 0)));
            Assert.Equal(RecoveryReturnAction.Logout, Step(g, now: cause == "time" ? 240 : 1,
                hp: cause == "injury" ? 67 : 68, foods: cause == "food" ? 1 : 3,
                session: cause != "authority", water: cause == "water", threat: cause == "threat" ? 5 : 8,
                pos: new V3(cause == "radius" ? 201 : 200, 30, 0)));
            Assert.Equal(RecoveryReturnAction.Logout, Step(g, now: 2, hp: 68));
        }

        [Fact]
        public void WorkTripStopsAtTheTravelBoundEvenInsideItsHomeRadius()
        {
            var g = Guard("work_trip");
            Assert.Equal(RecoveryReturnAction.Move, Step(g, hp: 68, stamina: 35, pos: new V3(100, 30, 0)));
            Assert.Equal(RecoveryReturnAction.Move, Step(g, now: 10, hp: 68, pos: new V3(-100, 30, 0)));
            Assert.Equal(RecoveryReturnAction.Move, Step(g, now: 20, hp: 68, pos: new V3(100, 30, 0)));
            Assert.Equal(RecoveryReturnAction.Logout, Step(g, now: 21, hp: 68, pos: new V3(99, 30, 0)));
        }

        [Theory]
        [InlineData("work_trip_unbounded")]
        [InlineData("Local")]
        public void UnknownReturnProfilesCannotGrantMovement(string profile)
        {
            Assert.Equal(RecoveryReturnAction.Logout, Step(Guard(profile), hp: 68, stamina: 35));
        }

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

        [Theory]
        [InlineData(true, false)]
        [InlineData(false, true)]
        [InlineData(false, false)]
        public void UncoveredPorchOrLastStairCanContinueToActualHomeShelter(bool grounded, bool sheltered)
        {
            var g = Guard();
            Assert.Equal(RecoveryReturnAction.Move, Step(g, threat: 20));
            Assert.Equal(RecoveryReturnAction.Move, Step(g, now: 1, pos: new V3(0, 28.5, 2.9),
                grounded: grounded, sheltered: sheltered, threat: 20));
            Assert.Equal(RecoveryReturnAction.Arrived, Step(g, now: 2, pos: new V3(0, 28.5, 1.4),
                grounded: true, sheltered: true, threat: 20));
        }

        [Theory]
        [InlineData(false, true, 20)]
        [InlineData(true, false, 20)]
        [InlineData(true, true, 8)]
        public void ActualHomeEndpointStillRequiresGroundShelterAndThreatClearance(bool grounded, bool sheltered, double threat)
        {
            var g = Guard();
            Assert.Equal(RecoveryReturnAction.Move, Step(g, threat: 20));
            Assert.Equal(RecoveryReturnAction.Logout, Step(g, now: 1, pos: new V3(0, 28.5, 1.4),
                grounded: grounded, sheltered: sheltered, threat: threat));
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
