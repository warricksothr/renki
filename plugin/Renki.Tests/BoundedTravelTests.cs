using System;
using Newtonsoft.Json.Linq;
using Renki.Movement;
using Renki.Protocol;
using Xunit;

namespace Renki.Tests
{
    public sealed class BoundedTravelTests
    {
        static TravelScope Scope() => new TravelScope { Center = V3.Zero, Radius = 60, TravelM = 50, DurationS = 40 };
        [Fact]
        public void CountsDetoursAndHeightAndKeepsFinalReceipt()
        {
            var s = new BoundedTravel(Scope(), V3.Zero, 0);
            Assert.Null(s.Observe(new V3(3, 4, 0), 1));
            Assert.Null(s.Observe(V3.Zero, 2));
            Assert.Equal(10, s.Travel, 6);
            var receipt = JObject.Parse(s.Receipt(V3.Zero).Substring(7));
            Assert.Equal(10, (double)receipt["travel"]);
            Assert.Equal(3, ((JArray)receipt["trail"]).Count);
            Assert.Equal(0, (double)receipt["final"][0]);
        }
        [Fact]
        public void BoundaryBudgetAndDeadlineNeverResetOnObservation()
        {
            var scope = Scope(); var s = new BoundedTravel(scope, V3.Zero, 10);
            Assert.False(s.Clear(V3.Zero, new V3(61, 0, 0)));
            Assert.Null(s.Observe(new V3(20, 0, 0), 11));
            Assert.Null(s.Observe(new V3(0, 0, 0), 12));
            Assert.Equal("survey leg travel limit", s.Observe(new V3(10, 0, 0), 13));
            var timed = new BoundedTravel(scope, V3.Zero, 10);
            Assert.Equal("survey leg time limit", timed.Observe(V3.Zero, 50));
            Assert.Equal("outside survey boundary", new BoundedTravel(scope, V3.Zero, 0).Observe(new V3(61, 0, 0), 1));
            Assert.Equal("invalid position", timed.Observe(new V3(double.NaN, 0, 0), 51));
        }

        [Theory]
        [InlineData("dry", false)]
        [InlineData("avoidHostiles", false)]
        [InlineData("recoverThreats", true)]
        [InlineData("to", "0:1")]
        [InlineData("stopDistance", 100)]
        [InlineData("stopDistance", 0)]
        public void ScopedArgsRejectUnsafeOptions(string key, object value)
        {
            var args = JObject.Parse(@"{""to"":[-15,40,0],""dry"":true,""avoidHostiles"":true,""scope"":{""center"":[-25,40,0],""radius"":60,""travelM"":50,""durationS"":40}}");
            Assert.NotNull(TaskArgs.Parse(new TaskRequest { Task = "move_to", Args = args }));
            args[key] = JToken.FromObject(value);
            Assert.Throws<BadArgsException>(() => TaskArgs.Parse(new TaskRequest { Task = "move_to", Args = args }));
        }
        [Theory]
        [InlineData(0, 50, 40)]
        [InlineData(251, 50, 40)]
        [InlineData(60, 501, 40)]
        [InlineData(60, 50, 91)]
        [InlineData(double.NaN, 50, 40)]
        public void RejectsInvalidScope(double radius, double travel, double duration)
        {
            Assert.False(new TravelScope { Center = V3.Zero, Radius = radius, TravelM = travel, DurationS = duration }.Valid);
        }
    }
}
