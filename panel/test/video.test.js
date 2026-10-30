import { test } from "node:test";
import assert from "node:assert/strict";
import { VideoSession } from "../src/video.js";
const status = {
  connected: true,
  phase: "in_world",
  connection: 7,
  world: "Test",
  worldUid: "42",
};
function fixture() {
  let now = 1000,
    body = { enabled: true, content: "gameplay", scope: "7:Test:42" },
    calls = [];
  const v = new VideoSession(
    () => {},
    () => {},
    {
      now: () => now,
      fetch: async (path, options) => {
        calls.push([path, options]);
        return new Response(JSON.stringify(body), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        });
      },
    },
  );
  v.setStatus(status);
  v.pc = {
    connectionState: "connected",
    close() {
      this.closed = true;
    },
  };
  v.connectedAt = now;
  v.transportFrame = now;
  return { v, calls, time: (n) => (now = n), body: (b) => (body = b) };
}
test("decoded video and verified matching health both required; ICE alone never live", async () => {
  const { v, time } = fixture();
  await v.tick();
  assert.notEqual(v.state.phase, "gameplay");
  v.decoded();
  assert.notEqual(v.state.phase, "gameplay");
  v.decoded();
  assert.equal(v.state.phase, "gameplay");
  time(4101);
  v.present();
  assert.equal(
    v.state.phase,
    "standby",
    "expired health covers even a connected peer",
  );
});
test("scope loss immediately covers old game but preserves peer/session; rejoin waits for later decoded frames", async () => {
  const { v } = fixture();
  await v.tick();
  v.decoded();
  v.decoded();
  const pc = v.pc;
  v.setStatus(null);
  assert.equal(v.state.phase, "standby");
  assert.equal(v.pc, pc);
  v.setStatus({ ...status, connection: 8 });
  await v.tick();
  v.decoded();
  v.decoded();
  assert.notEqual(v.state.phase, "gameplay");
  v.setStatus(status);
  await v.tick();
  assert.notEqual(v.state.phase, "gameplay");
  v.decoded();
  v.decoded();
  assert.equal(v.state.phase, "gameplay");
});
test("stalled frames covered before bounded transport reconnect; health freshness is independent", async () => {
  const { v, time } = fixture();
  await v.tick();
  v.decoded();
  v.decoded();
  time(4101);
  await v.tick();
  assert.equal(v.state.phase, "reconnecting");
  assert.ok(v.pc);
  time(10001);
  await v.tick();
  assert.equal(v.pc, null);
  assert.equal(v.nextAttempt, 11001);
});
test("capture health failure covers fresh media without destroying transport", async () => {
  const { v, body } = fixture();
  await v.tick();
  v.decoded();
  v.decoded();
  const pc = v.pc;
  body({
    enabled: true,
    content: "standby",
    reason: "Waiting for verified game capture",
    scope: "7:Test:42",
  });
  await v.tick();
  assert.equal(v.state.phase, "standby");
  assert.equal(v.pc, pc);
});
test("failed negotiation budget stops after five attempts; manual retry reopens budget", async () => {
  let now = 0,
    calls = 0;
  const v = new VideoSession(
    () => {},
    () => {},
    {
      now: () => now,
      fetch: async () =>
        new Response(
          JSON.stringify({
            enabled: true,
            content: "gameplay",
            scope: "7:Test:42",
          }),
        ),
      peer: () => {
        calls++;
        throw Error("unsupported");
      },
    },
  );
  v.setStatus(status);
  for (let n = 0; n < 7; n++) {
    now += 20000;
    await v.tick();
    await new Promise((r) => setImmediate(r));
  }
  assert.equal(calls, 5);
  assert.equal(v.state.phase, "unavailable");
  v.retry();
  await new Promise((r) => setImmediate(r));
  assert.equal(calls, 6);
  v.dispose();
});
test("auth/health request failure hides old video; disabled config releases session", async () => {
  const { v, body } = fixture();
  await v.tick();
  v.decoded();
  v.decoded();
  v.env.fetch = async () => new Response("expired", { status: 401 });
  await v.tick();
  assert.equal(v.state.phase, "standby");
  v.env.fetch = async () => new Response(JSON.stringify({ enabled: false }));
  await v.tick();
  assert.equal(v.pc, null);
  assert.equal(v.state.phase, "standby");
});
test("cleanup only deletes fixed same-origin session paths", async () => {
  const { v, calls } = fixture();
  await v.remove("https://example.com/session/1");
  assert.equal(calls.length, 0);
  v.session = "/api/video/session/12345678-1234-1234-1234-123456789abc";
  v.dispose();
  assert.equal(calls[0][1].method, "DELETE");
  assert.equal(calls[0][1].headers["X-Renki-Act"], "video");
});
test("late negotiated session after disposal is deleted and cannot attach a stream", async () => {
  let resolveOffer,
    attached = 0;
  const removed = [];
  const pc = {
    iceGatheringState: "complete",
    connectionState: "new",
    addTransceiver() {},
    addEventListener() {},
    removeEventListener() {},
    async createOffer() {
      return { type: "offer", sdp: "offer" };
    },
    async setLocalDescription(d) {
      this.localDescription = d;
    },
    async setRemoteDescription() {
      throw Error("must not set a stale answer");
    },
    close() {
      this.closed = true;
    },
  };
  const v = new VideoSession(
    () => {},
    (s) => {
      if (s) attached++;
    },
    {
      peer: () => pc,
      fetch: async (path, opts) => {
        if (opts.method === "DELETE") {
          removed.push(path);
          return new Response(null, { status: 204 });
        }
        return new Promise((r) => (resolveOffer = r));
      },
    },
  );
  v.setStatus(status);
  const negotiation = v.connect();
  await new Promise((r) => setImmediate(r));
  v.dispose();
  resolveOffer(
    new Response("answer", {
      status: 201,
      headers: {
        Location: "/api/video/session/12345678-1234-1234-1234-123456789abc",
      },
    }),
  );
  await negotiation;
  assert.equal(attached, 0);
  assert.equal(pc.closed, true);
  assert.equal(removed.length, 1);
  assert.equal(v.pc, null);
});
