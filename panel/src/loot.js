const point = (p) =>
  Array.isArray(p) && p.length === 3 && p.every(Number.isFinite);
export const dropLabels = {
  eligible: "Eligible",
  deliberate: "Deliberate drop",
  out_of_reach: "Out of reach",
  item_not_selected: "Item not selected",
  unknown_yield: "Contents unknown",
  capacity: "Pack capacity",
  inventory_unknown: "Pack unknown",
  attempted: "Already attempted this run",
  target_limit: "Run limit reached",
  unobserved: "Not observed",
};
export const dropStyles = {
  eligible: { color: "#72e7ac", shape: "filled" },
  deliberate: { color: "#ffc879", shape: "cross" },
  out_of_reach: { color: "#b2bfda", shape: "hollow" },
};
// Use server-relative frame age plus local receipt age, avoiding clock skew.
export function lootForMap(row, status, receivedAt, now = Date.now()) {
  const d = row?.loot;
  receivedAt = row?.receivedAt ?? receivedAt;
  if (
    !status?.connected ||
    status.phase !== "in_world" ||
    !d?.fresh ||
    d.connection !== status.connection ||
    d.world !== status.world ||
    (d.worldUid || "") !== (status.worldUid || "") ||
    now < receivedAt ||
    now - receivedAt >= 5000 ||
    !Number.isFinite(Date.parse(d.at) - Date.parse(d.frameAt)) ||
    Date.parse(d.at) < Date.parse(d.frameAt) ||
    Date.parse(d.at) - Date.parse(d.frameAt) + now - receivedAt >= 5000 ||
    !point(d.anchor?.pos) ||
    !Number.isFinite(d.reach) ||
    d.reach <= 0 ||
    d.reach > 40
  )
    return null;
  const settle =
    d.settle &&
    point(d.settle.centre) &&
    Number.isFinite(d.settle.radius) &&
    d.settle.radius > 0 &&
    d.settle.radius <= 20
      ? d.settle
      : null;
  return {
    ...d,
    settle,
    drops: (d.drops || []).filter(
      (drop) => point(drop.pos) && dropLabels[drop.decision],
    ),
  };
}
