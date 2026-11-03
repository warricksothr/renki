import test from "node:test";
import assert from "node:assert/strict";
import { lootForMap, dropStyles } from "../src/loot.js";
const row = () => ({
  loot: {
    connection: 1,
    world: "one",
    worldUid: "uid",
    fresh: true,
    at: "2026-11-03T18:00:00Z",
    frameAt: "2026-11-03T17:59:59Z",
    anchor: { name: "Nimah", pos: [1, 2, 3] },
    reach: 15,
    settle: { centre: [1, 2, 3], radius: 6 },
    drops: [{ id: "one", pos: [4, 2, 3], decision: "eligible" }],
  },
});
const status = {
  connected: true,
  phase: "in_world",
  connection: 1,
  world: "one",
  worldUid: "uid",
};
test("Loot overlays expire and never cross connection, world or offline boundaries", () => {
  assert.ok(lootForMap(row(), status, 100, 200));
  for (const change of [
    { connected: false },
    { connection: 2 },
    { world: "two" },
    { worldUid: "other" },
    { phase: "menu" },
  ])
    assert.equal(lootForMap(row(), { ...status, ...change }, 100, 200), null);
  assert.equal(lootForMap(row(), status, 100, 4100), null);
  const r = row();
  r.loot.fresh = false;
  assert.equal(lootForMap(r, status, 100, 200), null);
  r.loot.fresh = true;
  r.loot.anchor.pos[0] = NaN;
  assert.equal(lootForMap(r, status, 100, 200), null);
});
test("Loot decision styles remain distinct and malformed points are excluded", () => {
  assert.equal(new Set(Object.values(dropStyles).map((s) => s.shape)).size, 3);
  const r = row();
  r.loot.drops.push({ id: "bad", pos: [NaN, 0, 0], decision: "eligible" });
  assert.equal(lootForMap(r, status, 100, 200).drops.length, 1);
  r.receivedAt = 100;
  assert.equal(
    lootForMap(r, status, 4000, 5100),
    null,
    "a poll must not refresh an unchanged frame",
  );
});
