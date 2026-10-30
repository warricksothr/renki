using System;
using System.Collections.Generic;
using Renki.Tasks;
using Xunit;

namespace Renki.Tests
{
    public sealed class KitingTests
    {
        static readonly V3 Area = new V3(-6, 0, 0);
        static KiteFrame Frame(double bot = 0, double enemy = 20) => new KiteFrame
        { Bot = new V3(bot, 0, 0), Enemy = new V3(enemy, 0, 0), Hp = 80, HpMax = 80, Stamina = 60 };
        static Kiting Policy() => new Kiting(V3.Zero, Area, new V3(20, 0, 0), 0);

        [Fact]
        public void DistantDiscoverySelectsLocalGroundAndKeepsTheCombatLeash()
        {
            Assert.True(Kiting.NeedsLocalArea(new V3(60, 0, 0), V3.Zero, false));
            Assert.False(Kiting.NeedsLocalArea(new V3(60, 0, 0), V3.Zero, true));
            Assert.False(Kiting.NeedsLocalArea(new V3(10, 0, 0), V3.Zero, false));
            var at = new V3(60, 0, 0);
            var p = new Kiting(at, at, at + new V3(3, 0, 0), 0);
            var f = Frame(60, 63); f.Targeting = true;
            Assert.Equal(KiteAction.Engage, p.Decide(f, 0));
            f.Enemy = new V3(101, 0, 0);
            Assert.Equal(KiteAction.Retreat, p.Decide(f, 1));
            Assert.Equal("kiting travel limit", p.Reason);
        }

        [Fact]
        public void SecondSurveyPassCoversNewGroundWithinTheSameBudgetAndRadius()
        {
            var origin = new V3(100, 50, 100);
            var points = new HashSet<V3>();
            for (var i = 0; i < Kiting.SurveyAttempts; i++)
            {
                var point = Kiting.SurveyPoint(origin, i);
                Assert.Equal(origin.Y, point.Y);
                Assert.InRange(V3.FlatDistance(origin, point), 0, 12.001);
                if (i != 17) Assert.True(points.Add(point), "only the center may be retried");
            }
            Assert.Equal(33, points.Count);
            Assert.Equal(origin, Kiting.SurveyPoint(origin, 17));
            Assert.InRange(Kiting.SurveyPoint(origin, 18).Z - origin.Z, 2.29, 2.30);
        }

        [Fact]
        public void SurveyCannotEscapeItsBoundedCandidateSchedule()
        {
            Assert.Throws<ArgumentOutOfRangeException>(() => Kiting.SurveyPoint(V3.Zero, -1));
            Assert.Throws<ArgumentOutOfRangeException>(() => Kiting.SurveyPoint(V3.Zero, Kiting.SurveyAttempts));
        }

        [Fact]
        public void PursuitProgressStartsAtTheActualScoutPosition()
        {
            var p = new Kiting(V3.Zero, V3.Zero, new V3(30, 0, 0), new V3(20, 0, 0), 0);
            var f = Frame(20, 30);
            Assert.Equal(KiteAction.Retreat, p.Decide(f, 6));
            Assert.Equal("enemy did not follow", p.Reason);
        }

        [Fact]
        public void NativeSequenceDrawsObservesPursuitAndLeadsBackBeforeEngaging()
        {
            var p = Policy(); var f = Frame();
            Assert.Equal(KiteAction.Draw, p.Decide(f, 0));
            f.Bot = new V3(12, 0, 0);
            Assert.Equal(KiteAction.Hold, p.Decide(f, 2));
            f.Enemy = new V3(19, 0, 0); f.Following = true;
            Assert.Equal(KiteAction.Lead, p.Decide(f, 3));
            Assert.False(p.Run);
            Assert.Equal(Area, p.Goal(f, KiteAction.Lead));
            f.Bot = new V3(6, 0, 0); f.Enemy = new V3(14, 0, 0);
            Assert.Equal(KiteAction.Lead, p.Decide(f, 4));
            f.Bot = new V3(0, 0, 0); f.Enemy = new V3(8, 0, 0);
            Assert.Equal(KiteAction.Lead, p.Decide(f, 5));
            f.Bot = new V3(-5, 0, 0); f.Enemy = new V3(-1, 0, 0);
            Assert.Equal(KiteAction.Engage, p.Decide(f, 6));
        }

        [Fact]
        public void PursuitRequiresActualMovementTowardTheBot()
        {
            Assert.False(Kiting.ObservedPursuit(new V3(10, 0, 0), new V3(10, 0, 0), V3.Zero));
            Assert.False(Kiting.ObservedPursuit(new V3(10, 0, 0), new V3(11, 0, 0), V3.Zero));
            Assert.False(Kiting.ObservedPursuit(new V3(10, 0, 0), new V3(10, 0, 1), V3.Zero));
            Assert.True(Kiting.ObservedPursuit(new V3(10, 0, 0), new V3(9, 0, 0), V3.Zero));
            var p = Policy(); var f = Frame(-6, -1);
            Assert.NotEqual(KiteAction.Engage, p.Decide(f, 1));
        }

        [Fact]
        public void CloseConfirmedAttackerAtCheckedStanceDoesNotNeedALure()
        {
            var f = Frame(-6, -3); f.Targeting = true;
            Assert.Equal(KiteAction.Engage, Policy().Decide(f, 0));
            f.Targeting = false;
            Assert.Equal(KiteAction.Retreat, Policy().Decide(f, 0));
        }

        [Fact]
        public void DirectDefenseRequiresBothCharactersAtCheckedGround()
        {
            var f = Frame(0, 3); f.Targeting = true;
            Assert.Equal(KiteAction.Retreat, Policy().Decide(f, 0));
            f = Frame(-4, 0.1); f.Targeting = true;
            Assert.NotEqual(KiteAction.Engage, Policy().Decide(f, 0));
            f = Frame(-6, -1); f.Targeting = true;
            Assert.NotEqual(KiteAction.Engage, Policy().Decide(f, 0));
        }

        [Theory]
        [InlineData("health")]
        [InlineData("stamina")]
        [InlineData("weapon")]
        [InlineData("unsafe")]
        [InlineData("lost")]
        [InlineData("ineligible")]
        [InlineData("travel")]
        [InlineData("time")]
        public void DirectDefenseCannotBypassSafetyAndWorkBounds(string cause)
        {
            var f = Frame(-6, -3); f.Targeting = true; var now = 0.0;
            if (cause == "health") f.Hp = 63;
            if (cause == "stamina") f.Stamina = 9;
            if (cause == "weapon") f.Weapon = false;
            if (cause == "unsafe") f.Safe = false;
            if (cause == "lost") f.Valid = false;
            if (cause == "ineligible") f.Eligible = false;
            if (cause == "travel") f.Enemy = new V3(41, 0, 0);
            if (cause == "time") now = 40;
            Assert.Equal(KiteAction.Retreat, Policy().Decide(f, now));
        }

        [Fact]
        public void LeadPacesDistanceAndPreservesStaminaInsteadOfAlwaysSprinting()
        {
            var p = Policy(); var f = Frame(0, 8); f.Following = true;
            Assert.Equal(KiteAction.Lead, p.Decide(f, 1)); Assert.False(p.Run);
            f.Stamina = 20;
            Assert.Equal(KiteAction.Hold, p.Decide(f, 2));
            f.Stamina = 60; f.Enemy = new V3(14, 0, 0);
            Assert.Equal(KiteAction.Hold, p.Decide(f, 3));
            f.Bot = new V3(4, 0, 0); f.Enemy = new V3(7, 0, 0);
            Assert.Equal(KiteAction.Lead, p.Decide(f, 4)); Assert.True(p.Run);
            f.Stamina = 29;
            Assert.Equal(KiteAction.Retreat, p.Decide(f, 5));
        }

        [Fact]
        public void NoFollowerOrLostFollowerDoesNotBecomeAnEndlessChase()
        {
            var p = Policy(); var f = Frame();
            Assert.Equal(KiteAction.Draw, p.Decide(f, 0));
            Assert.Equal(KiteAction.Retreat, p.Decide(f, 6));
            Assert.Equal("enemy did not follow", p.Reason);
            p = Policy(); f.Following = true;
            Assert.Equal(KiteAction.Hold, p.Decide(f, 1));
            f.Following = false;
            Assert.Equal(KiteAction.Retreat, p.Decide(f, 7));
            Assert.Equal("enemy stopped following", p.Reason);
        }

        [Theory]
        [InlineData("health")]
        [InlineData("stamina")]
        [InlineData("weapon")]
        [InlineData("unsafe")]
        [InlineData("lost")]
        [InlineData("ineligible")]
        [InlineData("travel")]
        [InlineData("time")]
        public void EmergencyAndWorkBoundsOverrideLeading(string cause)
        {
            var f = Frame(0, 8); f.Following = true; var now = 1.0;
            if (cause == "health") f.Hp = 63;
            if (cause == "stamina") f.Stamina = 9;
            if (cause == "weapon") f.Weapon = false;
            if (cause == "unsafe") f.Safe = false;
            if (cause == "lost") f.Valid = false;
            if (cause == "ineligible") f.Eligible = false;
            if (cause == "travel") f.Bot = new V3(41, 0, 0);
            if (cause == "time") now = 40;
            Assert.Equal(KiteAction.Retreat, Policy().Decide(f, now));
        }

        [Fact]
        public void TargetMotionWithoutProgressTowardGroundCannotExtendTheBudget()
        {
            var p = Policy(); var f = Frame(0, 8); f.Following = true;
            Assert.Equal(KiteAction.Lead, p.Decide(f, 0));
            Assert.Equal(KiteAction.Retreat, p.Decide(f, 8));
            Assert.Equal("no progress toward fighting ground", p.Reason);
        }

        [Fact]
        public void LeadDoesNotAimThroughAnEnemyBetweenBotAndArea()
        {
            var p = Policy(); var f = Frame(0, -3); f.Following = true;
            var goal = p.Goal(f, KiteAction.Lead);
            Assert.True(goal.Z != 0);
            Assert.True(goal.X > 0);
        }
    }
}
