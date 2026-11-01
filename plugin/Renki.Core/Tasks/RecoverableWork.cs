namespace Renki.Tasks
{
    public interface IWorkEncounter
    {
        bool Tick(double dt, bool canSuspend, out Outcome failure);
        void Cancel();
    }

    public interface IRecoverableWork : IBodyTask
    {
        // Observe before the encounter can issue any input. False means a
        // game use has already been sent and cannot safely be suspended.
        bool Checkpoint();
        Outcome Resume();
    }

    /// <summary>One body retains its original work child through an encounter.</summary>
    public sealed class RecoverableWorkTask : IBodyTask
    {
        readonly IRecoverableWork work;
        readonly IWorkEncounter encounter;
        bool held, canSuspend, cancelled;

        public RecoverableWorkTask(IRecoverableWork work, IWorkEncounter encounter)
        { this.work = work; this.encounter = encounter; }

        public Outcome Tick(double dt)
        {
            if (cancelled) return Outcome.Failed(Protocol.Wire.ReasonStopped, "collection cancelled");
            if (!held) canSuspend = work.Checkpoint();
            if (encounter.Tick(dt, canSuspend, out var failure))
            {
                if (!canSuspend)
                    return Finish(Outcome.Failed(Protocol.Wire.ReasonStopped, "collection use is unresolved at encounter; not suspended or replayed"));
                held = true;
                // Escape can auto-collect an item. Until Resume validates the
                // checkpoint, an encounter failure is not an ordinary empty
                // source result that a controller may turn into a receipt.
                return failure == null ? null : Finish(Outcome.Failed(Protocol.Wire.ReasonStopped,
                    "collection encounter ended before source revalidation; not replayed: " + failure.Reason + ": " + failure.Detail));
            }
            if (held)
            {
                var refused = work.Resume();
                if (refused != null) return Finish(refused);
                held = false;
            }
            var ended = work.Tick(dt);
            return ended == null ? null : Finish(ended);
        }

        Outcome Finish(Outcome result) { Cancel(); return result; }
        public void Cancel()
        {
            if (cancelled) return;
            cancelled = true; work.Cancel(); encounter.Cancel();
        }
    }

    /// <summary>Inventory gain during escape cannot prove a specific pickup.</summary>
    public sealed class WorkCheckpoint
    {
        readonly string entity, item;
        readonly int count, carried;
        readonly bool available;

        public WorkCheckpoint(string entity, string item, int count, int carried, bool available)
        { this.entity = entity; this.item = item; this.count = count; this.carried = carried; this.available = available; }

        public bool Matches(WorkCheckpoint next) => next != null && available && next.available &&
            !string.IsNullOrEmpty(entity) && !string.IsNullOrEmpty(item) && count > 0 && carried >= 0 &&
            entity == next.entity && item == next.item && count == next.count && carried == next.carried;
    }
}
