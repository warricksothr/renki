using System;
using Newtonsoft.Json.Linq;
using Renki.Protocol;
using Renki.Tasks;
using Xunit;

namespace Renki.Tests
{
    public sealed class TravelRecoveryTests
    {
        [Fact]
        public void EscapeReceiptAndContinuousQuietAreRequiredBeforeResuming()
        {
            var p = new TravelRecovery();
            Assert.Equal(TravelMode.Walk, p.Decide(0, false, false));
            Assert.Equal(TravelMode.Escape, p.Decide(1, true, false));
            Assert.Equal(TravelMode.Escape, p.Decide(10, false, false));
            p.Escaped(true);
            Assert.Equal(TravelMode.Observe, p.Decide(11, false, false));
            Assert.Equal(TravelMode.Observe, p.Decide(15.99, false, false));
            Assert.Equal(TravelMode.Walk, p.Decide(16, false, false));
            Assert.Equal(1, p.Attempts);
        }

        [Fact]
        public void RenewedPursuitBreaksObservationAndCannotResetAttemptBudget()
        {
            var p = new TravelRecovery();
            for (var i = 0; i < TravelRecovery.AttemptLimit; i++)
            {
                Assert.Equal(TravelMode.Escape, p.Decide(i * 4, true, false));
                p.Escaped(true);
                Assert.Equal(TravelMode.Observe, p.Decide(i * 4 + 1, false, false));
            }
            Assert.Equal(TravelMode.Refused, p.Decide(12, true, false));
            p.Escaped(true);
            Assert.Equal(TravelMode.Refused, p.Decide(20, false, false));
            Assert.Equal(3, p.Attempts);
        }

        [Theory]
        [InlineData("escape_time")]
        [InlineData("observe_time")]
        [InlineData("failed_escape")]
        [InlineData("unsafe_body")]
        [InlineData("cancel")]
        [InlineData("clock")]
        public void FailureRemainsTerminalWithoutManufacturingArrival(string cause)
        {
            var p = new TravelRecovery();
            p.Decide(1, true, false);
            if (cause == "observe_time") p.Escaped(true);
            if (cause == "failed_escape") p.Escaped(false);
            if (cause == "cancel") p.Cancel();
            Assert.Equal(TravelMode.Refused, p.Decide(cause.EndsWith("time") ? 31 : cause == "clock" ? 0 : 2,
                false, cause == "unsafe_body"));
            Assert.NotNull(p.Reason);
            Assert.Equal(TravelMode.Refused, p.Decide(40, false, false));
        }

        [Fact]
        public void RepeatedQuietResumptionsStillConsumeTheSameLifetimeBudget()
        {
            var p = new TravelRecovery();
            for (var i = 0; i < 3; i++)
            {
                p.Decide(i * 10, true, false); p.Escaped(true);
                p.Decide(i * 10 + 1, false, false);
                Assert.Equal(TravelMode.Walk, p.Decide(i * 10 + 6, false, false));
            }
            Assert.Equal(TravelMode.Refused, p.Decide(30, true, false));
        }

        [Fact]
        public void BlockedRouteRequiresObservedChangeAndAQuietWaitBeforeReplanning()
        {
            var p = new TravelRecovery();
            p.Decide(0, false, false);
            Assert.Equal(TravelMode.Blocked, p.RouteBlocked());
            Assert.Equal(TravelMode.Blocked, p.Decide(4.99, false, false, routeChanged: true));
            Assert.Equal(TravelMode.Blocked, p.Decide(5, false, false));
            Assert.Equal(TravelMode.Blocked, p.Decide(20, false, false));
            Assert.Equal(TravelMode.Walk, p.Decide(21, false, false, routeChanged: true));
            Assert.Equal(1, p.Attempts);
        }

        [Fact]
        public void ChangedRouteAndEscapeShareTheLifetimeAttemptBudget()
        {
            var p = new TravelRecovery();
            p.Decide(0, false, false); p.RouteBlocked();
            Assert.Equal(TravelMode.Escape, p.Decide(1, true, false));
            p.Escaped(true); p.Decide(2, false, false);
            Assert.Equal(TravelMode.Walk, p.Decide(7, false, false));
            Assert.Equal(TravelMode.Blocked, p.RouteBlocked());
            Assert.Equal(TravelMode.Walk, p.Decide(12, false, false, routeChanged: true));
            Assert.Equal(TravelMode.Refused, p.RouteBlocked());
            Assert.Equal(3, p.Attempts);
        }

        [Theory]
        [InlineData("timeout")]
        [InlineData("cancel")]
        [InlineData("unsafe")]
        [InlineData("clock")]
        public void WaitingForChangedConditionsCannotExtendAnUnsafeOrExhaustedTask(string reason)
        {
            var p = new TravelRecovery();
            p.Decide(1, false, false); p.RouteBlocked();
            if (reason == "cancel") p.Cancel();
            Assert.Equal(TravelMode.Refused, p.Decide(reason == "timeout" ? 31 : reason == "clock" ? 0 : 2,
                false, reason == "unsafe", routeChanged: true));
            Assert.Equal(TravelMode.Refused, p.Decide(32, false, false, routeChanged: true));
        }

        [Theory]
        [InlineData(35, true, true, false)]
        [InlineData(22, true, true, true)]
        [InlineData(22, false, true, false)]
        [InlineData(19, false, true, true)]
        [InlineData(14, false, false, true)]
        [InlineData(16, false, false, false)]
        public void AwarenessDistinguishesCloseAndEngagedThreats(double gap, bool target, bool engaged, bool expected) =>
            Assert.Equal(expected, TravelRecovery.Engaged(gap, target, engaged));

        [Fact]
        public void RecoveryDoesNotAuthorizeUnsafeReservesOrFooting()
        {
            // Finishing escape must leave the known-target trigger range;
            // otherwise observation would immediately consume every retry.
            Assert.False(TravelRecovery.Engaged(TravelRecovery.EscapeDistance, true, true));
            Assert.True(TravelRecovery.Safe(40, 60, 10, 1, false, false));
            Assert.False(TravelRecovery.Safe(19, 60, 10, 1, false, false));
            Assert.False(TravelRecovery.Safe(40, 110, 10, 1, false, false));
            Assert.False(TravelRecovery.Safe(40, 60, 7, 1, false, false));
            Assert.False(TravelRecovery.Safe(40, 60, 10, 0, false, false));
            Assert.False(TravelRecovery.Safe(40, 60, 10, 1, true, false));
            Assert.False(TravelRecovery.Safe(40, 60, 10, 1, false, true));
            Assert.False(TravelRecovery.Safe(double.PositiveInfinity, 60, 10, 1, false, false));
        }

        [Fact]
        public void ProtocolRequiresExplicitCheckedTravelAndValidBoolean()
        {
            TaskRequest Request(string flags) => new TaskRequest { Task = "move_to", Args = JObject.Parse("{\"to\":[1,2,3]" + flags + "}") };
            Assert.False(((MoveToArgs)TaskArgs.Parse(Request(""))).RecoverThreats);
            Assert.True(((MoveToArgs)TaskArgs.Parse(Request(",\"dry\":true,\"avoidHostiles\":true,\"recoverThreats\":true"))).RecoverThreats);
            Assert.Throws<BadArgsException>(() => TaskArgs.Parse(Request(",\"recoverThreats\":true")));
            Assert.Throws<BadArgsException>(() => TaskArgs.Parse(Request(",\"dry\":true,\"recoverThreats\":true")));
            Assert.Throws<BadArgsException>(() => TaskArgs.Parse(Request(",\"dry\":true,\"avoidHostiles\":true,\"recoverThreats\":\"yes\"")));
        }
    }
}
