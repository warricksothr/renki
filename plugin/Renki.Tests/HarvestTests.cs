using Renki.Tasks;
using Renki.Protocol;
using Newtonsoft.Json.Linq;
using System.Collections.Generic;
using Renki.Movement;
using Xunit;

namespace Renki.Tests
{
    public sealed class HarvestTests
    {
        [Fact]
        public void FoodRefreshDoesNotInventDamageOrClearTheDamageLatch()
        {
            Assert.False(HarvestHealth.Interrupted(false, 60.7, 60.7, 60, 66.1, true));
            Assert.True(HarvestHealth.Interrupted(true, 60.7, 60.7, 60, 66.1, true));
            Assert.True(HarvestHealth.Interrupted(true, 60.7, 60.7, 66.1, 66.1, true));
            Assert.True(HarvestHealth.Interrupted(false, 60.7, 60.7, 60, 66.1, false));
            Assert.False(HarvestHealth.Interrupted(false, 60.7, 60.7, 60, 60, false));
        }

        [Theory]
        [InlineData(false, false, false, false)]
        [InlineData(false, true, false, false)]
        [InlineData(false, false, true, false)]
        [InlineData(false, true, true, true)]
        [InlineData(false, null, false, true)]
        [InlineData(false, false, null, true)]
        [InlineData(true, false, false, true)]
        public void PlayerProximityUsesKnownPvpOnlyForOrdinaryStrikes(bool hazard, bool? attacker, bool? target, bool blocks)
        {
            Assert.Equal(blocks, HarvestSafety.PlayerBlocks(hazard, attacker, target));
        }

        [Fact]
        public void OutdoorApproachDoesNotInheritRaisedStartingFloor()
        {
            var resource = new V3(-25.49, 32.57, 32.54);
            var contact = new V3(-25.8, 33.17, 32.2);
            var fromHouse = new V3(-37.45, 36.09, 7.59);
            var goal = HarvestReach.StandAt(resource, contact, fromHouse, .65);
            Assert.Equal(resource.Y, goal.Y);
            Assert.InRange(V3.FlatDistance(goal, contact), .649, .651);
            Assert.False(HarvestReach.InRange(fromHouse, contact, .6, 1));
            Assert.True(HarvestReach.InRange(goal, contact, .6, 1));
            Assert.Equal(resource.Y + 2, HarvestReach.GroundProbeTop(resource.Y, contact.Y));
        }

        [Fact]
        public void AHighLogPivotDoesNotExcludeGroundBelowALowerMeasuredContact()
        {
            var pivot = new V3(168.02, 90.96, -281.97);
            var feet = new V3(169.27, 83.51, -273.58);
            var lowContactY = feet.Y + .8;
            var top = HarvestReach.GroundProbeTop(pivot.Y, lowContactY);
            Assert.True(pivot.Y + 2 - 8 > feet.Y); // Previous ray misses this floor.
            Assert.InRange(feet.Y, top - 8, top);
            Assert.Equal(lowContactY + 2, top);
            // An actual high contact remains out of axe range from this floor.
            Assert.False(HarvestReach.InRange(feet, new V3(feet.X, pivot.Y, feet.Z), .8, 1.4));
        }

        [Fact]
        public void BroadBoundsAreNotStrikeContactAndCloserPositionIsInReach()
        {
            var missed = new V3(-27.5, 32.57, 30.86);
            var trunkContact = new V3(-25.8, 33.17, 32.2);
            Assert.False(HarvestReach.InRange(missed, trunkContact, .6, 1));
            var closer = HarvestReach.StandAt(new V3(-25.49, 32.57, 32.54), trunkContact, missed, .65);
            Assert.True(HarvestReach.InRange(closer, trunkContact, .6, 1));
            Assert.False(HarvestReach.InRange(closer + new V3(0, 3.5, 0), trunkContact, .6, 1));
        }

        [Fact]
        public void StumpStrikeRangeDoesNotRequireClimbingItsBase()
        {
            var feet = new V3(-30.52, 32.48, 32.22);
            var contact = new V3(-30.22, 33.01, 33.2);
            var reach = HarvestReach.Range(2.2); // Observed flint axe.
            Assert.True(HarvestReach.InRange(feet, contact, .8, reach));
            var standOff = HarvestReach.StandOff(reach, false);
            Assert.True(standOff > .65);
            Assert.True(standOff + .15 < reach); // Arrival margin.
            Assert.True(HarvestReach.StandOff(reach, true) < standOff);
            var shortReach = HarvestReach.Range(.8);
            Assert.True(HarvestReach.StandOff(shortReach, false) < shortReach);
            // Raising the feet onto the resource must not be necessary.
            Assert.False(HarvestReach.InRange(feet + new V3(0, 3, 0), contact, .8, reach));
        }

        [Fact]
        public void FoodMaximumDecayIsNotDamage()
        {
            Assert.False(HarvestHealth.Hurt(115, 114.7, 114.7));
            Assert.True(HarvestHealth.Hurt(115, 113.7, 114.7));
            Assert.True(HarvestHealth.Hurt(115, 114.7, 115));
        }

        [Fact]
        public void FullBeechUsesMeasuredTrunkContactRatherThanOverheadMeshCenter()
        {
            var feet = new V3(175.83, 84.45, -249.43);
            var contact = new V3(176.21, 84.97, -250.22);
            var boundsCenter = new V3(176.24, 101.99, -249.88);
            var hits = new[] { new SightHit(.1, true, false), new SightHit(.92, false, true) };
            Assert.False(HarvestReach.InRange(feet, boundsCenter, .8, 2.5));
            Assert.True(HarvestReach.CanStrike(feet, contact, .8, 1.4, hits));
            Assert.False(HarvestReach.CanStrike(feet + new V3(0, 3, 0), contact, .8, 1.4, hits));
            Assert.False(HarvestReach.CanStrike(feet, contact, .8, .5, hits));
        }

        [Fact]
        public void PhysicalStrikeContactStillNeedsTheTargetBeforeAnyOtherSolid()
        {
            var feet = V3.Zero; var contact = new V3(1, .8, 0);
            Assert.False(HarvestReach.CanStrike(feet, contact, .8, 1.4, new[]
                { new SightHit(1, false, true), new SightHit(.5, false, false) }));
            Assert.False(HarvestReach.CanStrike(feet, contact, .8, 1.4, new SightHit[0]));
            Assert.False(HarvestReach.CanStrike(feet, contact, .8, 1.4, new[] { new SightHit(.1, true, false) }));
            Assert.True(HarvestReach.CanStrike(feet, contact, .8, 1.4, new[]
                { new SightHit(1.3, false, false), new SightHit(1, false, true) }));
        }

        [Fact]
        public void HarvestWaitsForRecoveryAndCancelsOnTheFirstHealthLoss()
        {
            Assert.True(HarvestHealth.Ready(25, 25));
            Assert.True(HarvestHealth.Ready(20, 25));
            Assert.False(HarvestHealth.Ready(19.9, 25));
            Assert.False(HarvestHealth.Ready(25, 0));
            Assert.False(HarvestHealth.Hurt(25, 25, 25));
            Assert.True(HarvestHealth.Hurt(25, 24.9, 25));
            Assert.True(HarvestHealth.Hurt(25, 20.4, 25));
        }

        [Fact]
        public void RemoteLogsNeedQuietPosesEvenWhenLocalVelocityIsZero()
        {
            var quiet = new HarvestStillness();
            var up = new V3(0, 1, 0); var forward = new V3(0, 0, 1);
            Assert.False(quiet.Observe(V3.Zero, up, forward, false, 0));
            Assert.False(quiet.Observe(V3.Zero, up, forward, false, 1));
            Assert.True(quiet.Observe(V3.Zero, up, forward, false, 2));
            // Individually small network movements accumulate relative to
            // the quiet anchor; resetting the anchor every frame misses them.
            quiet.Observe(new V3(.01, 0, 0), up, forward, false, 2.1);
            Assert.False(quiet.Observe(new V3(.03, 0, 0), up, forward, false, 2.2));
            Assert.False(quiet.Observe(new V3(.03, 0, 0), up, forward, false, 4));
            Assert.True(quiet.Observe(new V3(.03, 0, 0), up, forward, false, 4.3));
            Assert.False(quiet.Observe(new V3(.03, 0, 0), up, new V3(.02, 0, .9998), false, 4.4));
            Assert.False(quiet.Observe(new V3(.03, 0, 0), up, new V3(.02, 0, .9998), true, 7));
        }

        [Fact]
        public void AlternativeTrunkEscapesNeverInitiallyApproachOrCrossTheResource()
        {
            var resource = new V3(175.46, 86.65, -265.53);
            var feet = new V3(175.83, 86, -263.24);
            var goals = new HashSet<V3>();
            for (var attempt = 0; attempt < HarvestEscape.CandidateLimit; attempt++)
            {
                var goal = HarvestEscape.Candidate(resource, feet, attempt);
                Assert.True(goals.Add(goal));
                Assert.InRange((goal - feet).Length, 5.999, 6.001);
                Assert.Equal(feet.Y, goal.Y);
                var previous = V3.FlatDistance(feet, resource);
                for (var step = 1; step <= 12; step++)
                {
                    var distance = V3.FlatDistance(feet + (goal - feet) * (step / 12.0), resource);
                    Assert.True(distance >= previous - .00001);
                    previous = distance;
                }
                Assert.True(previous >= 6);
            }
            Assert.Throws<System.ArgumentOutOfRangeException>(() => HarvestEscape.Candidate(resource, feet, 7));
            Assert.Throws<System.ArgumentException>(() => HarvestEscape.Candidate(resource, resource, 0));
        }

        sealed class Tool
        {
            public int Tier;
            public double Chop;
            public bool Usable = true;
        }
        static Tool Choose(int minimum, params Tool[] tools) => HarvestTool.Best(tools, minimum, t => t.Tier, t => t.Chop, t => t.Usable);

        [Fact]
        public void ToolMustBeUsableAndAbleToDamageTheResource()
        {
            var sword = new Tool { Tier = 4 };
            var broken = new Tool { Tier = 3, Chop = 100, Usable = false };
            var stone = new Tool { Tier = 0, Chop = 20 };
            var flint = new Tool { Tier = 1, Chop = 30 };
            Assert.Same(flint, Choose(0, sword, broken, stone, flint));
            Assert.Null(Choose(2, sword, broken, stone, flint));
            Assert.Null(Choose(0, sword, broken));
            // Upgraded lower-tier equipment can do more damage; min tier
            // is an eligibility check, not a reason to waste a slower tool.
            stone.Chop = 40;
            Assert.Same(stone, Choose(0, stone, flint));
            Assert.Same(flint, Choose(1, stone, flint));
        }

        [Fact]
        public void AbsenceOrCompetingDamageDoesNotClaimHarvestSuccess()
        {
            var evidence = new HarvestEvidence();
            Assert.False(evidence.Broken(10));
            evidence.Hit(false, 10);
            Assert.False(evidence.Broken(10));
            evidence.Hit(true, 11);
            Assert.True(evidence.Broken(11.5));
            Assert.False(evidence.Broken(14));
            evidence.Hit(false, 11.6);
            Assert.False(evidence.Broken(11.7));
            evidence.Hit(true, 12);
            Assert.True(evidence.Broken(12.1));
        }

        [Fact]
        public void HarvestNeedsGatherAndMovementAndStopCancelsItsBody()
        {
            var safety = new SafetyConfig();
            var host = new FakeHost();
            var reports = new List<TaskStatus>();
            var runner = new TaskRunner(safety, TaskNames.Supported, host, reports.Add, () => 0);
            var request = new TaskRequest { Id = "harvest", Task = TaskNames.Harvest, Args = JObject.Parse("{\"entity\":\"1:2\",\"collect\":false}") };
            runner.Welcome(TaskNames.Supported);
            runner.Handle(request);
            Assert.Equal(Wire.ReasonNotAllowed, reports[reports.Count - 1].Reason);
            safety.AllowGather = true; safety.AllowMovement = false;
            runner.Welcome(TaskNames.Supported);
            runner.Handle(request);
            Assert.Equal(Wire.ReasonNotAllowed, reports[reports.Count - 1].Reason);
            safety.AllowMovement = true;
            runner.Welcome(TaskNames.Supported);
            runner.Handle(request);
            Assert.Equal(Wire.StatusRunning, reports[reports.Count - 1].Status);
            runner.Handle(new TaskRequest { Id = "stop", Task = TaskNames.Stop });
            Assert.True(host.Bodies[0].Cancelled);
            Assert.Null(runner.Current);
        }
    }
}
