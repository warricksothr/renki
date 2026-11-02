using System.Collections.Generic;
using System.Linq;
using Newtonsoft.Json.Linq;
using Renki.Protocol;
using Renki.Tasks;
using Xunit;

namespace Renki.Tests
{
    public sealed class TransferTests
    {
        [Theory]
        [InlineData(12, "unreachable", true)]
        [InlineData(3, "stuck", true)]
        [InlineData(12.01, "unreachable", false)]
        [InlineData(3, "deep_water", false)]
        [InlineData(3, "not_allowed", false)]
        [InlineData(3, "refused", false)]
        [InlineData(3, "stopped", false)]
        [InlineData(double.NaN, "unreachable", false)]
        [InlineData(double.PositiveInfinity, "unreachable", false)]
        public void AlternateChestApproachesCannotBroadenOtherRefusals(double distance, string reason, bool allowed)
        {
            Assert.Equal(allowed, Transfer.CanReposition(new V3(0, 0, 0), new V3(distance, 0, 0), Outcome.Failed(reason)));
        }

        [Fact]
        public void CapturedBedToFoodChestFailureAllowsTheSameMeasuredSidesAsConditionalCargo()
        {
            // task-ql427mlm: ordinary withdrawal failed without moving because
            // its goal was inside piece_chest_wood/woodchest. Item matching
            // governs inventory, not whether a reachable chest side is tried.
            var from = new V3(-31.08, 36.49, 15.19);
            var chest = new V3(-34.79, 36.19, 10.07);
            Assert.True(Transfer.CanReposition(from, chest, Outcome.Failed(Wire.ReasonUnreachable)));
            Assert.False(Transfer.CanReposition(from, chest, null));
            Assert.False(Transfer.CanReposition(from, chest, Outcome.Failed(Wire.ReasonRefused)));
        }

        [Fact]
        public void ConditionalDepositRequiresEveryPrefabCurrentlyInTheDestination()
        {
            var items = new[] { new TransferItem { Prefab = "Coal", Count = 5 }, new TransferItem { Prefab = "CopperOre", Count = 3 } };
            Assert.False(Transfer.MatchesDestination(items, new string[0]));
            Assert.False(Transfer.MatchesDestination(items, new[] { "Wood", "Coal" }));
            Assert.False(Transfer.MatchesDestination(items, new[] { "Coal", "copperore" }));
            Assert.True(Transfer.MatchesDestination(items, new[] { "Coal", "CopperOre", "Wood" }));
            var args = JObject.Parse("{\"container\":\"1:2\",\"matchExisting\":true,\"items\":[{\"prefab\":\"Coal\",\"count\":5}]}");
            Assert.True(((TransferArgs)TaskArgs.Parse(new TaskRequest { Task = TaskNames.Deposit, Args = args })).MatchExisting);
            Assert.Throws<BadArgsException>(() => TaskArgs.Parse(new TaskRequest { Task = TaskNames.Withdraw, Args = args }));
        }

        sealed class Stack
        {
            public string Prefab;
            public int Count;
            public bool Equipped;
        }

        [Fact]
        public void PartialTransferConservesItemsAndReportsShortfall()
        {
            var source = new[] { new Stack { Prefab = "Wood", Count = 40 }, new Stack { Prefab = "Wood", Count = 30 }, new Stack { Prefab = "AxeStone", Count = 1, Equipped = true } };
            var room = 55;
            var stored = 0;
            var result = Transfer.Run(new[] { new TransferItem { Prefab = "Wood", Count = -1 }, new TransferItem { Prefab = "AxeStone", Count = -1 }, new TransferItem { Prefab = "Resin", Count = 3 } },
                source, s => s.Prefab, s => s.Count, s => s.Equipped, (stack, requested) =>
                {
                    var moved = System.Math.Min(room, requested);
                    stack.Count -= moved;
                    stored += moved;
                    room -= moved;
                    return moved;
                });
            Assert.Equal(55, result.Moved["Wood"]);
            Assert.Equal(15, result.Remaining["Wood"]);
            Assert.Equal(1, result.Remaining["AxeStone"]);
            Assert.Equal(3, result.Remaining["Resin"]);
            Assert.Equal(70, stored + source.Where(s => s.Prefab == "Wood").Sum(s => s.Count));
            Assert.Equal(1, source[2].Count);
        }

        [Fact]
        public void NamedCountIsAcrossStacksAndLeavesTheRest()
        {
            var source = new[] { new Stack { Prefab = "Wood", Count = 4 }, new Stack { Prefab = "Wood", Count = 20 } };
            var result = Transfer.Run(new[] { new TransferItem { Prefab = "Wood", Count = 10 } }, source,
                s => s.Prefab, s => s.Count, s => s.Equipped, (s, n) => { s.Count -= n; return n; });
            Assert.Equal(10, result.Moved["Wood"]);
            Assert.Empty(result.Remaining);
            Assert.Equal(14, source.Sum(s => s.Count));
        }

        [Theory]
        [InlineData("0")]
        [InlineData("-1")]
        [InlineData("1.5")]
        [InlineData("\"5\"")]
        [InlineData("null")]
        public void MalformedCountsAreRefused(string count)
        {
            var req = new TaskRequest { Task = TaskNames.Deposit, Args = JObject.Parse("{\"container\":\"1:2\",\"items\":[{\"prefab\":\"Wood\",\"count\":" + count + "}]}") };
            Assert.Throws<BadArgsException>(() => TaskArgs.Parse(req));
        }

        [Fact]
        public void TransfersRequireGatherAndMovement()
        {
            var safety = new SafetyConfig();
            Assert.False(safety.Permits(TaskNames.Deposit));
            safety.AllowGather = true;
            Assert.True(safety.Permits(TaskNames.Deposit));
            Assert.True(safety.Permits(TaskNames.Withdraw));
            safety.AllowMovement = false;
            Assert.False(safety.Permits(TaskNames.Withdraw));
        }
    }
}
