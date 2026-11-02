export function areaEvent(events) {
  const e = events?.randomEvent;
  return e?.name &&
    Array.isArray(e.center) &&
    e.center.length === 3 &&
    e.center.every(Number.isFinite) &&
    Number.isFinite(e.radius) &&
    e.radius > 0
    ? e
    : null;
}

export function eventSummary(events) {
  if (!events) return "World events unknown";
  const area = areaEvent(events);
  const local = events.localEvent;
  if (local && (local.kind !== "random" || !area))
    return `${local.message || local.name} · affecting Tas · area unknown`;
  if (area)
    return `${area.message || area.name} · ${area.inside ? "Tas inside" : "Tas outside"} · ${Math.round(area.radius)} m radius`;
  return "No world event observed";
}

export function eventDetails(events) {
  const e = areaEvent(events);
  if (!e) return [];
  return [
    `World event · ${e.message || e.name} (${e.name}) · radius ${Math.round(e.radius)} m`,
    `Observed timer ${Math.round(e.elapsedS)} s${e.durationS > 0 ? ` / ${Math.round(e.durationS)} s` : ""}${e.timerMayPause ? " · may pause while area is empty" : ""} · survivors may remain after event ends`,
  ];
}
