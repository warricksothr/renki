import { useEffect, useRef, useState } from "preact/hooks";
import { online, useStatus } from "./status.js";

const blank = () => ({
  name: "new base",
  center: [0, 0, 0],
  radius: 60,
  durationS: 600,
  travelM: 800,
  return: "home",
  sleeping: "",
  storage: [],
});
export function BaseSurvey() {
  const status = useStatus();
  const world = status?.world || "";
  const [open, setOpen] = useState(false);
  const [data, setData] = useState({ bases: [], active: "", surveys: [] });
  const [places, setPlaces] = useState([]);
  const [draft, setDraft] = useState(blank);
  const [expected, setExpected] = useState(null);
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const scope = useRef(world);
  scope.current = world;
  useEffect(() => {
    setData({ bases: [], active: "", surveys: [] });
    setPlaces([]);
    setDraft(blank());
    setExpected(null);
    setMessage("");
  }, [world]);
  useEffect(() => {
    if (!open || !world) return;
    const abort = new AbortController();
    let pending = false;
    const poll = async () => {
      if (pending || document.hidden) return;
      pending = true;
      try {
        const responses = await Promise.all([
          fetch(`/api/bases?world=${encodeURIComponent(world)}`, {
            signal: abort.signal,
          }),
          fetch(`/api/waypoints?world=${encodeURIComponent(world)}`, {
            signal: abort.signal,
          }),
        ]);
        for (const response of responses)
          if (!response.ok) throw new Error(await response.text());
        const [bases, waypoints] = await Promise.all(
          responses.map((r) => r.json()),
        );
        if (!abort.signal.aborted && scope.current === world) {
          setData(bases);
          setPlaces(waypoints.waypoints || []);
        }
      } catch (error) {
        if (!abort.signal.aborted) setMessage(error.message);
      } finally {
        pending = false;
      }
    };
    poll();
    const timer = setInterval(poll, 5000);
    return () => {
      abort.abort();
      clearInterval(timer);
    };
  }, [open, world]);
  const change = (key, value) => setDraft((d) => ({ ...d, [key]: value }));
  const edit = (name) => {
    const b = data.bases.find((b) => b.name === name);
    setDraft(b || blank());
    setExpected(b?.at || null);
    setMessage("");
  };
  const write = async (action) => {
    setBusy(true);
    setMessage("");
    try {
      const response = await fetch("/api/bases", {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-Renki-Act": "1" },
        body: JSON.stringify({
          action,
          world,
          name: draft.name,
          base: action === "save" ? draft : undefined,
          expectedAt: expected,
          id: data.surveys?.[0]?.id,
        }),
      });
      if (!response.ok) throw new Error(await response.text());
      const next = await response.json();
      if (scope.current !== world) return;
      setData(next);
      if (action === "save" || action === "select") {
        const saved = next.bases.find((b) => b.name === draft.name);
        setDraft(saved);
        setExpected(saved.at);
      }
      setMessage(
        action === "start"
          ? "Survey queued. Progress and gaps appear below."
          : action === "stop"
            ? "Survey stopped."
            : "Base settings saved. Survey starts only when requested.",
      );
    } catch (error) {
      if (scope.current === world) setMessage(error.message);
    } finally {
      setBusy(false);
    }
  };
  const markReturn = async () => {
    if (!online(status) || !status.pos) return;
    setBusy(true);
    try {
      const name = `${draft.name} return`.toLowerCase().trim();
      const old = places.find((p) => p.name === name);
      const response = await fetch("/api/waypoints", {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-Renki-Act": "1" },
        body: JSON.stringify({
          action: "set",
          world,
          name: old ? name : "",
          expectedAt: old?.at,
          waypoint: { name, kind: "place", pos: status.pos, radius: 0 },
        }),
      });
      if (!response.ok) throw new Error(await response.text());
      const next = await response.json();
      if (scope.current !== world) return;
      setPlaces(next.waypoints || []);
      change("return", name);
      setMessage(
        "Return place marked at Tas's position. Save the base to bind it.",
      );
    } catch (error) {
      setMessage(error.message);
    } finally {
      setBusy(false);
    }
  };
  const selectedBase = data.bases?.find((b) => b.name === data.active);
  const survey = data.surveys?.[0];
  const progress = survey?.survey;
  const active = ["ready", "queued", "running"].includes(survey?.status);
  const option = (p) => (
    <option value={p.name} key={p.name}>
      {p.name}
    </option>
  );
  return (
    <details id="base-survey" onToggle={(e) => setOpen(e.currentTarget.open)}>
      <summary>Base area &amp; familiarization</summary>
      {open && (
        <>
          <p class="place-help">
            Configure a boundary independently of sleeping, storage and return
            places. Radius {draft.radius} m means {draft.radius * 2} m across.
            Current world: {world || "unknown"}.
          </p>
          <div class="place-toolbar">
            <select
              aria-label="Base to edit"
              onChange={(e) => edit(e.target.value)}
              value={expected ? draft.name : ""}
            >
              <option value="">New base</option>
              {data.bases?.map((b) => (
                <option key={b.name} value={b.name}>
                  {b.name}
                  {b.name === data.active ? " · selected" : ""}
                </option>
              ))}
            </select>
            <button
              type="button"
              onClick={() => {
                setExpected(null);
                setDraft(blank());
              }}
            >
              New base
            </button>
          </div>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              write("save");
            }}
          >
            <div class="place-fields">
              <label>
                Base name{" "}
                <input
                  required
                  maxLength="40"
                  value={draft.name}
                  disabled={!!expected}
                  onInput={(e) => change("name", e.target.value)}
                />
              </label>
              {["X", "Y", "Z"].map((axis, i) => (
                <label key={axis}>
                  Center {axis}{" "}
                  <input
                    type="number"
                    required
                    step="0.1"
                    value={draft.center[i]}
                    onInput={(e) => {
                      const p = [...draft.center];
                      p[i] = Number(e.target.value);
                      change("center", p);
                    }}
                  />
                </label>
              ))}
              <label>
                Survey radius (m){" "}
                <input
                  type="number"
                  required
                  min="10"
                  max="250"
                  value={draft.radius}
                  onInput={(e) => change("radius", Number(e.target.value))}
                />
              </label>
              <label>
                Total time, including return (minutes){" "}
                <input
                  type="number"
                  required
                  min="2"
                  max="20"
                  value={draft.durationS / 60}
                  onInput={(e) =>
                    change("durationS", Number(e.target.value) * 60)
                  }
                />
              </label>
              <label>
                Total travel, including return (m){" "}
                <input
                  type="number"
                  required
                  min="50"
                  max="4000"
                  value={draft.travelM}
                  onInput={(e) => change("travelM", Number(e.target.value))}
                />
              </label>
              <label>
                Return place{" "}
                <select
                  required
                  value={draft.return}
                  onChange={(e) => change("return", e.target.value)}
                >
                  <option value="home">Saved home</option>
                  {places.map(option)}
                </select>
              </label>
              <label>
                Sleeping place{" "}
                <select
                  value={draft.sleeping}
                  onChange={(e) => change("sleeping", e.target.value)}
                >
                  <option value="">Unassigned · report candidates</option>
                  {places.map(option)}
                </select>
              </label>
              <label>
                Storage places{" "}
                <select
                  multiple
                  size="3"
                  value={draft.storage}
                  onChange={(e) =>
                    change(
                      "storage",
                      Array.from(e.target.selectedOptions, (o) => o.value),
                    )
                  }
                >
                  {places.filter((p) => p.kind === "storage").map(option)}
                </select>
              </label>
            </div>
            <div class="place-toolbar">
              <button
                type="button"
                disabled={!online(status) || !status.pos || busy}
                onClick={() => change("center", [...status.pos])}
              >
                Use Tas for center
              </button>
              <button
                type="button"
                disabled={!online(status) || !status.pos || busy}
                onClick={markReturn}
              >
                Mark return at Tas
              </button>
              <button disabled={busy || !world}>Save &amp; select base</button>
              {expected && (
                <button
                  type="button"
                  disabled={busy}
                  onClick={() => write("select")}
                >
                  Select saved base
                </button>
              )}
            </div>
          </form>
          <p class="place-help">
            Save places above to bind separate buildings. Saved radius and
            budgets apply to each survey; old-base markers and your edited
            places are protected. Learning a chest location does not inspect its
            contents.
          </p>
          <p class="place-help">
            Survey uses saved base {selectedBase?.name || "(none)"}
            {selectedBase
              ? ` · radius ${selectedBase.radius} m · ${selectedBase.durationS / 60} minutes · ${selectedBase.travelM} m travel`
              : ""}
            . Save edits before starting.
          </p>
          <div class="place-toolbar">
            <button
              type="button"
              disabled={busy || !online(status) || !data.active || active}
              onClick={() => write("start")}
            >
              Familiarize with {selectedBase?.name || "selected base"}
            </button>
            <button
              type="button"
              disabled={busy || !active}
              onClick={() => write("stop")}
            >
              Stop survey
            </button>
          </div>
          {progress && (
            <div aria-live="polite">
              <p>
                {survey.status} · {progress.phase} · returned:{" "}
                {progress.returned ? "yes" : "no"}
              </p>
              <p>
                Measured terrain: {progress.coverage.measured}/
                {progress.coverage.total} samples · new: {progress.coverage.new}{" "}
                · refreshed: {progress.coverage.refreshed} · unknown:{" "}
                {progress.coverage.unknown}
              </p>
              <p>
                Travel: {Math.round(progress.travel)} m · places added:{" "}
                {progress.places.added?.length || 0} · places refreshed:{" "}
                {progress.places.updated?.length || 0} · protected:{" "}
                {progress.places.protected?.length || 0} · recorded legs:{" "}
                {progress.legs.length} · blocked legs:{" "}
                {progress.legs.filter((l) => l.status !== "succeeded").length}
              </p>
              <p>{survey.reason}</p>
              {progress.unresolved?.length > 0 && (
                <p>Unresolved places: {progress.unresolved.join(", ")}</p>
              )}
            </div>
          )}
          <p role="status">{message}</p>
        </>
      )}
    </details>
  );
}
