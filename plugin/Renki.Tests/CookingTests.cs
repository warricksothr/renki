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
        [Fact]
        public void SupportingFireConsumesOneWoodAndRequiresFuelAndHeatReceipts()
        {
            var fuel = new CookingFuel(10, 0);
            Assert.Equal(CookingFuelAction.Add, fuel.Next(0, 10, 0, false));
            Assert.Equal(CookingFuelAction.Wait, fuel.Next(.1, 9, 0, false));
            Assert.Equal(CookingFuelAction.Wait, fuel.Next(1, 9, .99, false));
            Assert.Equal(CookingFuelAction.Ready, fuel.Next(2, 9, .98, true));
            Assert.Equal(CookingFuelAction.Ready, fuel.Next(3, 9, .98, true));
        }

        [Theory]
        [InlineData(10, 1, true)]
        [InlineData(9, 0, false)]
        [InlineData(9, 1, false)]
        public void BlockedOrUnconfirmedFireDoesNotConsumeAnotherWood(int wood, double fuel, bool heated)
        {
            var receipt = new CookingFuel(10, 0);
            Assert.Equal(CookingFuelAction.Add, receipt.Next(0, 10, 0, false));
            Assert.Equal(CookingFuelAction.Wait, receipt.Next(1, wood, fuel, heated));
            Assert.Equal(CookingFuelAction.Failed, receipt.Next(6, wood, fuel, heated));
            Assert.Equal(CookingFuelAction.Failed, receipt.Next(7, 10, 0, false));
            Assert.Contains("not adding again", receipt.Failure);
        }

        [Fact]
        public void FuelingRequiresAnEmptyFireAndCarriedWood()
        {
            Assert.Equal(CookingFuelAction.Failed, new CookingFuel(0, 0).Next(0, 0, 0, false));
            Assert.Equal(CookingFuelAction.Failed, new CookingFuel(10, 1).Next(0, 10, 1, false));
        }

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
            Assert.Equal(CookAction.Failed, batch.Next(30, 1, 6, Empty(), true, 0));
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

        [Fact]
        public void RemoteBonusDeliveryWaitsForTheWholeRequestedYieldAndSettlement()
        {
            var batch = Inserted();
            Assert.Equal(CookAction.Remove, batch.Next(21, 1, 5, Food("CookedMeat", 1), true, 0));
            Assert.True(batch.ExpectYield(2));
            Assert.Equal(CookAction.Wait, batch.Next(22, 1, 6, Empty(), true, 1, received: 1));
            Assert.Equal(CookAction.Collect, batch.Next(23, 1, 7, Empty(), true, 2, received: 2, deliverySettled: false));
            Assert.Equal(CookAction.Succeeded, batch.Next(24, 1, 7, Empty(), true, 2, received: 2));
        }

        [Fact]
        public void RemotePickupCanArriveBeforeTheEmptySlotButCannotCompleteBeforeIt()
        {
            var batch = Inserted();
            Assert.Equal(CookAction.Remove, batch.Next(21, 1, 5, Food("CookedMeat", 1), true, 0));
            Assert.True(batch.ExpectYield(1));
            Assert.Equal(CookAction.Wait, batch.Next(22, 1, 6, Food("CookedMeat", 1), true, 1, received: 1));
            Assert.Equal(CookAction.Collect, batch.Next(23, 1, 6, Empty(), true, 1, received: 1, deliverySettled: false));
            Assert.Equal(CookAction.Succeeded, batch.Next(24, 1, 6, Empty(), true, 1, received: 1));
        }

        [Theory]
        [InlineData(0)]
        [InlineData(3)]
        public void InvalidOrRepeatedNormalRemovalCannotProveDelivery(int count)
        {
            var batch = Inserted();
            Assert.Equal(CookAction.Remove, batch.Next(21, 1, 5, Food("CookedMeat", 1), true, 0));
            Assert.False(batch.ExpectYield(count));
            Assert.Equal(CookAction.Failed, batch.Next(22, 1, 6, Empty(), true, 1, received: 1));
            batch = Inserted();
            Assert.Equal(CookAction.Remove, batch.Next(21, 1, 5, Food("CookedMeat", 1), true, 0));
            Assert.True(batch.ExpectYield(1)); Assert.False(batch.ExpectYield(1));
        }

        static CookingRemoteOutput Remote(int count = 1) => new CookingRemoteOutput("CookedMeat", 9, 7, true, count, new[] { "old" });

        [Theory]
        [InlineData("old", "CookedMeat", 9, 0, 0, 7)]
        [InlineData("new", "RawMeat", 9, 0, 0, 7)]
        [InlineData("new", "CookedMeat", 8, 0, 0, 7)]
        [InlineData("new", "CookedMeat", 9, -.2, 0, 7)]
        [InlineData("new", "CookedMeat", 9, 3.1, 0, 7)]
        [InlineData("new", "CookedMeat", 9, 0, 2.1, 7)]
        [InlineData("new", "CookedMeat", 9, 0, 0, 8)]
        [InlineData("new", "CookedMeat", 9, double.NaN, 0, 7)]
        public void OldReloadedUnrelatedAndOutOfWindowDropsAreNotRemoteReceipts(string id, string prefab,
            long creator, double born, double distance, long crafter)
        {
            var remote = Remote();
            Assert.Equal(CookingDropMatch.Ignore, remote.Observe(id, prefab, creator, born, distance, 1, crafter));
            Assert.False(remote.Complete); Assert.Equal(0, remote.Count);
        }

        [Fact]
        public void DuplicateIdentityDoesNotCountTwiceAndExtraOrMergedOutputsAreAmbiguous()
        {
            var remote = Remote(2);
            Assert.Equal(CookingDropMatch.Accepted, remote.Observe("one", "CookedMeat", 9, .1, .2, 1, 7));
            Assert.Equal(CookingDropMatch.Ignore, remote.Observe("one", "CookedMeat", 9, .1, .2, 1, 7));
            Assert.False(remote.Complete);
            Assert.Equal(CookingDropMatch.Accepted, remote.Observe("two", "CookedMeat", 9, .3, .2, 1, 7));
            Assert.True(remote.Complete);
            Assert.Equal(CookingDropMatch.Ambiguous, remote.Observe("three", "CookedMeat", 9, .3, .2, 1, 7));
            Assert.False(remote.Complete);
            Assert.Equal(CookingDropMatch.Ambiguous, Remote().Observe("merged", "CookedMeat", 9, .1, .2, 2, 7));
        }

        [Fact]
        public void StationsWithoutCrafterMetadataStillRequireTheRemoteCreatorAndBirthWindow()
        {
            var remote = new CookingRemoteOutput("CookedMeat", 9, 7, false, 1, Array.Empty<string>());
            Assert.Equal(CookingDropMatch.Ignore, remote.Observe("wrong", "CookedMeat", 8, .1, .2, 1, 0));
            Assert.Equal(CookingDropMatch.Ignore, remote.Observe("old", "CookedMeat", 9, -2, .2, 1, 0));
            Assert.Equal(CookingDropMatch.Accepted, remote.Observe("new", "CookedMeat", 9, .1, .2, 1, 0));
            Assert.True(remote.Complete);
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
