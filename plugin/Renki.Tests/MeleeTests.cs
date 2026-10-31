using System.Collections.Generic;
using Newtonsoft.Json.Linq;
using Renki.Protocol;
using Renki.Tasks;
using Xunit;

namespace Renki.Tests
{
    public sealed class MeleeTests
    {
        static MeleeFrame Healthy() => new MeleeFrame
        {
            Valid = true,
            Eligible = true,
            Weapon = true,
            Hp = 100,
            HpMax = 100,
            Stamina = 60,
            AttackCost = 6,
            DodgeCost = 10,
            BlockCost = 25,
            TargetHp = 20,
            CanBlock = true,
            InRange = true,
        };

        [Theory]
        [InlineData(30, 45, 35, 2, true, true, true)]
        [InlineData(29.9, 40, 35, 2, true, true, false)]
        [InlineData(30, 50, 35, 2, true, true, false)]
        [InlineData(30, 45, 34, 2, true, true, false)]
        [InlineData(30, 45, 35, 1, true, true, false)]
        [InlineData(30, 45, 35, 2, false, true, false)]
        [InlineData(30, 45, 35, 2, true, false, false)]
        public void DefensiveEntryNeedsReservesFoodArmourAndNativeConfirmation(double hp, double max, double stamina, double armour, bool fed, bool confirmed, bool allowed)
        {
            Assert.Equal(allowed, MeleeEncounter.DefensiveReady(hp, max, stamina, armour, fed, confirmed));
        }

        [Theory]
        [InlineData("health")]
        [InlineData("fraction")]
        [InlineData("stamina")]
        [InlineData("distance")]
        [InlineData("leash")]
        [InlineData("time")]
        [InlineData("bystander")]
        public void DefenseCannotChaseOrLoseItsSurvivalReserves(string cause)
        {
            var f = Healthy(); f.Hp = 30; f.HpMax = 45;
            var policy = new MeleeEncounter(0, 20, 2, defensive: true);
            Assert.Equal(MeleeAction.Strike, policy.Decide(f, 1));
            Assert.Equal(MeleeAction.Retreat, new MeleeEncounter(0, 20).Decide(f, 1));
            double now = 2;
            if (cause == "health") f.Hp = 24.9;
            if (cause == "fraction") f.HpMax = 51;
            if (cause == "stamina") f.Stamina = 9;
            if (cause == "distance") f.TargetDistance = 6.1;
            if (cause == "leash") f.DistanceFromStart = 6.1;
            if (cause == "time") now = 12;
            if (cause == "bystander") f.Unsafe = true;
            Assert.Equal(MeleeAction.Retreat, policy.Decide(f, now));
        }

        [Fact]
        public void DefensiveProtocolNeverStartsSearchOrKiting()
        {
            var args = Assert.IsType<AttackArgs>(TaskArgs.Parse(new TaskRequest { Task = "attack", Args = JObject.Parse("{\"entity\":\"1:2\",\"defensive\":true}") }));
            Assert.True(args.Defensive);
            Assert.Throws<BadArgsException>(() => TaskArgs.Parse(new TaskRequest { Task = "attack", Args = JObject.Parse("{\"entity\":\"1:2\",\"defensive\":true,\"kite\":true}") }));
            Assert.Throws<BadArgsException>(() => TaskArgs.Parse(new TaskRequest { Task = "attack", Args = JObject.Parse("{\"defensive\":true}") }));
        }

        [Fact]
        public void NormalMeleeTracksApproachAttackWindowAndRecovery()
        {
            var policy = new MeleeEncounter(0, 20); var f = Healthy();
            f.InRange = false;
            Assert.Equal(MeleeAction.Approach, policy.Decide(f, 1));
            f.InRange = true;
            Assert.Equal(MeleeAction.Strike, policy.Decide(f, 2));
            f.Busy = true;
            Assert.Equal(MeleeAction.Wait, policy.Decide(f, 3));
            f.Busy = false; f.Incoming = true;
            Assert.Equal(MeleeAction.Block, policy.Decide(f, 4));
            f.CanDodge = true;
            Assert.Equal(MeleeAction.Dodge, policy.Decide(f, 5));
            f.Incoming = false; f.Stamina = 30;
            Assert.Equal(MeleeAction.Wait, policy.Decide(f, 6));
        }

        [Fact]
        public void ExhaustionMustNotSpendTheEscapeReserve()
        {
            var policy = new MeleeEncounter(0, 20); var f = Healthy();
            f.Incoming = true; f.CanDodge = true; f.Stamina = 19;
            Assert.Equal(MeleeAction.Retreat, policy.Decide(f, 1));
            f.Stamina = 20;
            Assert.Equal(MeleeAction.Dodge, policy.Decide(f, 2));
            f.CanDodge = false;
            Assert.Equal(MeleeAction.Retreat, policy.Decide(f, 3));
            f.Stamina = 35;
            Assert.Equal(MeleeAction.Block, policy.Decide(f, 4));
            f.Incoming = false; f.Stamina = 9;
            Assert.Equal(MeleeAction.Retreat, policy.Decide(f, 5));
        }

        [Fact]
        public void ExplicitTargetCanBeClosedOnBeforeTheEncounterLeashStarts()
        {
            // The live attempt escaped to 25 m before the command arrived.
            var policy = new MeleeEncounter(0, 20, 25); var f = Healthy();
            f.TargetDistance = f.DistanceFromStart = 25; f.InRange = false;
            Assert.Equal(MeleeAction.Closing, policy.Decide(f, 1));
            Assert.False(policy.Engaged);
            f.TargetDistance = f.DistanceFromStart = 20;
            Assert.Equal(MeleeAction.Closing, policy.Decide(f, 4));
            // Passing within 12 m is still approach, not a fight yet.
            f.TargetDistance = 12; f.DistanceFromStart = 18;
            Assert.Equal(MeleeAction.Closing, policy.Decide(f, 8));
            Assert.False(policy.Engaged);
            // The host reanchors immediately before supplying this frame.
            f.TargetDistance = 3; f.DistanceFromStart = 0;
            Assert.Equal(MeleeAction.Approach, policy.Decide(f, 9));
            Assert.True(policy.Engaged);
            f.InRange = true; f.TargetDistance = 1; f.DistanceFromStart = 11;
            Assert.Equal(MeleeAction.Strike, policy.Decide(f, 13));
            Assert.Equal(MeleeAction.Retreat, policy.Decide(f, 21));
            Assert.Equal("no observed progress against target", policy.Reason);
        }

        [Theory]
        [InlineData("time")]
        [InlineData("progress")]
        [InlineData("target range")]
        [InlineData("travel area")]
        [InlineData("health")]
        [InlineData("stamina")]
        [InlineData("bystander")]
        [InlineData("weapon")]
        public void ClosingIsBoundedAndKeepsEmergencyRetreat(string cause)
        {
            var policy = new MeleeEncounter(0, 20, 25); var f = Healthy();
            f.TargetDistance = f.DistanceFromStart = 25;
            var now = 1.0;
            if (cause == "time") { now = 12; f.TargetDistance = 12; f.DistanceFromStart = 12; }
            if (cause == "progress") now = 4;
            if (cause == "target range") f.TargetDistance = 30.01;
            if (cause == "travel area") f.DistanceFromStart = 30.01;
            if (cause == "health") f.Hp = 79;
            if (cause == "stamina") f.Stamina = 9;
            if (cause == "bystander") f.Unsafe = true;
            if (cause == "weapon") f.Weapon = false;
            Assert.Equal(MeleeAction.Retreat, policy.Decide(f, now));
            Assert.False(policy.Engaged);
        }

        [Fact]
        public void ClosingDoesNotRenewItsTimeBudgetWithDamageOrSmallOscillations()
        {
            var policy = new MeleeEncounter(0, 20, 25); var f = Healthy();
            f.TargetDistance = f.DistanceFromStart = 24.6; f.TargetHp = 15;
            Assert.Equal(MeleeAction.Closing, policy.Decide(f, 3));
            Assert.Equal(MeleeAction.Retreat, policy.Decide(f, 4));
            Assert.Equal("no progress closing on target", policy.Reason);
        }

        [Theory]
        [InlineData("health")]
        [InlineData("weapon")]
        [InlineData("unsafe")]
        [InlineData("leash")]
        public void WorseningConditionsSelectRetreatEvenDuringASwing(string cause)
        {
            var f = Healthy(); f.Busy = true;
            if (cause == "health") f.Hp = 79;
            if (cause == "weapon") f.Weapon = false;
            if (cause == "unsafe") f.Unsafe = true;
            if (cause == "leash") f.DistanceFromStart = 12.01;
            Assert.Equal(MeleeAction.Retreat, new MeleeEncounter(0, 20).Decide(f, 1));
        }

        [Fact]
        public void OnlyObservedDeathSucceedsAndDamageProgressCannotExtendWorkForever()
        {
            var f = Healthy(); var policy = new MeleeEncounter(0, 20);
            f.Valid = false; f.Dead = true;
            Assert.Equal(MeleeAction.Lost, policy.Decide(f, 1));
            f.Valid = true;
            Assert.Equal(MeleeAction.Won, policy.Decide(f, 2));
            f.Dead = false;
            Assert.Equal(MeleeAction.Retreat, policy.Decide(f, 12));
            f.TargetHp = 19;
            Assert.Equal(MeleeAction.Strike, policy.Decide(f, 13));
            f.TargetHp = 18;
            Assert.Equal(MeleeAction.Retreat, policy.Decide(f, 45));
        }

        [Theory]
        [InlineData("Greyling", 1, false, false, true, true)]
        [InlineData("Boar", 1, false, false, true, true)]
        [InlineData("Neck", 1, false, false, true, true)]
        [InlineData("Neck", 2, false, false, true, false)]
        [InlineData("Neck", 1, false, true, true, false)]
        [InlineData("Neck", 1, true, false, true, false)]
        [InlineData("Neck", 1, false, false, false, false)]
        [InlineData("Boar", 2, false, false, true, false)]
        [InlineData("Boar", 1, false, true, true, false)]
        [InlineData("Boar", 1, true, false, true, false)]
        [InlineData("Boar", 1, false, false, false, false)]
        [InlineData("Greyling", 2, false, false, true, false)]
        [InlineData("Greyling", 1, false, true, true, false)]
        [InlineData("Greyling", 1, true, false, true, false)]
        [InlineData("Greyling", 1, false, false, false, false)]
        [InlineData("Deer", 1, false, false, true, false)]
        [InlineData("Troll", 1, false, false, true, false)]
        public void HostilityAloneCannotChooseAnArbitraryCreature(string prefab, int level, bool player, bool tamed, bool hostile, bool allowed)
        {
            Assert.Equal(allowed, MeleeEncounter.Eligible(prefab, level, player, tamed, hostile));
        }

        [Fact]
        public void OtherOrdinaryCreaturesRequireAnExplicitTargetWhileSearchStillChoosesGreylings()
        {
            Assert.True(MeleeEncounter.SearchEligible("Greyling", 1, false, false, true, false));
            Assert.False(MeleeEncounter.SearchEligible("Boar", 1, false, false, true, false));
            Assert.True(MeleeEncounter.SearchEligible("Boar", 1, false, false, true, true));
            Assert.False(MeleeEncounter.SearchEligible("Neck", 1, false, false, true, false));
            Assert.True(MeleeEncounter.SearchEligible("Neck", 1, false, false, true, true));
            Assert.False(MeleeEncounter.SearchEligible("Neck", 2, false, false, true, true));
            Assert.False(MeleeEncounter.SearchEligible("Boar", 1, false, true, true, true));
            Assert.False(MeleeEncounter.SearchEligible("Troll", 1, false, false, true, true));
        }

        [Fact]
        public void MovingEnemyDoesNotSpendTheBotsPursuitLeash()
        {
            var policy = new MeleeEncounter(0, 20); var f = Healthy();
            f.InRange = false; f.TargetDistance = 14; f.DistanceFromStart = 3;
            Assert.Equal(MeleeAction.Approach, policy.Decide(f, 1));
            f.TargetDistance = 31;
            Assert.Equal(MeleeAction.Retreat, policy.Decide(f, 2));
            f.TargetDistance = 3; f.DistanceFromStart = 13;
            Assert.Equal(MeleeAction.Retreat, policy.Decide(f, 3));
        }

        [Fact]
        public void ClosingCanRunOnlyWithAttackDefenseEscapeAndExtraReserve()
        {
            var f = Healthy(); f.TargetDistance = 10; f.Stamina = 61;
            Assert.True(MeleeEncounter.RunApproach(f, 1.4));
            f.Stamina = 60;
            Assert.False(MeleeEncounter.RunApproach(f, 1.4));
            f.Stamina = 61; f.Incoming = true;
            Assert.False(MeleeEncounter.RunApproach(f, 1.4));
            f.Incoming = false; f.TargetDistance = 8;
            Assert.False(MeleeEncounter.RunApproach(f, 1.4));
        }

        [Theory]
        [InlineData("stop", false)]
        [InlineData("replacement", false)]
        [InlineData("timeout", false)]
        [InlineData("deadman", false)]
        [InlineData("reconnect", false)]
        [InlineData("death", false)]
        [InlineData("stop", true)]
        [InlineData("replacement", true)]
        [InlineData("timeout", true)]
        [InlineData("deadman", true)]
        [InlineData("reconnect", true)]
        [InlineData("death", true)]
        public void AttackUsesTheBodyCancellationLifecycle(string end, bool kite)
        {
            var host = new FakeHost(); var reports = new List<TaskStatus>();
            var runner = new TaskRunner(new SafetyConfig { AllowCombat = true }, TaskNames.Supported, host, reports.Add, () => 0);
            runner.Welcome(TaskNames.Supported);
            runner.Handle(new TaskRequest { Id = "a", Task = "attack", Args = kite ? JObject.Parse("{\"kite\":true}") : JObject.Parse("{\"entity\":\"1:2\"}"), TimeoutS = 1 });
            var body = Assert.Single(host.Bodies);
            if (end == "stop") runner.Handle(new TaskRequest { Id = "s", Task = "stop" });
            if (end == "replacement") runner.Handle(new TaskRequest { Id = "b", Task = "attack", Args = JObject.Parse("{\"entity\":\"1:3\"}") });
            if (end == "timeout") runner.FixedTick(2);
            if (end == "deadman") runner.EndSession("quiet");
            if (end == "reconnect") runner.Welcome(TaskNames.Supported);
            if (end == "death") runner.Died();
            Assert.True(body.Cancelled);
            Assert.True(host.Idles > 0);
            if (end != "replacement") Assert.Null(runner.Current);
        }

        [Theory]
        [InlineData(false, true)]
        [InlineData(true, false)]
        public void AttackNeedsBothCombatAndMovementPermission(bool combat, bool movement)
        {
            var host = new FakeHost(); var reports = new List<TaskStatus>();
            var runner = new TaskRunner(new SafetyConfig { AllowCombat = combat, AllowMovement = movement }, TaskNames.Supported, host, reports.Add, () => 0);
            runner.Welcome(TaskNames.Supported);
            runner.Handle(new TaskRequest { Id = "a", Task = "attack", Args = JObject.Parse("{\"entity\":\"1:2\"}") });
            Assert.Empty(host.Bodies);
            Assert.Equal(Wire.ReasonNotAllowed, Assert.Single(reports).Reason);
        }
    }
}
