using System;

namespace Renki.Tasks
{
    /// <summary>A scoped authoritative callback, retained after its source disappears.</summary>
    public sealed class DeathObservation : IDisposable
    {
        readonly Action<Action> unsubscribe;
        readonly Func<bool> currentScope, authoritativeSource;
        bool observed, disposed;

        public DeathObservation(Action<Action> subscribe, Action<Action> unsubscribe,
            Func<bool> currentScope, Func<bool> authoritativeSource)
        {
            this.unsubscribe = unsubscribe;
            this.currentScope = currentScope;
            this.authoritativeSource = authoritativeSource;
            subscribe(OnDeath);
        }

        public bool Observed
        {
            get
            {
                if (!disposed && !currentScope()) Dispose();
                return !disposed && observed;
            }
        }

        void OnDeath()
        {
            if (disposed) return;
            if (!currentScope()) { Dispose(); return; }
            if (authoritativeSource()) observed = true;
        }

        public void Dispose()
        {
            if (disposed) return;
            disposed = true; observed = false;
            unsubscribe(OnDeath);
        }
    }
}
