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
