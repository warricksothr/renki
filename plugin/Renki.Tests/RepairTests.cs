using Newtonsoft.Json.Linq;
using Renki.Protocol;
using Renki.Tasks;
using Xunit;

namespace Renki.Tests
{
    public sealed class RepairTests
    {
        [Fact]
        public void RepairIsAnExplicitStationOnlyActionBehindGatherAndMovementGates()
        {
            var safety = new SafetyConfig();
            Assert.Equal(Wire.ReasonNotAllowed, InteractRule.Check(InteractRule.Station, safety, repair: true).Reason);
            safety.AllowGather = true;
            Assert.Null(InteractRule.Check(InteractRule.Station, safety, repair: true));
            Assert.NotNull(InteractRule.Check(InteractRule.Station, safety));
            foreach (var kind in new[] { InteractRule.Bed, InteractRule.Door, InteractRule.Container, InteractRule.Boss, InteractRule.Other })
                Assert.Equal(Wire.ReasonInvalidTarget, InteractRule.Check(kind, safety, repair: true).Reason);
            safety.AllowMovement = false;
            Assert.False(safety.Permits(TaskNames.Interact));
        }

        [Fact]
        public void RepairCannotBecomeSleepHoldOrAlternateUse()
        {
            var request = new TaskRequest { Task = TaskNames.Interact, Args = JObject.Parse("{\"entity\":\"1:2\",\"repair\":true}") };
            Assert.True(((InteractArgs)TaskArgs.Parse(request)).Repair);
            foreach (var other in new[] { "sleep", "hold", "alt" })
            {
                request.Args[other] = true;
                Assert.Throws<BadArgsException>(() => TaskArgs.Parse(request));
                request.Args.Remove(other);
            }
            request.Args["repair"] = "yes";
            Assert.Throws<BadArgsException>(() => TaskArgs.Parse(request));
        }
    }
}
