import { scope } from "./status.js";

const sessionPath = /^\/api\/video\/session\/[0-9a-f-]{36}$/;
// Transport and content confidence have separate lifetimes. This class never
// stops/restarts media because Tas leaves a world; it only covers the picture.
export class VideoSession {
  constructor(emit, attach, env = {}) {
    this.env = {
      fetch: (...args) => fetch(...args),
      peer: () => new RTCPeerConnection({ iceServers: [] }),
      now: () => performance.now(),
      ...env,
    };
    this.emit = emit;
    this.attach = attach;
    this.state = { phase: "standby", reason: "Video is not configured" };
    this.generation = 0;
    this.attempts = 0;
    this.nextAttempt = 0;
    this.frames = 0;
    this.lastFrame = -Infinity;
    this.healthAt = -Infinity;
  }
  show(phase, reason) {
    this.state = { ...this.state, phase, reason, canRetry: this.attempts >= 5 };
    this.emit(this.state);
  }
  setStatus(status) {
    const next = scope(status);
    if (next !== this.scope) {
      this.scope = next;
      this.invalidate();
    }
    this.present();
  }
  invalidate() {
    this.verified = false;
    this.frames = 0;
    this.lastFrame = -Infinity;
  }
  present() {
    if (this.disposed) return;
    if (!this.scope) return this.show("standby", "Tas is resting");
    if (!this.verified || this.env.now() - this.healthAt > 2500)
      return this.show(
        "standby",
        this.healthReason || "Waiting for verified game capture",
      );
    if (this.attempts >= 5 && !this.pc)
      return this.show(
        "unavailable",
        "Video is unavailable. Try again when ready.",
      );
    if (!this.pc || this.pc.connectionState !== "connected")
      return this.show(
        this.attempts > 1 ? "reconnecting" : "connecting",
        "Connecting to video",
      );
    if (this.env.now() - this.lastFrame > 3000 || this.frames < 2)
      return this.show("reconnecting", "Waiting for fresh video");
    this.show("gameplay", "Video arriving · audio off");
  }
  frame() {
    if (this.verified && this.env.now() - this.healthAt <= 2500) {
      this.frames++;
      this.lastFrame = this.env.now();
    }
    this.present();
  }
  async tick() {
    if (this.disposed) return;
    this.present();
    if (
      this.pc &&
      this.env.now() - this.connectedAt > 8000 &&
      (this.pc.connectionState !== "connected" ||
        this.env.now() - this.transportFrame > 5000)
    )
      this.failed();
    if (this.polling) return;
    this.polling = true;
    const generation = this.generation;
    try {
      const response = await this.env.fetch("/api/video", {
        cache: "no-store",
        signal: AbortSignal.timeout(2000),
      });
      if (!response.ok) throw Error("Video status unavailable");
      const data = await response.json();
      if (this.disposed || generation !== this.generation) return;
      this.enabled = data.enabled;
      const verified =
        data.content === "gameplay" &&
        !!this.scope &&
        data.scope === this.scope;
      if (verified !== this.verified) this.invalidate();
      this.verified = verified;
      this.healthReason = data.enabled
        ? data.reason
        : "Video is not configured";
      this.healthAt = this.env.now();
      if (!data.enabled) this.close();
      else if (
        !this.pc &&
        !this.connecting &&
        this.attempts < 5 &&
        this.env.now() >= this.nextAttempt
      )
        this.connect();
    } catch {
      if (!this.disposed) {
        this.invalidate();
        this.healthReason = "Video status unavailable";
      }
    } finally {
      this.polling = false;
      this.present();
    }
  }
  async remove(path) {
    if (!sessionPath.test(path || "")) return;
    try {
      await this.env.fetch(path, {
        method: "DELETE",
        headers: { "X-Renki-Act": "video" },
        signal: AbortSignal.timeout(2000),
        keepalive: true,
      });
    } catch {
      /* ICE teardown also releases the viewer. */
    }
  }
  async connect() {
    this.connecting = true;
    this.attempts++;
    const generation = ++this.generation;
    this.connectedAt = this.transportFrame = this.env.now();
    this.connectAbort = new AbortController();
    const signal = this.connectAbort.signal;
    const timer = setTimeout(() => this.connectAbort?.abort(), 12000);
    try {
      const pc = (this.pc = this.env.peer());
      pc.addTransceiver("video", { direction: "recvonly" });
      pc.ontrack = (e) => {
        if (generation === this.generation)
          this.attach(e.streams[0] || new MediaStream([e.track]));
      };
      pc.onconnectionstatechange = () => {
        if (generation !== this.generation) return;
        if (["failed", "closed", "disconnected"].includes(pc.connectionState))
          this.failed();
        else this.present();
      };
      await pc.setLocalDescription(await pc.createOffer());
      await new Promise((resolve, reject) => {
        const finish = () => {
          pc.removeEventListener("icegatheringstatechange", check);
          signal.removeEventListener("abort", abort);
        };
        const check = () => {
          if (pc.iceGatheringState === "complete") {
            finish();
            resolve();
          }
        };
        const abort = () => {
          finish();
          reject(Error("Video negotiation timed out"));
        };
        pc.addEventListener("icegatheringstatechange", check);
        signal.addEventListener("abort", abort, { once: true });
        if (signal.aborted) abort();
        else check();
      });
      const response = await this.env.fetch("/api/video/whep", {
        method: "POST",
        headers: { "Content-Type": "application/sdp", "X-Renki-Act": "video" },
        body: pc.localDescription.sdp,
        signal,
      });
      if (response.status !== 201) throw Error("Video source unavailable");
      const path = response.headers.get("Location");
      if (!sessionPath.test(path || "")) throw Error("Invalid video session");
      if (generation !== this.generation || this.disposed) {
        this.remove(path);
        return;
      }
      this.session = path;
      await pc.setRemoteDescription({
        type: "answer",
        sdp: await response.text(),
      });
    } catch {
      if (generation === this.generation && !this.disposed) this.failed();
    } finally {
      clearTimeout(timer);
      if (generation === this.generation) this.connecting = false;
      this.present();
    }
  }
  decoded() {
    this.transportFrame = this.env.now();
    // Reset the failure budget only after sustained decoded video, rather than
    // an ICE connection that repeatedly produces no frames.
    if (this.env.now() - this.connectedAt > 15000) this.attempts = 0;
    this.frame();
  }
  close() {
    this.generation++;
    this.connectAbort?.abort();
    const pc = this.pc;
    this.pc = null;
    if (pc) {
      pc.onconnectionstatechange = null;
      pc.ontrack = null;
      pc.close();
    }
    this.remove(this.session);
    this.session = null;
    this.connecting = false;
    this.attach(null);
    this.frames = 0;
    this.lastFrame = -Infinity;
  }
  failed() {
    this.close();
    this.nextAttempt =
      this.env.now() +
      Math.min(8000, 1000 * 2 ** Math.max(0, this.attempts - 1));
    this.present();
  }
  retry() {
    this.attempts = 0;
    this.nextAttempt = 0;
    this.tick();
  }
  dispose() {
    this.disposed = true;
    this.close();
  }
}
