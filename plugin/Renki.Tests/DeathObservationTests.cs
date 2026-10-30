using System;
using Renki.Tasks;
using Xunit;

namespace Renki.Tests
{
    public sealed class DeathObservationTests
    {
        sealed class Body
        {
            public Action Death;
            public bool Valid = true, Owner = true, Dead, SameIdentity = true;
            public int Subscriptions, Removals;
            public DeathObservation Watch(Func<bool> scope) => new DeathObservation(
                callback => { Death += callback; Subscriptions++; },
                callback => { Death -= callback; Removals++; },
                scope, () => Valid && Owner && Dead && SameIdentity);
            public void DieAndDestroy()
            {
                Dead = true;
                Death?.Invoke(); // Character.OnDeath, before ZNetScene resets its view.
                Valid = false;
            }
        }

        [Fact]
        public void CallbackSurvivesImmediateViewResetBeforeNextTaskTick()
        {
            var body = new Body();
            using var death = body.Watch(() => true);
            body.DieAndDestroy();
            Assert.False(body.Valid);
            Assert.True(death.Observed);
        }

        [Fact]
        public void UnloadOrAnotherCreaturesDeathCannotConfirmSelectedTarget()
        {
            var selected = new Body(); var other = new Body();
            using var death = selected.Watch(() => true);
            other.DieAndDestroy();
            selected.Valid = false;
            Assert.False(death.Observed);
        }

        [Theory]
        [InlineData("remote owner")]
        [InlineData("wrong identity")]
        [InlineData("not dead")]
        [InlineData("invalid view")]
        public void UnauthoritativeCallbackCannotConfirmDeath(string cause)
        {
            var body = new Body { Dead = true };
            using var death = body.Watch(() => true);
            if (cause == "remote owner") body.Owner = false;
            if (cause == "wrong identity") body.SameIdentity = false;
            if (cause == "not dead") body.Dead = false;
            if (cause == "invalid view") body.Valid = false;
            body.Death?.Invoke();
            Assert.False(death.Observed);
        }

        [Theory]
        [InlineData(false)]
        [InlineData(true)]
        public void ScopeChangeRevokesCallbacksAndAlreadyLatchedDeath(bool beforeDeath)
        {
            var current = true; var body = new Body();
            using var death = body.Watch(() => current);
            if (beforeDeath) current = false;
            body.DieAndDestroy();
            current = false;
            Assert.False(death.Observed);
            current = true;
            Assert.False(death.Observed); // Reconnection cannot resurrect old evidence.
            Assert.Null(body.Death);
            Assert.Equal(1, body.Removals);
        }

        [Fact]
        public void HandOffKeepsTheSameSubscriptionAndRetainsDeathBetweenStages()
        {
            var body = new Body();
            DeathObservation kite = body.Watch(() => true);
            body.DieAndDestroy();
            var melee = kite; kite = null;
            Assert.True(melee.Observed);
            Assert.Equal(1, body.Subscriptions);
            melee.Dispose();
            Assert.Null(body.Death);
        }

        [Theory]
        [InlineData(false)]
        [InlineData(true)]
        public void CancelOrRetreatUnsubscribesAndRejectsAlreadyQueuedCallbacks(bool latched)
        {
            var body = new Body(); var death = body.Watch(() => true);
            var queued = body.Death;
            if (latched) body.DieAndDestroy();
            death.Dispose(); death.Dispose();
            body.Dead = true; queued();
            Assert.False(death.Observed);
            Assert.Null(body.Death);
            Assert.Equal(1, body.Removals);
        }
    }
}
