using System;
using System.Collections.Generic;
using System.Linq;
using Newtonsoft.Json.Linq;
using Renki.Protocol;
using Renki.Tasks;
using Xunit;

namespace Renki.Tests
{
    public sealed class FuelTests
    {
        [Theory]
        [InlineData(4, 2.5)] // only synchronized fuel changed
        [InlineData(3, 1.5)] // only pack changed
        [InlineData(4, 1.5)] // neither changed, including native full ACK
        public void MissingReceiptNeverReissuesFuel(int pack, double fuel)
        {
            var batch = new FireplaceFuel(2, 20);
            Assert.Equal(FuelAction.Add, batch.Next(0, 4, 1.5));
            Assert.Equal(FuelAction.Wait, batch.Next(1, pack, fuel));
            Assert.Equal(FuelAction.Failed, batch.Next(5.01, pack, fuel));
            Assert.Equal(FuelAction.Failed, batch.Next(6, 3, 2.5));
            Assert.Equal(0, batch.Added);
        }

        [Fact]
        public void TwoUnitsWaitForEachRemoteReceiptAndCountOnlyVerifiedUses()
        {
            var batch = new FireplaceFuel(2, 20);
            Assert.Equal(FuelAction.Add, batch.Next(0, 4, 1.5));
            Assert.Equal(FuelAction.Wait, batch.Next(.1, 3, 1.5));
            Assert.Equal(FuelAction.Wait, batch.Next(2, 3, 2.49));
            Assert.Equal(1, batch.Added);
            Assert.Equal(FuelAction.Add, batch.Next(2.25, 3, 2.48));
            Assert.Equal(FuelAction.Wait, batch.Next(2.3, 2, 2.48));
            Assert.Equal(FuelAction.Ready, batch.Next(3, 2, 3.47));
            Assert.Equal(2, batch.Added);
            Assert.Equal(FuelAction.Ready, batch.Next(4, 2, 3.46));
        }

        [Theory]
        [InlineData(19.01, 0)]
        [InlineData(20, 0)]
        public void NativeCeilingFullRuleStopsWithoutConsumption(double fuel, int pack)
        {
            var batch = new FireplaceFuel(2, 20);
            Assert.Equal(FuelAction.Ready, batch.Next(0, pack, fuel));
            Assert.Equal(0, batch.Added);
            Assert.Equal("fire full", batch.Detail);
        }

        [Fact]
        public void BecomingFullAfterOneVerifiedUnitReportsPartialSuccess()
        {
            var batch = new FireplaceFuel(2, 20);
            Assert.Equal(FuelAction.Add, batch.Next(0, 4, 18.5));
            Assert.Equal(FuelAction.Ready, batch.Next(.1, 3, 19.5));
            Assert.Equal(1, batch.Added);
        }

        [Theory]
        [InlineData(2, 2.5)]
        [InlineData(5, 2.5)]
        [InlineData(3, 3.5)]
        [InlineData(3, double.NaN)]
        [InlineData(3, double.PositiveInfinity)]
        public void AmbiguousReceiptStopsWithoutAnotherUnit(int pack, double fuel)
        {
            var batch = new FireplaceFuel(2, 20);
            batch.Next(0, 4, 1.5);
            Assert.Equal(FuelAction.Failed, batch.Next(1, pack, fuel));
            Assert.Equal(0, batch.Added);
            Assert.Equal(FuelAction.Failed, batch.Next(2, 3, 2.5));
        }

        [Theory]
        [InlineData(0, 20)]
        [InlineData(17, 20)]
        [InlineData(2, double.NaN)]
        [InlineData(2, double.PositiveInfinity)]
        public void InvalidBoundsNeverRequestUse(int count, double capacity)
            => Assert.Equal(FuelAction.Failed, new FireplaceFuel(count, capacity).Next(0, 4, 1));

        static TaskRequest Request(string args = "{\"entity\":\"1:2\",\"item\":\"Wood\",\"count\":2}")
            => new TaskRequest { Id = "fuel", Task = TaskNames.AddTo, Args = JObject.Parse(args), TimeoutS = 30 };

        [Theory]
        [InlineData("{}")]
        [InlineData("{\"entity\":\"1:2\",\"item\":\"Wood\",\"count\":0}")]
        [InlineData("{\"entity\":\"1:2\",\"item\":\"Wood\",\"count\":17}")]
        [InlineData("{\"entity\":\"1:2\",\"item\":\"Wood\",\"count\":1.5}")]
        [InlineData("{\"entity\":\"1:2\",\"item\":\"Wood\",\"count\":\"all\"}")]
        [InlineData("{\"entity\":\"1:2\",\"item\":\"Wood\",\"count\":2,\"toggle\":true}")]
        public void MalformedOrBroadenedRequestsAreRejected(string json)
            => Assert.Throws<BadArgsException>(() => TaskArgs.Parse(Request(json)));

        static SafetyConfig Enabled() => new SafetyConfig { AllowStations = true, AllowGather = true };
        [Theory]
        [InlineData("station")]
        [InlineData("gather")]
        [InlineData("movement")]
        [InlineData("session")]
        public void EveryPermissionIsRequiredBeforeTheNativeTaskStarts(string missing)
        {
            var safety = Enabled();
            if (missing == "station") safety.AllowStations = false;
            if (missing == "gather") safety.AllowGather = false;
            if (missing == "movement") safety.AllowMovement = false;
            var host = new FakeHost(); var replies = new List<TaskStatus>();
            var runner = new TaskRunner(safety, TaskNames.Supported, host, replies.Add, () => 0);
            runner.Welcome(TaskNames.Supported.Where(t => missing != "session" || t != TaskNames.AddTo));
            runner.Handle(Request());
            Assert.Empty(host.Bodies); Assert.Equal(Wire.ReasonNotAllowed, replies.Last().Reason);
            Assert.False(new SafetyConfig().Permits(TaskNames.AddTo));
        }

        [Theory]
        [InlineData("stop")]
        [InlineData("disconnect")]
        [InlineData("replacement")]
        [InlineData("timeout")]
        public void InterruptedFuelingIsCancelledAndNeverResumed(string cause)
        {
            double now = 0; var host = new FakeHost();
            var runner = new TaskRunner(Enabled(), TaskNames.Supported, host, _ => { }, () => now);
            runner.Welcome(TaskNames.Supported); runner.Handle(Request());
            var body = Assert.Single(host.Bodies);
            if (cause == "stop") runner.Handle(new TaskRequest { Task = TaskNames.Stop });
            if (cause == "disconnect") { runner.EndSession("lost"); runner.Welcome(TaskNames.Supported); }
            if (cause == "replacement") { var req = Request(); req.Id = "replacement"; runner.Handle(req); }
            if (cause == "timeout") { now = 31; runner.FixedTick(31); }
            Assert.True(body.Cancelled);
        }
    }
}
