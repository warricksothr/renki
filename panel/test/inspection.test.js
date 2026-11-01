import test from "node:test";
import assert from "node:assert/strict";
import {
  InspectionSession,
  inspectionPresentation,
  inspectionGuardResult,
} from "../src/inspection.js";

const status = {
  connected: true,
  phase: "in_world",
  connection: 7,
  world: "one",
  worldUid: "uid",
};
const snapshot = (id = "first") => ({
  schemaVersion: 1,
  contract: "inspection-v1",
  ...status,
  goal: { id },
  freshness: { status: "fresh", ageS: 0.5 },
});
const settle = () => new Promise((resolve) => setImmediate(resolve));
const response = (value) => ({ ok: true, json: async () => value });
function fixture() {
  const requests = [];
  const session = new InspectionSession(() => {}, {
    now: () => 1000,
    fetch: (path, options) =>
      new Promise((resolve) => requests.push({ path, options, resolve })),
  });
  session.setStatus(status);
  return { session, requests };
}

test("goal selection aborts prior request and ignores its late response; all requests are GET", async () => {
  const { session, requests } = fixture();
  session.select("second & goal");
  assert.equal(requests[0].options.signal.aborted, true);
  assert.equal(requests[1].path, "/api/inspection?goal=second%20%26%20goal");
  requests[1].resolve(response(snapshot("second")));
  await settle();
  requests[0].resolve(response(snapshot("first")));
  await settle();
  assert.equal(session.state.snapshot.goal.id, "second");
  assert.ok(requests.every((r) => r.options.method === "GET"));
  session.dispose();
});

test("world, connection and offline changes invalidate pending responses", async () => {
  for (const changed of [
    { ...status, world: "two" },
    { ...status, worldUid: "other" },
    { ...status, connection: 8 },
    { ...status, connected: false },
  ]) {
    const { session, requests } = fixture();
    session.setStatus(changed);
    requests[0].resolve(response(snapshot()));
    await settle();
    assert.equal(session.state.snapshot, null);
    assert.equal(requests.length, 2);
    session.dispose();
    requests[1].resolve(response(snapshot()));
    await settle();
    assert.equal(session.state.snapshot, null);
  }
});

test("failed refresh retains saved evidence but withholds live ownership and guards", async () => {
  const { session, requests } = fixture();
  requests[0].resolve(response(snapshot()));
  await settle();
  const poll = session.poll();
  requests[1].resolve({ ok: false, status: 503 });
  await poll;
  assert.equal(session.state.snapshot.goal.id, "first");
  const view = inspectionPresentation(session.state, status, 1100);
  assert.equal(view.fresh, false);
  assert.match(view.reason, /503/);
  assert.equal(
    inspectionGuardResult({ layer: "pursuit", result: "pass" }, view),
    "stale",
  );
  assert.equal(
    inspectionGuardResult({ layer: "progress", result: "recorded" }, view),
    "recorded",
  );
  session.dispose();
});

test("freshness expires locally and pursuit evidence has the stricter one-second limit", () => {
  const state = { snapshot: snapshot(), receivedAt: 1000 };
  assert.equal(inspectionPresentation(state, status, 1000).fresh, true);
  const delayed = inspectionPresentation(state, status, 1600);
  assert.equal(delayed.fresh, true);
  assert.equal(
    inspectionGuardResult({ layer: "pursuit", result: "pass" }, delayed),
    "stale",
  );
  assert.equal(inspectionPresentation(state, status, 4000).fresh, false);
  for (const changed of [
    { ...status, connected: false },
    { ...status, phase: "menu" },
    { ...status, world: "two" },
    { ...status, worldUid: "other" },
    { ...status, connection: 8 },
  ]) {
    assert.equal(inspectionPresentation(state, changed, 1000).fresh, false);
  }
});
