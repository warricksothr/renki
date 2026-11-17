import { useEffect, useState } from "preact/hooks";
import { MapWorkspace } from "./workspace.jsx";
import { Camera } from "./camera.jsx";

export function Viewer() {
  return (
    <>
      <header class="topbar">
        <span class="brand">renki · live view</span>
        <span id="link" class="chip">
          Connecting…
        </span>
        <span id="who" />
        <span id="viewer-session">Read-only · map &amp; video</span>
      </header>
      <main id="workspace" class="viewer">
        <div class="visual-workspace">
          <MapWorkspace />
          <Camera readOnly />
        </div>
      </main>
    </>
  );
}

export async function viewerLogin() {
  const token = new URLSearchParams(location.hash.slice(1)).get("token");
  if (!token) return true;
  history.replaceState(null, "", location.pathname);
  try {
    const response = await fetch("/api/viewer/login", {
      method: "POST",
      headers: { "Content-Type": "application/json", "X-Renki-Act": "viewer" },
      body: JSON.stringify({ token }),
    });
    if (response.ok) return true;
  } catch {
    /* Show an actionable message without leaking the credential. */
  }
  document.getElementById("app").textContent =
    "Viewer access unavailable. Ask the operator for a new link.";
  return false;
}

export function Sharing({ active }) {
  const [state, setState] = useState(null),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  const load = async (action) => {
    setBusy(true);
    setError("");
    try {
      const response = await fetch(
        "/api/sharing",
        action
          ? {
              method: "POST",
              headers: {
                "Content-Type": "application/json",
                "X-Renki-Act": "sharing",
              },
              body: JSON.stringify({ action }),
            }
          : { cache: "no-store" },
      );
      if (!response.ok) throw Error("Viewer sharing is unavailable.");
      setState(await response.json());
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  useEffect(() => {
    if (active) load();
  }, [active]);
  const url = state?.token
    ? `${location.origin}/view#token=${encodeURIComponent(state.token)}`
    : "";
  return (
    <section class="card" id="sharing">
      <h2>Map &amp; video sharing</h2>
      <p>
        The viewer can watch the map and video. It cannot control Tas or read
        settings, inventory, chat or decision history. Recipients need access to
        this panel’s network address.
      </p>
      <p>
        Anyone with this link can view. Rotate or disable access to revoke it.
        An existing media connection may keep playing until it disconnects.
      </p>
      {url && (
        <label>
          Viewer link
          <input
            aria-label="Viewer link"
            class="viewer-link"
            value={url}
            readOnly
            onFocus={(e) => e.currentTarget.select()}
          />
        </label>
      )}
      <div class="place-toolbar">
        <button disabled={busy} onClick={() => load("rotate")}>
          {url ? "Rotate viewer link" : "Enable viewer link"}
        </button>
        <button disabled={busy || !url} onClick={() => load("disable")}>
          Disable viewer access
        </button>
        {url && (
          <a href={url} target="_blank" rel="noopener">
            Open viewer
          </a>
        )}
      </div>
      <p role="status">
        {error ||
          (state
            ? state.enabled
              ? "Viewer link enabled"
              : "Viewer sharing disabled"
            : "")}
      </p>
    </section>
  );
}
