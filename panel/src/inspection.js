import { scope, online } from "./status.js";

// Read-only fetches share the exact API used by agent operators. A generation
// prevents slow responses from resurrecting a previous goal or connection.
export class InspectionSession {
  constructor(publish, env = {}) {
    this.publish = publish;
    this.env = {
      fetch: (...args) => fetch(...args),
      now: () => Date.now(),
      ...env,
    };
    this.state = { snapshot: null, receivedAt: 0, error: "", selected: "" };
    this.generation = 0;
    this.disposed = false;
  }
  update(patch) {
    this.state = { ...this.state, ...patch };
    this.publish(this.state);
  }
  invalidate() {
    this.generation++;
    this.abort?.abort();
    this.pending = null;
  }
  setStatus(status) {
    const changed =
      !this.status ||
      scope(status) !== scope(this.status) ||
      status?.connected !== this.status?.connected;
    this.status = status;
    if (changed) {
      this.invalidate();
      this.update({ error: "Refreshing scoped inspection" });
      this.poll();
    }
  }
  select(id) {
    this.invalidate();
    this.update({ selected: id, snapshot: null, receivedAt: 0, error: "" });
    this.poll();
  }
  async poll() {
    if (this.disposed || this.pending || !this.status) return;
    const generation = this.generation;
    const abort = new AbortController();
    this.abort = abort;
    this.pending = abort;
    try {
      const response = await this.env.fetch(
        `/api/inspection${this.state.selected ? `?goal=${encodeURIComponent(this.state.selected)}` : ""}`,
        { method: "GET", cache: "no-store", signal: abort.signal },
      );
      if (!response.ok)
        throw new Error(`Inspection unavailable (${response.status})`);
      const snapshot = await response.json();
      if (snapshot.schemaVersion !== 1 || snapshot.contract !== "inspection-v1")
        throw new Error("Unsupported inspection response");
      if (this.disposed || generation !== this.generation) return;
      this.update({ snapshot, receivedAt: this.env.now(), error: "" });
    } catch (error) {
      if (this.disposed || generation !== this.generation) return;
      this.update({ error: error.message || "Inspection unavailable" });
    } finally {
      if (this.pending === abort) this.pending = null;
    }
  }
  dispose() {
    this.disposed = true;
    this.invalidate();
  }
}

export function inspectionPresentation(state, status, now = Date.now()) {
  const snapshot = state.snapshot;
  const matches =
    online(status) &&
    snapshot &&
    snapshot.connection === status.connection &&
    snapshot.world === status.world &&
    (snapshot.worldUid || "") === (status.worldUid || "");
  const age =
    (snapshot?.freshness?.ageS ?? Infinity) +
    Math.max(0, now - state.receivedAt) / 1000;
  const fresh = !!(
    matches &&
    !state.error &&
    snapshot.freshness.status === "fresh" &&
    age >= 0 &&
    age <= 3
  );
  return {
    fresh,
    age,
    reason:
      state.error ||
      (!online(status)
        ? "Offline · saved evidence only"
        : fresh
          ? "Fresh world observation"
          : "Live observation unavailable or stale"),
  };
}

export function inspectionGuardResult(guard, presentation) {
  if (guard.layer === "progress" || guard.source === "goal.kind")
    return guard.result;
  if (
    !presentation.fresh ||
    (guard.layer === "pursuit" && presentation.age > 1)
  )
    return "stale";
  return guard.result;
}
