using System;
using System.Collections.Generic;
using System.Linq;
using Newtonsoft.Json.Linq;
using Renki.Movement;
using Renki.Perception;
using Renki.Protocol;
using Renki.Tasks;
using Xunit;

namespace Renki.Tests
{
    sealed class FakeBody : IBodyTask
    {
        public Outcome Next;
        public bool Cancelled;
        public int Ticks;

        public Outcome Tick(double dt)
        {
            Ticks++;
            return Next;
        }

        public void Cancel() => Cancelled = true;
    }

    sealed class FakeHost : ITaskHost
    {
        public readonly List<FakeBody> Bodies = new List<FakeBody>();
        public readonly List<string> Instants = new List<string>();
        public int Idles;
        public TaskRefused Refuse;

        public IBodyTask StartBody(TaskRequest req, object args)
        {
            if (Refuse != null) throw Refuse;
            var b = new FakeBody();
            Bodies.Add(b);
            return b;
        }

        public Outcome RunInstant(TaskRequest req, object args)
        {
            Instants.Add(req.Task + (args is SayArgs s ? ":" + s.Text : ""));
            return Outcome.Succeeded();
        }

        public void Idle() => Idles++;
    }

    public class TaskRunnerTests
    {
        readonly List<TaskStatus> reports = new List<TaskStatus>();
        readonly FakeHost host = new FakeHost();
        double clock;
        readonly SafetyConfig safety = new SafetyConfig();
        readonly TaskRunner runner;

        public TaskRunnerTests()
        {
            runner = new TaskRunner(safety, TaskNames.M1, host, reports.Add, () => clock);
        }

        static TaskRequest Req(string id, string task, string args = null, double timeout = 0) =>
            new TaskRequest { Id = id, Task = task, Args = args == null ? null : JObject.Parse(args), TimeoutS = timeout };

        string Log() => string.Join(" ", reports.Select(r => $"{r.Re}:{r.Status}{(r.Reason != null ? "/" + r.Reason : "")}"));

        [Fact]
        public void NothingRunsBeforeWelcome()
        {
            runner.Handle(Req("t-1", "follow", "{\"player\":\"Sothr\"}"));
            Assert.Equal("t-1:rejected/not_allowed", Log());
        }

        [Fact]
        public void BodyLifecycle()
        {
            runner.Welcome(TaskNames.M1);
            runner.Handle(Req("t-1", "follow", "{\"player\":\"Sothr\"}"));
            runner.FixedTick(0.02);
            Assert.Equal("t-1", runner.Current.Id);
            host.Bodies[0].Next = Outcome.Failed(Wire.ReasonTargetLost);
            runner.FixedTick(0.02);
            Assert.Equal("t-1:accepted t-1:running t-1:failed/target_lost", Log());
            Assert.Null(runner.Current);
        }

        [Fact]
        public void NewBodyTaskSupersedes()
        {
            runner.Welcome(TaskNames.M1);
            runner.Handle(Req("t-1", "follow", "{\"player\":\"Sothr\"}"));
            runner.Handle(Req("t-2", "move_to", "{\"to\":[1,2,3]}"));
            Assert.True(host.Bodies[0].Cancelled);
            Assert.Equal("t-1:accepted t-1:running t-1:cancelled/superseded t-2:accepted t-2:running", Log());
        }

        [Fact]
        public void InstantTasksRunAlongside()
        {
            runner.Welcome(TaskNames.M1);
            runner.Handle(Req("t-1", "follow", "{\"player\":\"Sothr\"}"));
            runner.Handle(Req("t-2", "emote", "{\"name\":\"wave\"}"));
            Assert.False(host.Bodies[0].Cancelled);
            Assert.Equal("t-1", runner.Current.Id);
            Assert.Contains("t-2:succeeded", Log());
        }

        [Fact]
        public void StopCancelsEverything()
        {
            runner.Welcome(TaskNames.M1);
            runner.Handle(Req("t-1", "follow", "{\"player\":\"Sothr\"}"));
            runner.Handle(Req("t-2", "stop"));
            Assert.True(host.Bodies[0].Cancelled);
            Assert.EndsWith("t-1:cancelled/stopped t-2:succeeded", Log());
        }

        [Fact]
        public void Timeout()
        {
            runner.Welcome(TaskNames.M1);
            runner.Handle(Req("t-1", "move_to", "{\"to\":\"0:1\"}", timeout: 1));
            for (var i = 0; i < 60; i++) runner.FixedTick(0.02);
            Assert.EndsWith("t-1:failed/timeout", Log());
            Assert.True(host.Bodies[0].Cancelled);
        }

        [Fact]
        public void RejectionReasons()
        {
            safety.AllowChat = false;
            runner.Welcome(new[] { "stop", "follow", "say", "emote", "harvest" });
            runner.Handle(Req("t-1", "harvest", "{\"entity\":\"0:1\"}"));
            runner.Handle(Req("t-2", "say", "{\"text\":\"hi\"}"));
            runner.Handle(Req("t-3", "move_to", "{\"to\":[0,0,0]}"));
            runner.Handle(Req("t-4", "follow", "{}"));
            host.Refuse = new TaskRefused(Wire.ReasonInvalidTarget, "no such player");
            runner.Handle(Req("t-5", "follow", "{\"player\":\"Nobody\"}"));
            Assert.Equal("t-1:rejected/unsupported t-2:rejected/not_allowed t-3:rejected/not_allowed " +
                         "t-4:rejected/bad_args t-5:rejected/invalid_target", Log());
        }

        [Fact]
        public void WelcomeOnlyNarrows()
        {
            safety.AllowMovement = false;
            runner.Welcome(TaskNames.M1);
            Assert.DoesNotContain("follow", runner.Allowed);
            Assert.Contains("stop", runner.Allowed);
        }

        [Fact]
        public void DeadManStopsAndNothingResumes()
        {
            runner.Welcome(TaskNames.M1);
            runner.Handle(Req("t-1", "follow", "{\"player\":\"Sothr\"}"));
            runner.EndSession("no controller message for 10 s");
            Assert.True(host.Bodies[0].Cancelled);
            Assert.Null(runner.Current);
            Assert.False(runner.InSession);
            runner.FixedTick(0.02);
            Assert.Equal(0, host.Bodies[0].Ticks);
            runner.Handle(Req("t-2", "say", "{\"text\":\"hi\"}"));
            Assert.EndsWith("t-2:rejected/not_allowed", Log());
            runner.Welcome(TaskNames.M1);
            Assert.Null(runner.Current);
        }

        [Fact]
        public void SayIsCleanedAndRateLimited()
        {
            runner.Welcome(TaskNames.M1);
            runner.Handle(Req("t-1", "say", "{\"text\":\"/kill all\\nnow\"}"));
            Assert.Equal("say:kill all now", host.Instants[0]);
            runner.Handle(Req("t-2", "say", "{\"text\":\"two\"}"));
            runner.Handle(Req("t-3", "say", "{\"text\":\"three\"}"));
            runner.Handle(Req("t-4", "say", "{\"text\":\"four\"}"));
            Assert.EndsWith("t-4:rejected/not_allowed", Log());
            clock = 2.1;
            runner.Handle(Req("t-5", "say", "{\"text\":\"five\"}"));
            Assert.EndsWith("t-5:succeeded", Log());
            runner.Handle(Req("t-6", "say", "{\"text\":\"///\"}"));
            Assert.EndsWith("t-6:rejected/bad_args", Log());
        }

        [Fact]
        public void DyingFailsTheBodyTask()
        {
            runner.Welcome(TaskNames.M1);
            runner.Handle(Req("t-1", "follow", "{\"player\":\"Sothr\"}"));
            runner.Died();
            Assert.EndsWith("t-1:failed/died", Log());
        }
    }

    public class MovementTests
    {
        [Fact]
        public void StuckSequenceThenGiveUp()
        {
            var d = new StuckDetector();
            var pos = new V3(0, 0, 0);
            var seen = new List<Recovery>();
            for (var t = 0.0; t < 40 && !seen.Contains(Recovery.GiveUp); t += 0.1)
            {
                var r = d.Update(pos, t);
                if (r != Recovery.None) seen.Add(r);
            }
            Assert.Equal(new[]
            {
                Recovery.Jump, Recovery.Strafe, Recovery.Repath, Recovery.Climb, Recovery.Strafe, Recovery.Repath,
                Recovery.Climb, Recovery.Strafe, Recovery.Repath, Recovery.GiveUp,
            }, seen);
        }

        [Fact]
        public void TheStraightLineStopsAtDeepWater()
        {
            // A lake between x 20 and 60, 5 m deep; the water level is 30.
            double Ground(double x, double z) => x > 20 && x < 60 ? 25 : 34;
            var crossing = DeepWater.Crossing(new V3(0, 34, 0), new V3(100, 34, 0), Ground, 30);
            Assert.NotNull(crossing);
            Assert.InRange(crossing.Value.X, 20, 24);
            Assert.Null(DeepWater.Crossing(new V3(0, 34, 0), new V3(18, 34, 0), Ground, 30));
            // Wading depth is fine.
            Assert.Null(DeepWater.Crossing(new V3(0, 34, 0), new V3(100, 34, 0), (x, z) => 29, 30));
            // Already swimming: the way out is open, a way back in is not.
            Assert.Null(DeepWater.Crossing(new V3(40, 30, 0), new V3(0, 34, 0), Ground, 30));
            Assert.NotNull(DeepWater.Crossing(new V3(40, 30, 0), new V3(0, 34, 0), (x, z) => x > 20 ? 25 : x > 10 ? 34 : 25, 30));
            // Ground not loaded is not water.
            Assert.Null(DeepWater.Crossing(new V3(0, 34, 0), new V3(100, 34, 0), (x, z) => x > 50 ? double.NaN : 34, 30));
        }

        [Fact]
        public void TheShoreIsTheNearestDryGround()
        {
            // Swimming at x 40 in the lake: the west shore (x 20) is nearer.
            double Ground(double x, double z) => x > 20 && x < 70 ? 25 : 34;
            var shore = DeepWater.Shore(new V3(40, 30, 0), Ground, 30);
            Assert.NotNull(shore);
            Assert.True(shore.Value.X <= 20, $"shore at {shore.Value.X}");
            Assert.Null(DeepWater.Shore(new V3(0, 30, 0), (x, z) => 0, 30));
            // A shore it could not get onto is passed over for the next one.
            var next = DeepWater.Shore(new V3(40, 30, 0), Ground, 30, tried: new[] { shore.Value });
            Assert.NotNull(next);
            Assert.True(V3.FlatDistance(next.Value, shore.Value) >= DeepWater.Tried, $"next shore at {next.Value.X}, {next.Value.Z}");
        }

        [Fact]
        public void ArrivalIsOnTheGoalsLevel()
        {
            var home = new V3(-35.9, 36.09, 6.53);
            Assert.True(Arrival.Reached(new V3(-37.0, 36.09, 6.5), home, 2));
            // Seen on the VM: under the raised floor, 1.9 m across and 2 m down.
            Assert.False(Arrival.Reached(new V3(-37.76, 34.0, 6.36), home, 2));
            // A storey up is not there either; a step up is.
            Assert.False(Arrival.Reached(new V3(-35.9, 38.09, 6.53), home, 2));
            Assert.True(Arrival.Reached(new V3(-35.9, 37.0, 6.53), home, 2));
        }

        [Fact]
        public void WallsAreTallThinAndLong()
        {
            // wood_wall: 2 long, 0.2 thick, 2 tall, centred on its origin.
            var w = WallShape.Of(new V3(-1, 0, -0.1), new V3(1, 2, 0.1));
            Assert.NotNull(w);
            Assert.Equal(new V3(-1, 0, 0), w.Value.A);
            Assert.Equal(new V3(1, 0, 0), w.Value.B);
            Assert.Equal(2, w.Value.H);
            // Running along z.
            Assert.Equal(new V3(0, 0, 1), WallShape.Of(new V3(-0.1, 0, -1), new V3(0.1, 1.5, 1)).Value.B);
            Assert.NotNull(WallShape.Of(new V3(-2, 0, -0.5), new V3(2, 2, 0.5)));  // stone, 1 thick
            Assert.Null(WallShape.Of(new V3(-1, 0, -1), new V3(1, 0.2, 1)));     // a floor
            Assert.Null(WallShape.Of(new V3(-1, 0, -0.1), new V3(1, 1, 0.1)));   // a half wall
            Assert.Null(WallShape.Of(new V3(-1, 0, -0.5), new V3(0.5, 2, 0.5))); // a stone block
            Assert.Null(WallShape.Of(new V3(-0.1, 0, -0.1), new V3(0.1, 2, 0.1))); // a post
            Assert.Null(WallShape.Of(new V3(-1, 0, -1), new V3(1, 2, 1)));       // a roof, or a block
        }

        [Fact]
        public void HoldKeepsTheCountWhileARecoveryTakesItsTime()
        {
            var d = new StuckDetector();
            d.Update(new V3(0, 0, 0), 0);
            Assert.Equal(Recovery.Jump, d.Update(new V3(0, 0, 0), 3.1));
            for (var t = 3.2; t < 12; t += 0.1) d.Hold(t); // a climb, waiting
            Assert.Equal(Recovery.None, d.Update(new V3(0, 0, 0), 12));
            Assert.Equal(Recovery.Strafe, d.Update(new V3(0, 0, 0), 15.1));
            Assert.Equal(2, d.Attempts);
        }

        [Fact]
        public void ClimbWaitsForStaminaThenJumpsInARow()
        {
            var c = new Climber();
            c.Start(0);
            // 10 a jump: 31 needed. Waits while it comes back.
            Assert.Equal(ClimbStep.Wait, c.Tick(0, 5, 50, 10, true));
            Assert.Equal(ClimbStep.Wait, c.Tick(2, 20, 50, 10, true));
            Assert.Equal(ClimbStep.Jump, c.Tick(4, 31, 50, 10, true));
            Assert.Equal(ClimbStep.Push, c.Tick(4.5, 21, 50, 10, true));
            Assert.Equal(ClimbStep.Jump, c.Tick(4.8, 21, 50, 10, true));
            Assert.Equal(ClimbStep.Jump, c.Tick(5.6, 11, 50, 10, true));
            Assert.Equal(ClimbStep.Push, c.Tick(6, 1, 50, 10, true));
            Assert.Equal(ClimbStep.Done, c.Tick(6.4, 1, 50, 10, true));
            Assert.False(c.Active);
        }

        [Fact]
        public void ClimbDoesNotWaitForStaminaThatCannotComeBack()
        {
            var c = new Climber();
            c.Start(0);
            Assert.Equal(ClimbStep.Jump, c.Tick(0, 12, 50, 10, false)); // overloaded: no regen
            Assert.Equal(ClimbStep.Done, c.Tick(0.8, 2, 50, 10, false)); // and no more to spend
            c.Start(10);
            Assert.Equal(ClimbStep.Wait, c.Tick(15, 5, 50, 10, true));
            Assert.Equal(ClimbStep.Done, c.Tick(20, 5, 50, 10, true)); // waited long enough; none to jump with
            // A small bar still waits only for what it holds.
            Assert.Equal(19, new Climber().Needed(10, 20));
        }

        [Fact]
        public void ProgressResetsStuck()
        {
            var d = new StuckDetector();
            var x = 0.0;
            for (var t = 0.0; t < 30; t += 0.1)
            {
                x += 0.15; // 1.5 m/s, a slow walk
                Assert.Equal(Recovery.None, d.Update(new V3(x, 0, 0), t));
            }
        }

        [Fact]
        public void ShufflingAgainstAWallIsStillStuck()
        {
            // Pressed on a wall, strafing up to 2 m either way and back: lots
            // of movement, but never out of the circle.
            var d = new StuckDetector();
            var seen = new List<Recovery>();
            for (var t = 0.0; t < 40 && !seen.Contains(Recovery.GiveUp); t += 0.1)
            {
                var r = d.Update(new V3(2 * Math.Sin(t), 0, 0), t);
                if (r != Recovery.None) seen.Add(r);
            }
            Assert.Equal(10, seen.Count);
            Assert.Equal(Recovery.GiveUp, seen[9]);
        }

        [Fact]
        public void LeavingTheCircleStartsANewOne()
        {
            var d = new StuckDetector();
            var t = 0.0;
            for (; t < 3.05; t += 0.1) d.Update(new V3(0, 0, 0), t);
            Assert.Equal(Recovery.Strafe, d.Update(new V3(0, 0, 0), 6.1));
            Assert.Equal(Recovery.None, d.Update(new V3(3.5, 0, 0), 6.2)); // out: a new circle
            Assert.Equal(Recovery.None, d.Update(new V3(3.5, 0, 0), 9.1));
            Assert.Equal(Recovery.Repath, d.Update(new V3(3.5, 0, 0), 9.3)); // but still stuck
        }

        [Fact]
        public void GettingAwayStartsAgain()
        {
            var d = new StuckDetector();
            var t = 0.0;
            for (; t < 3.05; t += 0.1) d.Update(new V3(0, 0, 0), t);
            Assert.Equal(Recovery.Strafe, d.Update(new V3(0, 0, 0), 6.1));
            Assert.Equal(Recovery.None, d.Update(new V3(6.5, 0, 0), 6.2)); // past Escape
            Assert.Equal(Recovery.Jump, d.Update(new V3(6.5, 0, 0), 9.3));
        }

        [Fact]
        public void StrafingOverTheEdgeAndBackIsStillStuck()
        {
            // Seen on a slope in the game: every strafe carried the bot about
            // 3 m sideways, just out of the circle, and the next one back.
            var d = new StuckDetector();
            var seen = new List<Recovery>();
            var x = 0.0;
            for (var t = 0.0; t < 60 && !seen.Contains(Recovery.GiveUp); t += 0.1)
            {
                var r = d.Update(new V3(x, 0, 0), t);
                if (r == Recovery.None) continue;
                seen.Add(r);
                if (r == Recovery.Strafe) x = x == 0 ? 3.2 : 0;
            }
            Assert.Equal(Recovery.GiveUp, seen[seen.Count - 1]);
            Assert.Contains(Recovery.Repath, seen);
        }

        [Fact]
        public void NearestKeepsTheClosestFirst()
        {
            var n = new Nearest<string>(2);
            n.Add("far", 30);
            n.Add("near", 2);
            n.Add("middle", 10);
            Assert.Equal(3, n.Seen);
            Assert.Equal(new[] { "near", "middle" }, n.Take());
            n.Clear();
            Assert.Empty(n.Take());
            n.Cap = 0;
            n.Add("any", 1);
            Assert.Empty(n.Take());
        }

        [Fact]
        public void EachKindKeepsItsOwnNearest()
        {
            var n = new NearestByKind<string>(1);
            n.Add("tree", "near tree", 1);
            n.Add("tree", "far tree", 3);
            n.Add("drop", "wood", 20);
            Assert.Equal(new[] { "near tree", "wood" }, n.Take());
            Assert.Equal(3, n.Seen);
            n.Clear();
            Assert.Empty(n.Take());
        }

        [Fact]
        public void ADoorAheadAndWithinReachIsOpened()
        {
            var at = new V3(0, 0, 0);
            var north = new V3(0, 0, 1);
            Assert.True(DoorRules.Ahead(at, north, new V3(0.5, 0, 2)));
            Assert.False(DoorRules.Ahead(at, north, new V3(0, 0, -2)));   // behind
            Assert.False(DoorRules.Ahead(at, north, new V3(2.4, 0, 0)));  // beside
            Assert.False(DoorRules.Ahead(at, north, new V3(0, 0, 3)));    // out of reach
            Assert.True(DoorRules.Ahead(at, V3.Zero, new V3(0.2, 0, 0))); // in the doorway
            Assert.False(DoorRules.Past(new V3(0, 0, 3), new V3(0, 0, 0)));
            Assert.True(DoorRules.Past(new V3(0, 0, 4), new V3(0, 0, 0)));
            Assert.True(DoorRules.In(new V3(0, 0, 1), new V3(0, 0, 0)));
            Assert.False(DoorRules.In(new V3(0, 0, 2), new V3(0, 0, 0)));
            Assert.False(DoorRules.In(new V3(0, 3, 0.5), new V3(0, 0, 0))); // the floor above
        }

        [Fact]
        public void GatheringNeedsItsSwitchAndMovement()
        {
            var s = new SafetyConfig();
            Assert.False(s.Permits("pickup"));
            s.AllowGather = true;
            Assert.True(s.Permits("pickup"));
            s.AllowMovement = false;
            Assert.False(s.Permits("pickup"));
            Assert.False(s.Permits("harvest"));
        }

        [Fact]
        public void DroppingHasItsOwnSwitch()
        {
            var s = new SafetyConfig { AllowMovement = false };
            Assert.True(s.Permits("drop"));
            s.AllowDrop = false;
            Assert.False(s.Permits("drop"));
            Assert.True(TaskNames.IsInstant("drop"));
        }

        [Fact]
        public void PathFollowerAdvances()
        {
            var p = new PathFollower();
            p.Set(new[] { new V3(0, 0, 5), new V3(5, 0, 5), new V3(5, 0, 10) });
            Assert.Equal(new V3(0, 0, 5), p.Next(new V3(0, 0, 0)));
            Assert.Equal(new V3(5, 0, 5), p.Next(new V3(0, 0, 4.5)));
            Assert.Equal(new V3(5, 0, 10), p.Next(new V3(4.8, 0, 5.1)));
            Assert.Null(p.Next(new V3(5, 0, 9.8)));
        }

        [Fact]
        public void FollowPaceHysteresis()
        {
            var f = new FollowPace();
            Assert.Equal(Gait.Stop, f.Decide(3, 3.5, RunMode.Auto, false));
            Assert.Equal(Gait.Stop, f.Decide(4.2, 3.5, RunMode.Auto, false)); // inside the slack while stopped
            Assert.Equal(Gait.Walk, f.Decide(5, 3.5, RunMode.Auto, false));
            Assert.Equal(Gait.Walk, f.Decide(4.0, 3.5, RunMode.Auto, false)); // moving: walks until inside the distance
            Assert.Equal(Gait.Run, f.Decide(8, 3.5, RunMode.Auto, false));
            Assert.Equal(Gait.Run, f.Decide(5, 3.5, RunMode.Auto, true));
            Assert.Equal(Gait.Walk, f.Decide(8, 3.5, RunMode.Walk, true));
            Assert.Equal(Gait.Stop, f.Decide(3.4, 3.5, RunMode.Auto, false));
        }

        [Fact]
        public void LocalControls()
        {
            // Looking north, moving east is pure right.
            var l = Controls.ToLocal(new V3(1, 0, 0), new V3(0, 0, 1));
            Assert.Equal(1, l.X, 6);
            Assert.Equal(0, l.Z, 6);
            // Looking east, moving east is pure forward.
            l = Controls.ToLocal(new V3(1, 0, 0), new V3(1, 0, 0));
            Assert.Equal(0, l.X, 6);
            Assert.Equal(1, l.Z, 6);
            // Looking east, moving north is left.
            l = Controls.ToLocal(new V3(0, 0, 1), new V3(1, 0, 0));
            Assert.Equal(-1, l.X, 6);
            Assert.Equal(90, Controls.Yaw(new V3(1, 0, 0)), 6);
            Assert.Equal(270, Controls.Yaw(new V3(-1, 0, 0)), 6);
        }
    }
}
