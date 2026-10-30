import { online, scope } from "./status.js";

// One scoped image lease. Every asynchronous boundary checks the generation;
// an old connection, world or setting response can never repopulate the view.
export class CameraSession {
  constructor(emit, env = {}) {
    this.env = {
      fetch: globalThis.fetch.bind(globalThis),
      createURL: (b) => URL.createObjectURL(b),
      revokeURL: (u) => URL.revokeObjectURL(u),
      now: () => Date.now(),
      ...env,
    };
    this.emit = emit;
    this.state = {
      config: null,
      phase: "loading",
      message: "Reading camera settings…",
      scope: "",
    };
    this.generation = 0;
    this.status = null;
    this.busy = false;
    this.saving = false;
  }
  update(patch) {
    this.state = { ...this.state, ...patch };
    this.emit(this.state);
  }
  clear(patch = {}) {
    if (this.state.url) this.env.revokeURL(this.state.url);
    this.update({
      url: null,
      imageKey: null,
      frame: null,
      loaded: false,
      ...patch,
    });
  }
  setStatus(status) {
    const next = scope(status);
    const changed = next !== scope(this.status);
    this.status = status;
    if (changed || (!online(status) && this.state.phase !== "offline")) {
      this.generation++;
      this.abort?.abort();
      this.busy = false;
      this.clear({
        scope: next,
        phase: next ? "pending" : "offline",
        message: next
          ? "Waiting for a frame from this world…"
          : "Bot offline · no current scene",
      });
    }
  }
  fail(message) {
    this.clear({ phase: "failed", message });
  }
  loaded(url, width) {
    if (url !== this.state.url) return;
    if (width > 0) this.update({ loaded: true });
    else this.fail("The camera image could not be decoded. Retrying…");
  }
  async poll() {
    if (this.busy || this.saving) return;
    this.busy = true;
    const generation = this.generation;
    const abort = new AbortController();
    this.abort = abort;
    const timeout = setTimeout(() => abort.abort(), 8000);
    const valid = () => generation === this.generation;
    try {
      const response = await this.env.fetch("/api/camera", {
        cache: "no-store",
        signal: abort.signal,
      });
      if (response.status === 401)
        throw Error("Panel session expired. Sign in again.");
      if (!response.ok) throw Error("Camera status unavailable. Retrying…");
      const data = await response.json();
      if (!valid()) return;
      this.update({ config: data.config });
      if (!online(this.status))
        return this.clear({
          phase: "offline",
          message: "Bot offline · no current scene",
        });
      if (!data.config?.enabled)
        return this.clear({
          phase: "disabled",
          message: "Capture is off · enable it below to see the game",
        });
      if (!data.available)
        return this.clear({
          phase: "pending",
          message: "Capture enabled · waiting for the first frame",
        });
      if (
        scope({ ...data.frame, connected: true, phase: "in_world" }) !==
        scope(this.status)
      )
        return this.clear({
          phase: "pending",
          message: "Waiting for a frame from this world…",
        });
      // A numeric receipt age comes from the controller, independent of clock skew.
      const received = this.env.now() - Math.max(0, data.ageS || 0) * 1000;
      if (data.imageUrl !== this.state.imageKey) {
        const image = await this.env.fetch(data.imageUrl, {
          cache: "no-store",
          signal: abort.signal,
        });
        if (!valid()) return;
        if (image.status === 409)
          return this.clear({
            phase: "pending",
            message: "A newer frame arrived · loading on the next refresh",
          });
        if (!image.ok) throw Error("Camera image unavailable. Retrying…");
        const blob = await image.blob();
        if (!valid()) return;
        const url = this.env.createURL(blob);
        this.clear({
          url,
          imageKey: data.imageUrl,
          frame: data.frame,
          received,
          scope: scope(this.status),
          phase: "image",
          message: "",
          loaded: false,
        });
      } else
        this.update({
          frame: data.frame,
          received,
          phase: "image",
          message: "",
        });
    } catch (error) {
      if (valid())
        this.fail(
          error.name === "AbortError"
            ? "Camera request timed out. Retrying…"
            : error.message,
        );
    } finally {
      clearTimeout(timeout);
      if (valid()) this.busy = false;
    }
  }
  async setting(key, value) {
    if (this.saving) return;
    this.generation++;
    this.abort?.abort();
    this.busy = false;
    this.saving = true;
    this.clear({
      phase: "pending",
      message: "Saving camera setting…",
      saving: true,
    });
    const generation = this.generation;
    try {
      const response = await this.env.fetch("/api/setting", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Renki-Act": "camera",
        },
        body: JSON.stringify({ key, value }),
        signal: AbortSignal.timeout(8000),
      });
      if (!response.ok)
        throw Error((await response.text()) || "Camera setting refused");
      if (generation === this.generation)
        this.update({ notice: "Camera setting saved." });
    } catch (error) {
      this.update({ notice: `Could not save: ${error.message}` });
    } finally {
      this.saving = false;
      this.update({ saving: false });
      await this.poll();
    }
  }
  dispose() {
    this.generation++;
    this.abort?.abort();
    this.clear();
  }
}
export function cameraPresentation(state, now) {
  if (state.phase !== "image")
    return { label: state.phase, message: state.message };
  const receiptAge = Math.max(0, (now - state.received) / 1000);
  const captureAge = Math.max(
    0,
    (now - Date.parse(state.frame.capturedAt)) / 1000,
  );
  const stale =
    Math.max(receiptAge, captureAge) >
    Math.max(15, (state.config?.intervalS || 5) * 3);
  return {
    label: stale ? "stale" : state.loaded ? "latest frame" : "loading image",
    receiptAge,
    captureAge,
    stale,
    message: stale
      ? "No recent frame · this scene is out of date"
      : state.loaded
        ? "Latest received scene · periodic capture"
        : "Downloading and decoding image…",
  };
}
