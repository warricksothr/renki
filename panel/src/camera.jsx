import { useEffect, useRef, useState } from "preact/hooks";
import { useStatus, getStatus, subscribeStatus, scope } from "./status.js";
import { CameraSession, cameraPresentation } from "./camera.js";

export function Camera() {
  const status = useStatus();
  const [state, setState] = useState({
    phase: "loading",
    message: "Reading camera settings…",
  });
  const [intervalValue, setIntervalValue] = useState("5");
  useEffect(() => {
    if (state.config) setIntervalValue(String(state.config.intervalS));
  }, [state.config?.intervalS]);
  const [now, setNow] = useState(Date.now());
  const session = useRef();
  useEffect(() => {
    const camera = new CameraSession(setState);
    session.current = camera;
    camera.setStatus(getStatus());
    const unsubscribe = subscribeStatus((s) => {
      const previous = scope(camera.status);
      camera.setStatus(s);
      if (scope(s) !== previous) camera.poll();
    });
    const tick = () => {
      setNow(Date.now());
      if (!document.hidden) camera.poll();
    };
    const timer = setInterval(tick, 1000);
    document.addEventListener("visibilitychange", tick);
    camera.poll();
    return () => {
      unsubscribe();
      clearInterval(timer);
      document.removeEventListener("visibilitychange", tick);
      camera.dispose();
    };
  }, []);
  const presentation = cameraPresentation(state, now);
  const imageVisible = state.url && state.scope === scope(status);
  return (
    <section class="card camera" id="camera" aria-labelledby="camera-title">
      <div class="section-heading">
        <div>
          <span class="eyebrow">THROUGH TAS’S EYES</span>
          <h2 id="camera-title">Game camera</h2>
        </div>
        <span
          class={`camera-badge ${presentation.label === "stale" || state.phase === "failed" ? "warning" : ""}`}
        >
          {presentation.label}
        </span>
      </div>
      <div class={`camera-stage ${presentation.stale ? "stale" : ""}`}>
        {imageVisible && (
          <img
            key={state.url}
            id="camera-image"
            src={state.url}
            alt={`Game scene captured at ${new Date(state.frame.capturedAt).toLocaleTimeString()}`}
            onLoad={(e) =>
              session.current.loaded(state.url, e.currentTarget.naturalWidth)
            }
            onError={() =>
              state.url === session.current.state.url &&
              session.current.fail("The camera image failed to load. Retrying…")
            }
          />
        )}
        {(!imageVisible || !state.loaded || presentation.stale) && (
          <div class="camera-placeholder">
            <span class="camera-glyph" aria-hidden="true">
              ▣
            </span>
            <strong>{presentation.message}</strong>
            {state.phase === "disabled" && (
              <span>Images are disabled by default.</span>
            )}
            {state.phase === "offline" && (
              <span>The image clears when Tas leaves the world.</span>
            )}
          </div>
        )}
      </div>
      <div class="camera-caption" id="camera-status" role="status">
        <span>{presentation.message}</span>
        {state.frame && (
          <span>
            Received {presentation.receiptAge?.toFixed(1)}s ago · captured{" "}
            {presentation.captureAge?.toFixed(1)}s ago at{" "}
            {new Date(state.frame.capturedAt).toLocaleTimeString()}
          </span>
        )}
      </div>
      <div class="camera-controls">
        <label class="switch-label">
          <input
            id="camera-enabled"
            type="checkbox"
            checked={!!state.config?.enabled}
            disabled={!state.config || state.saving}
            onChange={(e) =>
              session.current.setting(
                "screenshot.enabled",
                e.currentTarget.checked,
              )
            }
          />{" "}
          Capture images
        </label>
        <form
          class="camera-frequency"
          onSubmit={(event) => {
            event.preventDefault();
            if (event.currentTarget.reportValidity())
              session.current.setting(
                "screenshot.intervalS",
                Number(intervalValue),
              );
          }}
        >
          <label>
            Every{" "}
            <input
              id="camera-interval"
              type="number"
              min="1"
              max="300"
              step="1"
              value={intervalValue}
              disabled={!state.config || state.saving}
              onInput={(event) => setIntervalValue(event.currentTarget.value)}
            />{" "}
            seconds
          </label>
          <button
            disabled={
              !state.config ||
              state.saving ||
              Number(intervalValue) === state.config.intervalS
            }
          >
            Apply
          </button>
        </form>
      </div>
      {state.notice && (
        <p class="camera-notice" role="status">
          {state.notice}
        </p>
      )}
      <details class="camera-details">
        <summary>Capture details</summary>
        <p>
          {state.frame
            ? `${state.frame.width} × ${state.frame.height} · ${state.frame.captureMs?.toFixed(1)} ms capture · ${state.frame.task || "idle"} · sequence ${state.frame.seq}`
            : "No current image metadata."}{" "}
          Image delivery is visual context; task receipts confirm outcomes.
        </p>
      </details>
    </section>
  );
}
