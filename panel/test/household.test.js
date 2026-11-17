import test from "node:test";
import assert from "node:assert/strict";
import { householdTargets, householdFoods } from "../src/household.js";
test("independent food targets retain per-food refill thresholds", () => {
  assert.deepEqual(
    householdTargets([
      { food: "Mushroom", count: "4", minimum: "2" },
      { food: "CookedMeat", count: 8, minimum: 3 },
    ]),
    {
      targets: { Mushroom: 4, CookedMeat: 8 },
      minimum: { Mushroom: 2, CookedMeat: 3 },
    },
  );
  assert.equal(
    Object.keys(
      householdTargets(
        householdFoods.map(([food]) => ({ food, count: 20, minimum: 1 })),
      ).targets,
    ).length,
    8,
  );
});
test("ambiguous, unsupported and out-of-range plans are refused", () => {
  const row = { food: "Mushroom", count: 4, minimum: 2 };
  for (const rows of [
    [],
    Array(9).fill(row),
    [row, row],
    [{ ...row, food: "DeerStew" }],
    [{ ...row, count: 1.5 }],
    [{ ...row, count: 21 }],
    [{ ...row, minimum: 5 }],
    [{ ...row, minimum: 0 }],
  ])
    assert.throws(() => householdTargets(rows));
});
