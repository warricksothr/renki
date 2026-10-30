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
            // The host reanchors at the bot's position when within 12 m.
            f.TargetDistance = f.DistanceFromStart = 12;
            Assert.Equal(MeleeAction.Approach, policy.Decide(f, 8));
            Assert.True(policy.Engaged);
            f.InRange = true; f.TargetDistance = 1; f.DistanceFromStart = 11;
            Assert.Equal(MeleeAction.Strike, policy.Decide(f, 13));
            Assert.Equal(MeleeAction.Retreat, policy.Decide(f, 20));
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

        [Theory]
        [InlineData("stop")]
        [InlineData("replacement")]
        [InlineData("timeout")]
        [InlineData("deadman")]
        [InlineData("reconnect")]
        [InlineData("death")]
        public void AttackUsesTheBodyCancellationLifecycle(string end)
        {
            var host = new FakeHost(); var reports = new List<TaskStatus>();
            var runner = new TaskRunner(new SafetyConfig { AllowCombat = true }, TaskNames.Supported, host, reports.Add, () => 0);
            runner.Welcome(TaskNames.Supported);
            runner.Handle(new TaskRequest { Id = "a", Task = "attack", Args = JObject.Parse("{\"entity\":\"1:2\"}"), TimeoutS = 1 });
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
