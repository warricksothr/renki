import { test } from "node:test";
import assert from "node:assert/strict";
import { subscribeEvents } from "../src/events.js";

test("proxy failure recreates CLOSED streams; native reconnect stays singular", () => {
  const sources = [],
    scheduled = new Map(),
    delays = [],
    statuses = [];
  let id = 0;
  const stop = subscribeEvents(
    {
      opened() {},
      record() {},
      interrupted() {},
      status: (e) => statuses.push(e),
    },
    {
      create: () => {
        const source = {
          readyState: 0,
          listeners: {},
          addEventListener(k, fn) {
            this.listeners[k] = fn;
          },
          close() {
            this.closed = true;
          },
        };
        sources.push(source);
        return source;
      },
      timers: {
        setTimeout(fn, ms) {
          delays.push(ms);
          scheduled.set(++id, fn);
          return id;
        },
        clearTimeout(key) {
          scheduled.delete(key);
        },
      },
    },
  );
  sources[0].onerror();
  assert.equal(scheduled.size, 0);
  sources[0].readyState = 2;
  sources[0].onerror();
  sources[0].onerror();
  assert.equal(scheduled.size, 1);
  scheduled.get(1)();
  scheduled.delete(1);
  assert.equal(sources.length, 2);
  assert.equal(sources[0].closed, true);
  sources[0].listeners.status("obsolete");
  sources[1].listeners.status("current");
  assert.deepEqual(statuses, ["current"]);
  sources[1].readyState = 2;
  sources[1].onerror();
  assert.deepEqual(delays, [1000, 2000]);
  stop();
  assert.equal(scheduled.size, 0);
  sources[1].onerror();
  assert.equal(scheduled.size, 0);
});
