import { test } from "node:test";
import assert from "node:assert/strict";
import { CameraSession, cameraPresentation } from "../src/camera.js";
const status = {
  connected: true,
  phase: "in_world",
  connection: 3,
  world: "World",
  worldUid: "9",
};
const metadata = {
  available: true,
  config: { enabled: true, intervalS: 5 },
  imageUrl: "/api/camera.jpg?connection=3&seq=7",
  ageS: 2,
  frame: { ...status, seq: 7, capturedAt: "2026-10-30T18:00:00Z" },
};
const json = (data) => ({ ok: true, status: 200, json: async () => data });
function camera(fetch) {
  const revoked = [],
    states = [];
  const c = new CameraSession((s) => states.push(s), {
    fetch,
    now: () => Date.parse("2026-10-30T18:00:03Z"),
    createURL: () => "blob:fixture",
    revokeURL: (u) => revoked.push(u),
  });
  c.setStatus(status);
  return { c, revoked, states };
}
test("authenticated image lease shows decoding, receipt/capture age, then stale warning", async () => {
  const { c } = camera(async (path) =>
    path === "/api/camera"
      ? json(metadata)
      : { ok: true, blob: async () => new Blob(["image"]) },
  );
  await c.poll();
  assert.equal(c.state.loaded, false);
  assert.equal(cameraPresentation(c.state, c.env.now()).label, "loading image");
  c.loaded(c.state.url, 960);
  const view = cameraPresentation(c.state, c.env.now());
  assert.equal(view.receiptAge, 2);
  assert.equal(view.captureAge, 3);
  assert.equal(view.label, "latest frame");
  assert.equal(cameraPresentation(c.state, c.env.now() + 16000).label, "stale");
  c.dispose();
});
test("disconnect immediately revokes image and late requests cannot revive it", async () => {
  let resolveBlob;
  const { c, revoked } = camera(async (path) =>
    path === "/api/camera"
      ? json(metadata)
      : {
          ok: true,
          blob: () => new Promise((resolve) => (resolveBlob = resolve)),
        },
  );
  const request = c.poll();
  await new Promise((resolve) => setImmediate(resolve));
  c.setStatus({ ...status, connected: false });
  resolveBlob(new Blob(["late image"]));
  await request;
  assert.equal(c.state.url, null);
  assert.equal(c.state.phase, "offline");
  assert.deepEqual(revoked, []);
});
test("changing connection/world/setting clears a decoded scene", async () => {
  const { c, revoked } = camera(async (path) =>
    path === "/api/camera"
      ? json(metadata)
      : { ok: true, blob: async () => new Blob(["image"]) },
  );
  await c.poll();
  c.loaded(c.state.url, 960);
  c.setStatus({ ...status, connection: 4 });
  assert.equal(c.state.url, null);
  assert.deepEqual(revoked, ["blob:fixture"]);
  await c.poll();
  assert.equal(c.state.phase, "pending");
  assert.equal(c.state.url, null);
});
test("disabled, pending, superseded, network failure and decode failure have no scene", async () => {
  for (const data of [
    { ...metadata, config: { enabled: false } },
    { ...metadata, available: false },
  ]) {
    const { c } = camera(async () => json(data));
    await c.poll();
    assert.equal(c.state.url, null);
    assert.ok(["disabled", "pending"].includes(c.state.phase));
  }
  const { c } = camera(async (path) =>
    path === "/api/camera" ? json(metadata) : { ok: false, status: 409 },
  );
  await c.poll();
  assert.equal(c.state.phase, "pending");
  assert.match(c.state.message, /newer frame/);
  c.env.fetch = async () => {
    throw Error("network unavailable");
  };
  await c.poll();
  assert.equal(c.state.phase, "failed");
  assert.equal(c.state.url, null);
  c.env.fetch = async (path) =>
    path === "/api/camera"
      ? json(metadata)
      : { ok: true, blob: async () => new Blob(["image"]) };
  await c.poll();
  c.loaded(c.state.url, 0);
  assert.equal(c.state.phase, "failed");
  assert.equal(c.state.url, null);
});
test("setting refusal preserves server authority and reports failure without unhandled rejection", async () => {
  let posted;
  const { c } = camera(async (path, options) => {
    if (path === "/api/setting") {
      posted = options;
      return { ok: false, text: async () => "Setting is pinned" };
    }
    return json({ ...metadata, available: false });
  });
  await c.setting("screenshot.enabled", true);
  assert.equal(posted.headers["X-Renki-Act"], "camera");
  assert.equal(JSON.parse(posted.body).value, true);
  assert.match(c.state.notice, /Setting is pinned/);
  assert.equal(c.state.saving, false);
  assert.equal(c.state.url, null);
});
