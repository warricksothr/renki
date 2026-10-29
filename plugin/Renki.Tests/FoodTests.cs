using System.Collections.Generic;
using Newtonsoft.Json.Linq;
using Renki.Protocol;
using Renki.Tasks;
using Xunit;

namespace Renki.Tests
{
    public class FoodTests
    {
        [Fact]
        public void EatingPreservesWalkingAndRespectsWelcomeAndEnabledSwitch()
        {
            var safety = new SafetyConfig();
            var host = new FakeHost();
            var reports = new List<TaskStatus>();
            var runner = new TaskRunner(safety, TaskNames.Supported, host, reports.Add, () => 0);
            var eat = new TaskRequest { Id = "eat", Task = TaskNames.UseItem, Args = JObject.Parse("{\"item\":\"Raspberry\"}") };
            runner.Welcome(TaskNames.M1);
            runner.Handle(eat);
            Assert.Equal(Wire.ReasonNotAllowed, reports[reports.Count - 1].Reason);
            Assert.Empty(host.Instants);

            runner.Welcome(TaskNames.Supported);
            runner.Handle(new TaskRequest { Id = "walk", Task = TaskNames.MoveTo, Args = JObject.Parse("{\"to\":[1,2,3]}") });
            runner.Handle(eat);
            Assert.Equal("walk", runner.Current.Id);
            Assert.False(host.Bodies[0].Cancelled);
            Assert.Contains(TaskNames.UseItem, host.Instants);
            Assert.Equal(Wire.StatusSucceeded, reports[reports.Count - 1].Status);

            safety.AllowMovement = false;
            Assert.True(safety.Permits(TaskNames.UseItem));
            safety.Enabled = false;
            Assert.False(safety.Permits(TaskNames.UseItem));
        }
    }
}
