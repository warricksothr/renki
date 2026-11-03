import { dropLabels } from "./loot.js";
import { humanize } from "./status.js";
const parameters = {
  items: "Items",
  radius: "Maximum reach (m)",
  baseRadius: "Initial reach (m)",
  growS: "Growth time (s)",
  settleDelayS: "Settle delay (s)",
  settleRadius: "Initial settle radius (m)",
  settleMaxRadius: "Maximum settle radius (m)",
  settleSlack: "Settle slack (m)",
  maxWeightFraction: "Maximum pack weight fraction",
  maxTargets: "Targets per run",
};
const display = (v) => (Array.isArray(v) ? v.map(humanize).join(", ") : v);
export function LootCard({ rows, selected, onSelect, overlay, error }) {
  const row = rows.find((r) => r.name === selected);
  const d = row?.loot;
  return (
    <article class="loot-card" aria-label="Loot behaviour">
      <div class="loot-heading">
        <h3>
          {row?.name || "Loot"}{" "}
          <span class="chip">
            {row
              ? row.definition.enabled
                ? "Enabled"
                : "Disabled"
              : "Not configured"}
          </span>
        </h3>
        {rows.length > 1 && (
          <label>
            Instance{" "}
            <select
              aria-label="Loot instance"
              value={selected}
              onChange={(e) => onSelect(e.currentTarget.value)}
            >
              {rows.map((r) => (
                <option key={r.name} value={r.name}>
                  {r.name}
                </option>
              ))}
            </select>
          </label>
        )}
      </div>
      <p class="muted">
        {row?.running ? "Working" : row?.pending ? "Queued" : "Waiting"} ·
        Candidate decisions show pickup policy; orders and safety still govern
        each run.
      </p>
      {overlay ? (
        <>
          <p data-testid="loot-reach">
            <strong>Anchor: {overlay.anchor.name}</strong> · Settled{" "}
            {Math.floor(overlay.settle?.settledS || 0)} s · Reach{" "}
            {overlay.reach.toFixed(1)} / {overlay.params.radius} m
          </p>
          {row.activity &&
            row.activity.player === overlay.anchor.name &&
            row.activity.ageS < 30 && (
              <p>
                Inferred nearby work: {humanize(row.activity.kind)} ·{" "}
                {humanize(row.activity.prefab)} ·{" "}
                {Math.floor(row.activity.ageS)} s ago
              </p>
            )}
          <div class="loot-legend" aria-label="Loot map key">
            <span class="loot-eligible">● Eligible</span>
            <span class="loot-deliberate">× Deliberate drop</span>
            <span class="loot-out_of_reach">○ Out of reach</span>
            <span>Dashed: settle area · Solid: pickup reach</span>
          </div>
          <details>
            <summary>Drop decisions ({overlay.drops.length})</summary>
            <ul class="loot-drops">
              {overlay.drops.map((drop) => (
                <li class={`loot-${drop.decision}`} key={drop.id}>
                  {humanize(drop.prefab)} · {dropLabels[drop.decision]}{" "}
                  <small>{drop.id}</small>
                </li>
              ))}
            </ul>
          </details>
        </>
      ) : (
        <p data-testid="loot-unavailable" class="muted">
          {error ||
            "Live reach unavailable · waiting for a fresh anchor observation"}
        </p>
      )}
      {d && (
        <details>
          <summary>Loot parameters and defaults</summary>
          <dl class="loot-parameters">
            {Object.entries(parameters).map(([key, label]) => (
              <div key={key}>
                <dt>{label}</dt>
                <dd>
                  {display(d.params[key])}{" "}
                  <small>default {display(d.defaults[key])}</small>
                </dd>
              </div>
            ))}
          </dl>
        </details>
      )}
      {row?.last && <p class="muted">Last run: {row.last}</p>}
    </article>
  );
}
