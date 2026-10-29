using Renki.Tasks;
using Renki.Protocol;
using Newtonsoft.Json.Linq;
using System.Collections.Generic;
using Xunit;

namespace Renki.Tests
{
    public sealed class HarvestTests
    {
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
