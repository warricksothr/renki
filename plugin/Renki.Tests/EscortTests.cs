using System;
using System.Collections.Generic;
using System.Linq;
using Newtonsoft.Json.Linq;
using Renki.Protocol;
using Renki.Movement;
using Renki.Tasks;
using Xunit;

namespace Renki.Tests
{
    public sealed class EscortTests
    {
        [Fact]
        public void NearbyLeaderAcrossAWallOrOnAnotherFloorStillNeedsARoute()
        {
            var pace = new FollowPace();
            Assert.Equal(Gait.Stop, pace.Decide(1, 3.5, RunMode.Auto, false));
            Assert.Equal(Gait.Walk, pace.Decide(1, 3.5, RunMode.Auto, false, blocked: true, mayStop: false));
            Assert.Equal(Gait.Walk, pace.Decide(0, 3.5, RunMode.Auto, false, mayStop: false));
            Assert.Equal(Gait.Stop, pace.Decide(1, 3.5, RunMode.Auto, false));
        }

        [Theory]
        [InlineData(1, true, true, true, EscortAction.Defend)]
        [InlineData(2, true, true, true, EscortAction.Escape)]
        [InlineData(1, false, true, true, EscortAction.Escape)]
        [InlineData(1, true, false, true, EscortAction.Escape)]
        [InlineData(1, true, true, false, EscortAction.Escape)]
        [InlineData(0, false, false, false, EscortAction.Collect)]
        public void DefenseRequiresOneConfirmedEligibleAttackerAndReserves(int threats, bool eligible, bool ready, bool defend, EscortAction expected)
        {
            Assert.Equal(expected, EscortPolicy.Decide(true, 3, true, threats, eligible, ready, defend, true, true, false, 0));
        }

        [Theory]
        [InlineData(false, 3, true, true, 0, EscortAction.Wait)]
        [InlineData(true, 11, true, true, 0, EscortAction.Follow)]
        [InlineData(true, 3, false, true, 0, EscortAction.Follow)]
        [InlineData(true, 3, true, false, 0, EscortAction.Follow)]
        [InlineData(true, 3, true, true, 12, EscortAction.Follow)]
        public void DetoursYieldToLeaderLossMovementLeashCapacityAndTimeout(bool visible, double gap, bool still, bool space, double age, EscortAction expected)
        {
            Assert.Equal(expected, EscortPolicy.Decide(visible, gap, still, 0, false, true, true, true, space, true, age));
        }

        [Fact]
        public void LosingTheLeaderCannotAuthorizeAChaseButDoesNotBlockEscape()
        {
            Assert.Equal(EscortAction.Escape, EscortPolicy.Decide(false, 3, true, 1, true, true, true, true, true, false, 0));
            Assert.Equal(EscortAction.Escape, EscortPolicy.Decide(true, 17, true, 1, true, true, true, true, true, false, 0));
        }

        [Theory]
        [InlineData(40, 50, 35, 2, true, true, true)]
        [InlineData(29, 30, 35, 2, true, true, false)]
        [InlineData(39, 50, 35, 2, true, true, false)]
        [InlineData(40, 50, 34, 2, true, true, false)]
        [InlineData(40, 50, 35, 1, true, true, false)]
        [InlineData(40, 50, 35, 2, false, true, false)]
        [InlineData(40, 50, 35, 2, true, false, false)]
        public void EntryPreservesHealthFoodEquipmentAndEscapeReserves(double hp, double max, double stamina, double armour, bool food, bool weapon, bool expected) =>
            Assert.Equal(expected, EscortPolicy.Ready(hp, max, stamina, armour, food, weapon));

        [Theory]
        [InlineData(true, false, true, true)]
        [InlineData(true, true, false, true)]
        [InlineData(true, true, true, false)]
        [InlineData(true, null, false, false)]
        [InlineData(true, false, null, false)]
        [InlineData(false, false, false, false)]
        public void OnlyNamedKnownPvpProtectedAlliesMayBeExempted(bool named, bool? own, bool? other, bool expected) =>
            Assert.Equal(expected, EscortPolicy.SafeAlly(named, own, other));

        [Theory]
        [InlineData("{}")]
        [InlineData("{\"player\":\" \"}")]
        [InlineData("{\"player\":\"Nimah\",\"distance\":0}")]
        [InlineData("{\"player\":\"Nimah\",\"distance\":9}")]
        [InlineData("{\"player\":\"Nimah\",\"protect\":[1]}")]
        [InlineData("{\"player\":\"Nimah\",\"collect\":\"yes\"}")]
        public void InvalidRequestsCannotStart(string json) =>
            Assert.Throws<BadArgsException>(() => TaskArgs.Parse(Request(json)));

        static TaskRequest Request(string json = "{\"player\":\"Nimah\"}") =>
            new TaskRequest { Id = "e-1", Task = TaskNames.Escort, Args = JObject.Parse(json) };

        [Theory]
        [InlineData("follow")]
        [InlineData("flee")]
        [InlineData("attack")]
        [InlineData("pickup")]
        [InlineData("interact")]
        [InlineData("combat safety")]
        [InlineData("gather safety")]
        public void WrapperCannotBypassEitherChildPermissionLayer(string withheld)
        {
            var host = new FakeHost(); var replies = new List<TaskStatus>();
            var safety = new SafetyConfig { AllowGather = withheld != "gather safety", AllowCombat = withheld != "combat safety" };
            var runner = new TaskRunner(safety, TaskNames.Supported, host, replies.Add, () => 0);
            runner.Welcome(TaskNames.Supported.Where(t => t != withheld));
            runner.Handle(Request());
            Assert.Empty(host.Bodies);
            Assert.Equal(Wire.ReasonNotAllowed, replies.Last().Reason);
            // An explicit follow-only escort remains available with movement permission.
            runner.Handle(Request("{\"player\":\"Nimah\",\"collect\":false,\"defend\":false}"));
            if (withheld == "follow" || withheld == "flee") Assert.Empty(host.Bodies);
            else Assert.Single(host.Bodies);
        }

        [Theory]
        [InlineData("stop")]
        [InlineData("replacement")]
        [InlineData("disconnect")]
        [InlineData("welcome")]
        [InlineData("timeout")]
        public void CompositionCannotSurviveCancellationOrReconnect(string cause)
        {
            var host = new FakeHost(); var safety = new SafetyConfig { AllowGather = true, AllowCombat = true };
            var runner = new TaskRunner(safety, TaskNames.Supported, host, _ => { }, () => 0);
            runner.Welcome(TaskNames.Supported);
            var request = Request(); request.TimeoutS = 10;
            runner.Handle(request);
            var body = host.Bodies.Single();
            if (cause == "stop") runner.Handle(new TaskRequest { Id = "s", Task = TaskNames.Stop });
            if (cause == "replacement") runner.Handle(new TaskRequest { Id = "f", Task = TaskNames.Follow, Args = JObject.Parse("{\"player\":\"Nimah\"}") });
            if (cause == "disconnect") runner.EndSession("connection lost");
            if (cause == "welcome") runner.Welcome(TaskNames.Supported);
            if (cause == "timeout") runner.FixedTick(11);
            Assert.True(body.Cancelled);
            runner.Welcome(TaskNames.Supported);
            Assert.Null(runner.Current);
        }
    }
}
