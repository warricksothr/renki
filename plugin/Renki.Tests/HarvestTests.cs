using Renki.Tasks;
using Renki.Protocol;
using Newtonsoft.Json.Linq;
using System.Collections.Generic;
using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public sealed class HarvestTests
    {
        [Fact]
        public void FoodMaximumDecayIsNotDamage()
        {
            Assert.False(HarvestHealth.Hurt(115, 114.7, 114.7));
            Assert.True(HarvestHealth.Hurt(115, 113.7, 114.7));
            Assert.True(HarvestHealth.Hurt(115, 114.7, 115));
        }

        [Fact]
        public void HarvestWaitsForRecoveryAndCancelsOnTheFirstHealthLoss()
        {
            Assert.True(HarvestHealth.Ready(25, 25));
            Assert.True(HarvestHealth.Ready(20, 25));
            Assert.False(HarvestHealth.Ready(19.9, 25));
            Assert.False(HarvestHealth.Ready(25, 0));
            Assert.False(HarvestHealth.Hurt(25, 25, 25));
            Assert.True(HarvestHealth.Hurt(25, 24.9, 25));
            Assert.True(HarvestHealth.Hurt(25, 20.4, 25));
        }

        [Fact]
        public void RemoteLogsNeedQuietPosesEvenWhenLocalVelocityIsZero()
        {
            var quiet = new HarvestStillness();
            var up = new V3(0, 1, 0); var forward = new V3(0, 0, 1);
            Assert.False(quiet.Observe(V3.Zero, up, forward, false, 0));
            Assert.False(quiet.Observe(V3.Zero, up, forward, false, 1));
            Assert.True(quiet.Observe(V3.Zero, up, forward, false, 2));
            // Individually small network movements accumulate relative to
            // the quiet anchor; resetting the anchor every frame misses them.
            quiet.Observe(new V3(.01, 0, 0), up, forward, false, 2.1);
            Assert.False(quiet.Observe(new V3(.03, 0, 0), up, forward, false, 2.2));
            Assert.False(quiet.Observe(new V3(.03, 0, 0), up, forward, false, 4));
            Assert.True(quiet.Observe(new V3(.03, 0, 0), up, forward, false, 4.3));
            Assert.False(quiet.Observe(new V3(.03, 0, 0), up, new V3(.02, 0, .9998), false, 4.4));
            Assert.False(quiet.Observe(new V3(.03, 0, 0), up, new V3(.02, 0, .9998), true, 7));
        }

        sealed class Tool
        {
            public int Tier;
            public double Chop;
            public bool Usable = true;
        }
        static Tool Choose(int minimum, params Tool[] tools) => HarvestTool.Best(tools, minimum, t => t.Tier, t => t.Chop, t => t.Usable);

        [Fact]
        public void ToolMustBeUsableAndAbleToDamageTheResource()
        {
            var sword = new Tool { Tier = 4 };
            var broken = new Tool { Tier = 3, Chop = 100, Usable = false };
            var stone = new Tool { Tier = 0, Chop = 20 };
            var flint = new Tool { Tier = 1, Chop = 30 };
            Assert.Same(flint, Choose(0, sword, broken, stone, flint));
            Assert.Null(Choose(2, sword, broken, stone, flint));
            Assert.Null(Choose(0, sword, broken));
            // Upgraded lower-tier equipment can do more damage; min tier
            // is an eligibility check, not a reason to waste a slower tool.
            stone.Chop = 40;
            Assert.Same(stone, Choose(0, stone, flint));
            Assert.Same(flint, Choose(1, stone, flint));
        }

        [Fact]
        public void AbsenceOrCompetingDamageDoesNotClaimHarvestSuccess()
        {
            var evidence = new HarvestEvidence();
            Assert.False(evidence.Broken(10));
            evidence.Hit(false, 10);
            Assert.False(evidence.Broken(10));
            evidence.Hit(true, 11);
            Assert.True(evidence.Broken(11.5));
            Assert.False(evidence.Broken(14));
            evidence.Hit(false, 11.6);
            Assert.False(evidence.Broken(11.7));
            evidence.Hit(true, 12);
            Assert.True(evidence.Broken(12.1));
        }

        [Fact]
        public void HarvestNeedsGatherAndMovementAndStopCancelsItsBody()
        {
            var safety = new SafetyConfig();
            var host = new FakeHost();
            var reports = new List<TaskStatus>();
            var runner = new TaskRunner(safety, TaskNames.Supported, host, reports.Add, () => 0);
            var request = new TaskRequest { Id = "harvest", Task = TaskNames.Harvest, Args = JObject.Parse("{\"entity\":\"1:2\",\"collect\":false}") };
            runner.Welcome(TaskNames.Supported);
            runner.Handle(request);
            Assert.Equal(Wire.ReasonNotAllowed, reports[reports.Count - 1].Reason);
            safety.AllowGather = true; safety.AllowMovement = false;
            runner.Welcome(TaskNames.Supported);
            runner.Handle(request);
            Assert.Equal(Wire.ReasonNotAllowed, reports[reports.Count - 1].Reason);
            safety.AllowMovement = true;
            runner.Welcome(TaskNames.Supported);
            runner.Handle(request);
            Assert.Equal(Wire.StatusRunning, reports[reports.Count - 1].Status);
            runner.Handle(new TaskRequest { Id = "stop", Task = TaskNames.Stop });
            Assert.True(host.Bodies[0].Cancelled);
            Assert.Null(runner.Current);
        }
    }
}
