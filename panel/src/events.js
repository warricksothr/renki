// EventSource retries transport failures, but a proxy's 502/503 can permanently
// close it. Recreate only CLOSED sources; leave native retry ownership intact.
export function subscribeEvents(
  { opened, record, status, interrupted },
  { create = () => new EventSource("/api/events"), timers = globalThis } = {},
) {
  let source,
    timer = null,
    delay = 1000,
    stopped = false;
  function connect() {
    if (stopped) return;
    const current = (source = create());
    const active = () => !stopped && current === source;
    current.onopen = () => {
      if (!active()) return;
      if (timer !== null) timers.clearTimeout(timer);
      timer = null;
      delay = 1000;
      opened();
    };
    current.addEventListener("record", (e) => active() && record(e));
    current.addEventListener("status", (e) => active() && status(e));
    current.onerror = () => {
      if (!active()) return;
      interrupted();
      if (current.readyState === 2 && timer === null) {
        timer = timers.setTimeout(() => {
          timer = null;
          current.close();
          connect();
        }, delay);
        delay = Math.min(delay * 2, 10000);
      }
    };
  }
  connect();
  return () => {
    stopped = true;
    if (timer !== null) timers.clearTimeout(timer);
    source?.close();
  };
}
