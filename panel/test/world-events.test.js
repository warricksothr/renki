import test from "node:test";
import assert from "node:assert/strict";
import { areaEvent, eventSummary, eventDetails } from "../src/world-events.js";

test("unknown, observed clear and local notices remain distinct", () => {
  assert.equal(eventSummary(null), "World events unknown");
  assert.equal(eventSummary({}), "No world event observed");
  const forced = {
    localEvent: { name: "boss", message: "Boss nearby", kind: "local" },
  };
  assert.match(eventSummary(forced), /Boss nearby.*area unknown/);
  assert.equal(areaEvent(forced), null);
});

test("event area and paused timer are observations, not an end countdown", () => {
  const events = {
    randomEvent: {
      name: "army_greydwarf",
      message: "The forest is moving",
      center: [10, 30, 20],
      radius: 96,
      inside: false,
      elapsedS: 121,
      durationS: 120,
      timerMayPause: true,
    },
  };
  assert.match(eventSummary(events), /Tas outside.*96 m/);
  assert.match(
    eventDetails(events).join(" "),
    /may pause.*survivors may remain/,
  );
  assert.ok(areaEvent(events));
  events.randomEvent.center = [0, NaN, 0];
  assert.equal(areaEvent(events), null);
});
