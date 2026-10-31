using System;
using System.Collections.Generic;
using System.Linq;
using Newtonsoft.Json.Linq;
using Renki.Protocol;
using Renki.Tasks;
using Xunit;

namespace Renki.Tests
{
    public sealed class CookingTests
    {
        static CookSlot[] Empty() => new[] { new CookSlot(), new CookSlot() };
        static CookSlot[] Food(string item = "RawMeat", int status = 0, long peer = 7) =>
            new[] { new CookSlot { Item = item, Status = status, Producer = peer }, new CookSlot() };
        static CookingBatch Batch() => new CookingBatch("RawMeat", 7, 2, 5, 2, 20);

        static CookingBatch Inserted()
        {
            var batch = Batch();
            Assert.Equal(CookAction.Insert, batch.Next(0, 2, 5, Empty(), true, 0));
            Assert.Equal(CookAction.Wait, batch.Next(.1, 1, 5, Food(), true, 0));
            return batch;
        }

        [Fact]
        public void NormalBonusYieldNeedsInputOwnedSlotSpawnAndInventoryReceipts()
        {
            var batch = Inserted();
            Assert.Equal(CookAction.Remove, batch.Next(21, 1, 5, Food("CookedMeat", 1), true, 0));
            Assert.Equal(CookAction.Collect, batch.Next(21.1, 1, 5, Empty(), true, 2));
            Assert.Equal(CookAction.Collect, batch.Next(22, 1, 6, Empty(), true, 2));
            Assert.Equal(CookAction.Succeeded, batch.Next(23, 1, 7, Empty(), true, 2, received: 2));
            Assert.Equal(2, batch.OutputCount);
            Assert.Equal(CookAction.Succeeded, batch.Next(24, 1, 7, Empty(), true, 2));
        }

        [Theory]
        [InlineData(false, true)]
        [InlineData(true, false)]
        [InlineData(false, false)]
        public void UncertainInsertionIsNeverRepeated(bool consumed, bool inserted)
        {
            var batch = Batch();
            Assert.Equal(CookAction.Insert, batch.Next(0, 2, 5, Empty(), true, 0));
            var slots = inserted ? Food() : Empty();
            Assert.Equal(CookAction.Wait, batch.Next(1, consumed ? 1 : 2, 5, slots, true, 0));
            Assert.Equal(CookAction.Failed, batch.Next(4, consumed ? 1 : 2, 5, slots, true, 0));
            Assert.Contains("not inserting again", batch.Failure);
            Assert.Equal(CookAction.Failed, batch.Next(5, 2, 5, Empty(), true, 0));
        }

        [Theory]
        [InlineData("busy")]
        [InlineData("cold")]
        [InlineData("ingredient changed")]
        public void AdmissionDoesNotConsumeAnIngredientWithoutAnEmptyHeatedStation(string cause)
        {
            var batch = Batch();
            Assert.Equal(CookAction.Failed, batch.Next(0, cause == "ingredient changed" ? 1 : 2, 5,
                cause == "busy" ? Food() : Empty(), cause != "cold", 0));
        }

        [Theory]
        [InlineData("burnt")]
        [InlineData("removed")]
        [InlineData("foreign producer")]
        [InlineData("another batch")]
        [InlineData("cold")]
        [InlineData("timeout")]
        public void ChangedOrUnsafeBatchCannotRemoveAnotherPlayersFood(string cause)
        {
            var batch = Inserted();
            var slots = cause == "removed" ? Empty() : Food(cause == "burnt" ? "Coal" : "RawMeat",
                cause == "burnt" ? 2 : 0, cause == "foreign producer" ? 8 : 7);
            if (cause == "another batch") slots[1] = new CookSlot { Item = "DeerMeat", Producer = 8 };
            Assert.Equal(CookAction.Failed, batch.Next(cause == "timeout" ? 51 : 2, 1, 5, slots, cause != "cold", 0));
        }

        [Fact]
        public void DisappearedFoodWithoutSpawnReceiptIsUncertainAndDoesNotRetryRemoval()
        {
            var batch = Inserted();
            Assert.Equal(CookAction.Remove, batch.Next(21, 1, 5, Food("CookedMeat", 1), true, 0));
            Assert.Equal(CookAction.Wait, batch.Next(22, 1, 6, Empty(), true, 0));
            Assert.Equal(CookAction.Failed, batch.Next(25, 1, 6, Empty(), true, 0));
            Assert.Contains("not removing again", batch.Failure);
        }

        [Fact]
        public void AMatchingGainFromAnotherDropCannotReplaceTheOriginPickupReceipt()
        {
            var batch = Inserted();
            Assert.Equal(CookAction.Remove, batch.Next(21, 1, 5, Food("CookedMeat", 1), true, 0));
            Assert.Equal(CookAction.Collect, batch.Next(22, 1, 6, Empty(), true, 1));
            Assert.Equal(CookAction.Failed, batch.Next(38, 1, 6, Empty(), true, 1));
        }

        [Theory]
        [InlineData(3, 8)]
        [InlineData(1, 7)]
        public void UnexpectedYieldOrUnrelatedFoodGainCannotProveTheBatch(int spawned, int inventory)
        {
            var batch = Inserted();
            Assert.Equal(CookAction.Remove, batch.Next(21, 1, 5, Food("CookedMeat", 1), true, 0));
            Assert.Equal(CookAction.Failed, batch.Next(22, 1, inventory, Empty(), true, spawned));
        }

        [Fact]
        public void UnsupportedRecipesAndInvalidCookingTimesAreRefused()
        {
            Assert.False(CookingBatch.Supported("Coal"));
            Assert.Equal("CookedDeerMeat", CookingBatch.Result("DeerMeat"));
            Assert.Equal("NeckTailGrilled", CookingBatch.Result("NeckTail"));
            foreach (var value in new[] { double.NaN, double.PositiveInfinity, 0, 121 })
                Assert.Throws<ArgumentException>(() => new CookingBatch("RawMeat", 7, 2, 5, 2, value));
        }

        static TaskRequest Request() => new TaskRequest
        {
            Id = "cook",
            Task = TaskNames.Interact,
            Args = JObject.Parse("{\"entity\":\"1:2\",\"cook\":\"RawMeat\"}"),
            TimeoutS = 90
        };

        [Fact]
        public void CookingIsExplicitAndCannotChangeIntoAnotherInteractionMode()
        {
            var req = Request(); Assert.Equal("RawMeat", ((InteractArgs)TaskArgs.Parse(req)).Cook);
            foreach (var key in new[] { "repair", "sleep", "alt", "hold" })
            {
                req.Args[key] = true; Assert.Throws<BadArgsException>(() => TaskArgs.Parse(req)); req.Args.Remove(key);
            }
            foreach (JToken value in new JToken[] { true, "Coal", "CookedMeat" })
            { req.Args["cook"] = value; Assert.Throws<BadArgsException>(() => TaskArgs.Parse(req)); }
            var safety = new SafetyConfig();
            Assert.NotNull(InteractRule.Check(InteractRule.Cooking, safety, cook: true));
            safety.AllowGather = true;
            Assert.Null(InteractRule.Check(InteractRule.Cooking, safety, cook: true));
            Assert.NotNull(InteractRule.Check(InteractRule.Cooking, safety));
            Assert.NotNull(InteractRule.Check(InteractRule.Station, safety, cook: true));
        }

        [Theory]
        [InlineData("pickup")]
        [InlineData("gather")]
        [InlineData("movement")]
        public void NestedPickupCannotBypassSessionOrNativeSafety(string withheld)
        {
            var host = new FakeHost(); var replies = new List<TaskStatus>();
            var safety = new SafetyConfig { AllowGather = withheld != "gather", AllowMovement = withheld != "movement" };
            var runner = new TaskRunner(safety, TaskNames.Supported, host, replies.Add, () => 0);
            runner.Welcome(TaskNames.Supported.Where(t => t != withheld)); runner.Handle(Request());
            Assert.Empty(host.Bodies); Assert.Equal(Wire.ReasonNotAllowed, replies.Last().Reason);
        }

        [Theory]
        [InlineData("stop")]
        [InlineData("disconnect")]
        [InlineData("replacement")]
        [InlineData("timeout")]
        public void StopReplacementAndReconnectCannotResumeAnInsertedBatch(string cause)
        {
            var host = new FakeHost(); var runner = new TaskRunner(new SafetyConfig { AllowGather = true },
                TaskNames.Supported, host, _ => { }, () => 0);
            runner.Welcome(TaskNames.Supported); runner.Handle(Request());
            var body = Assert.Single(host.Bodies);
            if (cause == "stop") runner.Handle(new TaskRequest { Task = TaskNames.Stop });
            if (cause == "disconnect") runner.EndSession("lost");
            if (cause == "replacement") runner.Handle(new TaskRequest { Task = TaskNames.MoveTo, Args = JObject.Parse("{\"to\":[1,2,3]}") });
            if (cause == "timeout") runner.FixedTick(91);
            Assert.True(body.Cancelled); runner.Welcome(TaskNames.Supported); Assert.Null(runner.Current);
        }
    }
}
