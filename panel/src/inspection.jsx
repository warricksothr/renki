import { useEffect, useRef, useState } from "preact/hooks";
import { useStatus, getStatus, subscribeStatus, humanize } from "./status.js";
import {
  InspectionSession,
  inspectionPresentation,
  inspectionGuardResult,
} from "./inspection.js";

export function Inspection() {
  const status = useStatus();
  const [state, setState] = useState({
    snapshot: null,
    receivedAt: 0,
    error: "",
    selected: "",
  });
  const [now, setNow] = useState(Date.now());
  const [selectedPhase, setSelectedPhase] = useState("");
  const session = useRef();
  useEffect(() => {
    const inspector = new InspectionSession(setState);
    session.current = inspector;
    inspector.setStatus(getStatus());
    const unsubscribe = subscribeStatus((s) => inspector.setStatus(s));
    const timer = setInterval(() => {
      setNow(Date.now());
      if (!document.hidden) inspector.poll();
    }, 1000);
    return () => {
      unsubscribe();
      clearInterval(timer);
      inspector.dispose();
    };
  }, []);
  const data = state.snapshot;
  const view = inspectionPresentation(state, status, now);
  const goal = data?.goal;
  const graph = data?.graph;
  const saved = graph?.id === "food-adapter-v1";
  const active = saved || view.fresh ? graph?.active : "";
  const phase =
    graph?.phases.find((p) => p.id === selectedPhase) ||
    graph?.phases.find((p) => p.id === active) ||
    graph?.phases[0];
  return (
    <section
      class="card inspection"
      id="inspection"
      aria-labelledby="inspection-title"
    >
      <div class="section-heading">
        <div>
          <span class="eyebrow">INTENT &amp; EVIDENCE</span>
          <h2 id="inspection-title">Decision inspector</h2>
        </div>
        <span class="muted" data-testid="inspection-freshness">
          {view.reason}
        </span>
      </div>
      <div class="inspection-summary">
        <label>
          Inspect goal{" "}
          <select
            aria-label="Inspect goal"
            value={state.selected}
            onChange={(e) => session.current?.select(e.currentTarget.value)}
          >
            <option value="">
              Automatic · current task or latest saved goal
            </option>
            {(data?.choices || []).map((g) => (
              <option key={g.id} value={g.id}>
                {humanize(g.kind)} · {humanize(g.status)} · {g.id.slice(0, 8)}
              </option>
            ))}
          </select>
        </label>
        <p data-testid="inspection-owner">
          <strong>Body: </strong>
          {view.fresh
            ? humanize(data?.body.owner)
            : "Live ownership unavailable"}
          {view.fresh && data?.body.task
            ? ` · ${data.body.task} ${data.body.taskId} · ${humanize(data.body.status)}`
            : ""}
        </p>
        {view.fresh && data?.body.phase && (
          <p data-testid="inspection-native-phase">
            Native phase: {humanize(data.body.phase)} ·{" "}
            {humanize(data.body.mode)}
          </p>
        )}
        {view.fresh && data?.body.reason && (
          <p class="inspection-reason">{data.body.reason}</p>
        )}
        {goal ? (
          <div data-testid="inspection-goal">
            <strong>
              {humanize(goal.kind)} · {humanize(goal.status)}
            </strong>
            <p>{goal.reason || "Saved objective"}</p>
            {goal.decision && <p>Host checkpoint: {goal.decision}</p>}
            <p class="muted">
              {goal.tasksLeft} tasks · {goal.attemptsLeft} attempts ·{" "}
              {Math.ceil(goal.remainingS)} s to deadline · waited{" "}
              {Math.round(goal.waitedS)} s
            </p>
            <p class="muted">
              {view.fresh && data.body.relatedGoal
                ? "This goal owns the current dispatched task."
                : "Saved goal; current body ownership is not established for this goal."}
            </p>
          </div>
        ) : (
          <p class="muted">No saved goal selected.</p>
        )}
      </div>
      <details class="inspection-detail">
        <summary>Phases &amp; guards</summary>
        {graph && (
          <div class="inspection-graph">
            <h3>
              {graph.label} · {saved ? "saved checkpoint" : "native routine"}
            </h3>
            <p class="muted">{graph.source}</p>
            <div
              class="inspection-phases"
              role="group"
              aria-label="Routine phases"
            >
              {graph.phases.map((p) => (
                <button
                  key={p.id}
                  type="button"
                  aria-pressed={phase?.id === p.id}
                  data-current={active === p.id ? "true" : "false"}
                  onClick={() => setSelectedPhase(p.id)}
                >
                  {p.label}
                  {active === p.id ? (saved ? " · saved" : " · current") : ""}
                </button>
              ))}
            </div>
            {phase && (
              <div class="inspection-phase">
                <h4>{phase.label}</h4>
                <p>{phase.description}</p>
                <ul>
                  {graph.edges
                    .filter((edge) => edge.from === phase.id)
                    .map((edge) => (
                      <li key={`${edge.from}-${edge.to}`}>
                        →{" "}
                        {graph.phases.find((p) => p.id === edge.to)?.label ||
                          edge.to}
                        : {edge.guard}
                      </li>
                    ))}
                </ul>
              </div>
            )}
            <p class="muted">
              Selecting a phase changes this view only. Guard values below
              describe the received snapshot, not a historical visit to the
              selected phase.
            </p>
          </div>
        )}
        <div class="inspection-table">
          <table>
            <caption>
              Guard evidence · necessary telemetry checks do not grant native
              admission
            </caption>
            <thead>
              <tr>
                <th>Check / layer</th>
                <th>Observed / source</th>
                <th>Required</th>
                <th>Result</th>
              </tr>
            </thead>
            <tbody>
              {(data?.guards || []).map((g) => (
                <tr key={g.code}>
                  <th scope="row">
                    {g.label}
                    <small>{g.layer}</small>
                  </th>
                  <td>
                    {g.observed}
                    <small>{g.source}</small>
                  </td>
                  <td>{g.required}</td>
                  <td>
                    <span
                      class={`inspection-result ${inspectionGuardResult(g, view)}`}
                    >
                      {humanize(inspectionGuardResult(g, view))}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </details>
      <details class="inspection-detail">
        <summary>Retained evidence &amp; coverage</summary>
        <div class="inspection-evidence">
          <p class="muted">
            Checkpoint evidence; this is not a complete causal decision history.
            Receipt rows without timestamps are not placed at invented times.
          </p>
          <ol>
            {(data?.evidence || []).map((e, i) => (
              <li key={`${e.kind}-${e.run}-${e.taskId}-${i}`}>
                <strong>{humanize(e.kind)}</strong> · {e.summary}
                <small>
                  {e.at ? new Date(e.at).toLocaleString() : "Time not recorded"}{" "}
                  · {e.taskId || "task not bound"} ·{" "}
                  {e.run || "run not recorded"}
                </small>
                {e.auditUrl && (
                  <a
                    href={e.auditUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    Open scoped task audit
                  </a>
                )}
              </li>
            ))}
          </ol>
          <ul class="muted">
            {(data?.gaps || []).map((gap) => (
              <li key={gap}>{gap}</li>
            ))}
          </ul>
          {data && (
            <p class="muted">
              Snapshot {new Date(data.at).toLocaleString()} · connection{" "}
              {data.connection} · state sequence{" "}
              {data.freshness.seq || "unavailable"} · contract {data.contract}
            </p>
          )}
          <p>
            <a
              href={`/api/inspection${state.selected ? `?goal=${encodeURIComponent(state.selected)}` : ""}`}
              target="_blank"
              rel="noopener noreferrer"
            >
              Read compact inspection JSON
            </a>{" "}
            · <a href="#map-pane">Live map</a> ·{" "}
            <a href="#camera">Live video</a>
          </p>
        </div>
      </details>
    </section>
  );
}
