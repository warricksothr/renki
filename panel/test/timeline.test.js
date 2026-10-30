import { test } from "node:test";
import assert from "node:assert/strict";
import { summarize } from "../src/timeline.js";
test("task failure uses the structured status and human-readable reason", () => {
  assert.deepEqual(
    summarize({
      cat: "task",
      line: "raw trace {…}",
      frame: {
        type: "task",
        task: "move_to",
        status: "failed",
        reason: "path_blocked",
      },
    }),
    { title: "move to · failed", reason: "path blocked" },
  );
});
test("unfamiliar records retain their message and tolerate missing fields", () => {
  assert.equal(
    summarize({
      msg: "Safety threshold reached",
      attrs: { reason: "low_health" },
    }).reason,
    "low health",
  );
  assert.equal(summarize({}).title, "Event");
});
