using System.Collections.Generic;
using System.Linq;
using Newtonsoft.Json.Linq;
using Renki.Protocol;
using Renki.Tasks;
using Xunit;

namespace Renki.Tests
{
    public sealed class RecoverableWorkTests
    {
        sealed class Work : IRecoverableWork
        {
            public int Checks, Ticks, Resumes, Cancels;
            public bool CanSuspend = true;
            public Outcome Result, ResumeResult;
            public bool Checkpoint() { Checks++; return CanSuspend; }
            public Outcome Resume() { Resumes++; return ResumeResult; }
            public Outcome Tick(double dt) { Ticks++; return Result; }
            public void Cancel() { Cancels++; }
        }

        sealed class Encounter : IWorkEncounter
        {
            public bool Held = true;
            public Outcome Failure;
            public int Cancels;
            public bool Tick(double dt, bool canSuspend, out Outcome failure) { failure = Failure; return Held; }
            public void Cancel() { Cancels++; }
        }

        sealed class Host : ITaskHost
        {
            public IBodyTask Body;
            public IBodyTask StartBody(TaskRequest req, object args) => Body;
            public Outcome RunInstant(TaskRequest req, object args) => Outcome.Succeeded();
            public void Idle() { }
        }

        [Fact]
        public void RecoveryRetainsTheOriginalChildAndRevalidatesBeforeAnyFurtherUse()
        {
            var work = new Work(); var encounter = new Encounter();
            var task = new RecoverableWorkTask(work, encounter);
            for (var i = 0; i < 5; i++) Assert.Null(task.Tick(.02));
            Assert.Equal(1, work.Checks); Assert.Equal(0, work.Ticks); Assert.Equal(0, work.Resumes);
            encounter.Held = false;
            Assert.Null(task.Tick(.02));
            Assert.Equal(1, work.Resumes); Assert.Equal(1, work.Ticks);
            work.Result = Outcome.Succeeded("original source collected");
            Assert.Equal(Wire.StatusSucceeded, task.Tick(.02).Status);
            Assert.Equal(2, work.Ticks); Assert.Equal(1, work.Cancels); Assert.Equal(1, encounter.Cancels);
        }

        [Theory]
        [InlineData("resume_refused")]
        [InlineData("encounter_failed")]
        [InlineData("pending_use")]
        public void UncertainOrFailedRecoveryNeverReplaysACollection(string cause)
        {
            var work = new Work(); var encounter = new Encounter();
            var task = new RecoverableWorkTask(work, encounter);
            if (cause == "pending_use") work.CanSuspend = false;
            if (cause == "encounter_failed") encounter.Failure = Outcome.Failed(Wire.ReasonRefused, "no checked escape");
            if (cause == "resume_refused")
            {
                Assert.Null(task.Tick(.02));
                work.ResumeResult = Outcome.Failed(Wire.ReasonStopped, "source changed");
                encounter.Held = false;
            }
            var ended = task.Tick(.02);
            Assert.Equal(Wire.StatusFailed, ended.Status);
            Assert.Equal(Wire.ReasonStopped, ended.Reason);
            Assert.Equal(0, work.Ticks);
            task.Cancel(); task.Tick(.02);
            Assert.Equal(1, work.Cancels); Assert.Equal(1, encounter.Cancels);
        }

        [Fact]
        public void AnIssuedPickupCanFinishWithoutSuspensionWhenNoEncounterExists()
        {
            var work = new Work { CanSuspend = false, Result = Outcome.Succeeded("confirmed pickup") };
            var task = new RecoverableWorkTask(work, new Encounter { Held = false });
            Assert.Equal(Wire.StatusSucceeded, task.Tick(.02).Status);
            Assert.Equal(1, work.Ticks); Assert.Equal(0, work.Resumes);
        }

        [Theory]
        [InlineData("stable", true)]
        [InlineData("entity", false)]
        [InlineData("item", false)]
        [InlineData("quantity", false)]
        [InlineData("pack_gain", false)]
        [InlineData("pack_consumed", false)]
        [InlineData("gone", false)]
        public void ChangedSourceOrPackCannotAuthorizeARepeatedUse(string changed, bool expected)
        {
            var before = new WorkCheckpoint("1:2", "Raspberry", 3, 4, true);
            var after = new WorkCheckpoint(changed == "entity" ? "1:3" : "1:2", changed == "item" ? "Mushroom" : "Raspberry",
                changed == "quantity" ? 2 : 3, changed == "pack_gain" ? 7 : changed == "pack_consumed" ? 3 : 4, changed != "gone");
            Assert.Equal(expected, before.Matches(after));
        }

        [Theory]
        [InlineData("stop")]
        [InlineData("timeout")]
        [InlineData("reconnect")]
        [InlineData("death")]
        [InlineData("deadman")]
        public void EnclosingBodyLifecycleCancelsBothHeldChildren(string end)
        {
            var work = new Work(); var encounter = new Encounter();
            var host = new Host { Body = new RecoverableWorkTask(work, encounter) };
            var runner = new TaskRunner(new SafetyConfig { AllowGather = true }, TaskNames.Supported, host, _ => { }, () => 0);
            runner.Welcome(TaskNames.Supported);
            runner.Handle(new TaskRequest
            {
                Id = "collection",
                Task = "pickup",
                TimeoutS = 1,
                Args = JObject.Parse("{\"entity\":\"1:2\",\"recoverThreats\":true}")
            });
            runner.FixedTick(.02);
            Assert.Equal("collection", runner.Current.Id);
            if (end == "stop") runner.Handle(new TaskRequest { Id = "stop", Task = "stop" });
            if (end == "timeout") runner.FixedTick(2);
            if (end == "reconnect") runner.Welcome(TaskNames.Supported);
            if (end == "death") runner.Died();
            if (end == "deadman") runner.EndSession("quiet");
            Assert.Null(runner.Current); Assert.Equal(0, work.Ticks);
            Assert.Equal(1, work.Cancels); Assert.Equal(1, encounter.Cancels);
        }

        [Theory]
        [InlineData("pickup")]
        [InlineData("interact")]
        public void RecoveryFlagsRequireBothFleeAndAttackPermissions(string task)
        {
            foreach (var withheld in new[] { "flee", "attack", "config", "none" })
            {
                var host = new FakeHost(); var reports = new List<TaskStatus>();
                var runner = new TaskRunner(new SafetyConfig { AllowGather = true, AllowCombat = withheld != "config" },
                    TaskNames.Supported, host, reports.Add, () => 0);
                runner.Welcome(TaskNames.Supported.Where(t => t != withheld));
                runner.Handle(new TaskRequest
                {
                    Id = "work",
                    Task = task,
                    Args = JObject.Parse("{\"entity\":\"1:2\",\"recoverThreats\":true,\"defendThreats\":true}")
                });
                if (withheld == "none") Assert.Single(host.Bodies);
                else { Assert.Empty(host.Bodies); Assert.Equal(Wire.ReasonNotAllowed, Assert.Single(reports).Reason); }
            }
        }

        [Theory]
        [InlineData("\"hold\":true")]
        [InlineData("\"alt\":true")]
        [InlineData("\"sleep\":true")]
        [InlineData("\"repair\":true")]
        [InlineData("\"cook\":\"RawMeat\"")]
        [InlineData("\"recoverThreats\":\"yes\"")]
        public void RecoveryCannotWrapMultiUseOrInFlightStationWork(string mode) =>
            Assert.Throws<BadArgsException>(() => TaskArgs.Parse(new TaskRequest
            {
                Task = "interact",
                Args = JObject.Parse("{\"entity\":\"1:2\",\"recoverThreats\":true," + mode + "}")
            }));
    }
}
