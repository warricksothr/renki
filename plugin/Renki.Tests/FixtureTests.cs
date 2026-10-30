using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using Newtonsoft.Json.Linq;
using Renki.Protocol;
using Xunit;

namespace Renki.Tests
{
    /// <summary>The C# types against the fixtures the Go tests use
    /// (internal/protocol/testdata/README.md says what each directory means).</summary>
    public class FixtureTests
    {
        static readonly string Root = FindTestdata();

        static string FindTestdata()
        {
            for (var dir = new DirectoryInfo(AppContext.BaseDirectory); dir != null; dir = dir.Parent)
            {
                var candidate = Path.Combine(dir.FullName, "internal", "protocol", "testdata");
                if (Directory.Exists(candidate)) return candidate;
            }
            throw new DirectoryNotFoundException("internal/protocol/testdata not found above " + AppContext.BaseDirectory);
        }

        static Sender SenderOf(string dir) => dir == "plugin" ? Sender.Plugin : Sender.Controller;

        public static IEnumerable<object[]> Strict() =>
            from dir in new[] { "plugin", "controller" }
            from file in Directory.GetFiles(Path.Combine(Root, dir), "*.json").OrderBy(f => f)
            select new object[] { dir, Path.GetFileName(file) };

        public static IEnumerable<object[]> Lenient() =>
            from dir in new[] { "plugin", "controller" }
            from file in Directory.GetFiles(Path.Combine(Root, "lenient", dir), "*.json").OrderBy(f => f)
            select new object[] { dir, Path.GetFileName(file) };

        public static IEnumerable<object[]> Rejects() =>
            from dir in new[] { "plugin", "controller" }
            from file in Directory.GetFiles(Path.Combine(Root, "reject", dir), "*.json").OrderBy(f => f)
            select new object[] { dir, Path.GetFileName(file) };

        // Compares as values, as the Go test does: 3 and 3.0 are the same
        // number, so every integer becomes a double first.
        static JToken Numbers(JToken t)
        {
            switch (t)
            {
                case JObject o:
                    return new JObject(o.Properties().Select(p => new JProperty(p.Name, Numbers(p.Value))));
                case JArray a:
                    return new JArray(a.Select(Numbers));
                case JValue v when v.Type == JTokenType.Integer:
                    return new JValue((double)v);
                default:
                    return t;
            }
        }

        static void AssertSameJson(string want, string got)
        {
            var w = Numbers(JToken.Parse(want));
            var g = Numbers(JToken.Parse(got));
            Assert.True(JToken.DeepEquals(w, g),
                $"round trip changed the message\nwant {w.ToString(Newtonsoft.Json.Formatting.None)}\n got {got}");
        }

        [Theory]
        [MemberData(nameof(Strict))]
        public void RoundTrips(string dir, string file)
        {
            var frame = File.ReadAllText(Path.Combine(Root, dir, file));
            var msg = Codec.Decode(frame, SenderOf(dir));
            Assert.StartsWith(msg.MessageType, file);
            AssertSameJson(frame, Codec.Encode(msg));
        }

        [Theory]
        [MemberData(nameof(Lenient))]
        public void IgnoresUnknownFields(string dir, string file)
        {
            var msg = Codec.Decode(File.ReadAllText(Path.Combine(Root, "lenient", dir, file)), SenderOf(dir));
            AssertSameJson(File.ReadAllText(Path.Combine(Root, dir, file)), Codec.Encode(msg));
        }

        [Theory]
        [MemberData(nameof(Rejects))]
        public void Rejects_(string dir, string file)
        {
            var e = Assert.Throws<ProtocolException>(() =>
                Codec.Decode(File.ReadAllText(Path.Combine(Root, "reject", dir, file)), SenderOf(dir)));
            var want = file.StartsWith("version_") ? DecodeFailure.Version : DecodeFailure.UnknownType;
            Assert.Equal(want, e.Failure);
        }

        [Fact]
        public void TaskFixturesHaveValidArgs()
        {
            foreach (var file in Directory.GetFiles(Path.Combine(Root, "controller"), "task_*.json"))
            {
                var req = (TaskRequest)Codec.Decode(File.ReadAllText(file), Sender.Controller);
                Assert.NotNull(TaskArgs.Parse(req));
            }
        }

        [Fact]
        public void UnknownTypeKeepsTheFrameId()
        {
            var e = Assert.Throws<ProtocolException>(() =>
                Codec.Decode("{\"v\":1,\"type\":\"teleport\",\"id\":\"x-1\",\"seq\":1,\"t\":0}", Sender.Controller));
            Assert.Equal("x-1", e.FrameId);
        }
    }

    public class TaskArgsTests
    {
        static TaskRequest Req(string task, string args) =>
            new TaskRequest { Task = task, Args = args == null ? null : JObject.Parse(args) };

        [Theory]
        [InlineData("move_to", "{}")]
        [InlineData("move_to", "{\"to\":{\"x\":1}}")]
        [InlineData("move_to", "{\"to\":[1,2]}")]
        [InlineData("follow", "{\"distance\":3}")]
        [InlineData("follow", "{\"player\":\"Sothr\",\"run\":\"sometimes\"}")]
        [InlineData("look_at", "{}")]
        [InlineData("say", "{\"text\":\"\"}")]
        [InlineData("say", "{\"text\":\"hi\",\"channel\":\"whisper\"}")]
        [InlineData("say", "{\"text\":\"hi\",\"channel\":\"ping\"}")]
        [InlineData("emote", "{}")]
        [InlineData("harvest", "{}")]
        [InlineData("attack", "{}")]
        [InlineData("attack", "{\"entity\":\"   \"}")]
        [InlineData("attack", "{\"entity\":7}")]
        [InlineData("attack", "{\"entity\":\"1:2\",\"style\":\"ranged\"}")]
        [InlineData("attack", "{\"kite\":\"true\"}")]
        [InlineData("attack", "{\"entity\":\"1:2\",\"at\":[1,2,3]}")]
        [InlineData("attack", "{\"kite\":true,\"at\":[1,2]}")]
        [InlineData("attack", "{\"kite\":true,\"at\":[1,2,3,4]}")]
        [InlineData("attack", "{\"kite\":true,\"at\":\"1:2\"}")]
        [InlineData("attack", "{\"kite\":true,\"at\":null}")]
        [InlineData("attack", "{\"kite\":true,\"at\":[1,null,3]}")]
        [InlineData("attack", "{\"kite\":true,\"at\":[NaN,2,3]}")]
        [InlineData("attack", "{\"kite\":true,\"at\":[Infinity,2,3]}")]
        [InlineData("harvest", "{\"entity\":\"0:1\",\"collect\":\"yes\"}")]
        [InlineData("pickup", "{}")]
        [InlineData("pickup", "{\"entity\":7}")]
        [InlineData("drop", "{\"count\":3}")]
        [InlineData("drop", "{\"prefab\":\"Wood\"}")]
        [InlineData("drop", "{\"prefab\":\"Wood\",\"count\":0}")]
        [InlineData("drop", "{\"prefab\":\"Wood\",\"count\":2.5}")]
        [InlineData("interact", "{\"alt\":true}")]
        [InlineData("interact", "{\"entity\":\"0:1\",\"sleep\":true,\"hold\":true}")]
        [InlineData("interact", "{\"entity\":\"0:1\",\"sleep\":true,\"alt\":true}")]
        [InlineData("flee", "{\"distance\":20}")]
        [InlineData("use_item", "{}")]
        [InlineData("use_item", "{\"item\":7}")]
        [InlineData("equip", "{}")]
        [InlineData("unequip", "{\"slot\":\"tail\"}")]
        [InlineData("flee", "{\"from\":\"0:1\",\"distance\":0}")]
        [InlineData("flee", "{\"from\":\"0:1\",\"toward\":7}")]
        [InlineData("interact", "{\"entity\":\"0:1\",\"hold\":\"yes\"}")]
        public void BadArgs(string task, string args)
        {
            Assert.Throws<BadArgsException>(() => TaskArgs.Parse(Req(task, args)));
        }

        [Fact]
        public void Defaults()
        {
            var m = (MoveToArgs)TaskArgs.Parse(Req("move_to", "{\"to\":\"0:7\"}"));
            Assert.Equal("0:7", m.To.Entity);
            Assert.Equal(1.5, m.StopDistance);
            var f = (FollowArgs)TaskArgs.Parse(Req("follow", "{\"player\":\"Sothr\"}"));
            Assert.Equal(3.5, f.Distance);
            Assert.Equal(RunMode.Auto, f.Run);
            Assert.NotNull(TaskArgs.Parse(Req("stop", null)));
            Assert.Equal("0:7120", ((PickupArgs)TaskArgs.Parse(Req("pickup", "{\"entity\":\"0:7120\"}"))).Entity);
            var d = (DropArgs)TaskArgs.Parse(Req("drop", "{\"prefab\":\"Wood\",\"count\":3}"));
            Assert.Equal(("Wood", 3), (d.Prefab, d.Count));
            var ia = (InteractArgs)TaskArgs.Parse(Req("interact", "{\"entity\":\"1234:310\",\"alt\":true}"));
            Assert.Equal(("1234:310", true, false), (ia.Entity, ia.Alt, ia.Hold));
            var fl = (FleeArgs)TaskArgs.Parse(Req("flee", "{\"from\":\"0:5521\"}"));
            Assert.Equal(("0:5521", 20.0), (fl.From.Entity, fl.Distance));
            Assert.Null(fl.Toward);
            Assert.Equal("AxeStone", ((EquipArgs)TaskArgs.Parse(Req("equip", "{\"item\":\"AxeStone\"}"))).Item);
            Assert.Equal("hands", ((UnequipArgs)TaskArgs.Parse(Req("unequip", "{\"slot\":\"hands\"}"))).Slot);
            Assert.Equal("Raspberry", ((UseItemArgs)TaskArgs.Parse(Req("use_item", "{\"item\":\"Raspberry\"}"))).Item);
            Assert.True(TaskNames.IsInstant("use_item"));
            Assert.True(TaskNames.IsInstant("equip") && TaskNames.IsInstant("unequip"));
            Assert.True(((HarvestArgs)TaskArgs.Parse(Req("harvest", "{\"entity\":\"0:1\"}"))).Collect);
            Assert.False(((HarvestArgs)TaskArgs.Parse(Req("harvest", "{\"entity\":\"0:1\",\"collect\":false}"))).Collect);
        }
    }
}
