// Reproducible local integration, deliberately separate from fixture scenarios.
// FFmpeg synthetic H264 -> pinned MediaMTX -> authenticated Go WHEP -> Chromium.
import { chromium, expect } from "@playwright/test";
import { spawn } from "node:child_process";
import { mkdir, open, writeFile, readFile } from "node:fs/promises";
import { resolve } from "node:path";

const out = resolve(
  process.env.RENKI_PROOF_CODEC === "vp9"
    ? ".checks/streaming/proof-vp9"
    : ".checks/streaming/proof",
);
await mkdir(out, { recursive: true });
const codec = process.env.RENKI_PROOF_CODEC === "vp9" ? "vp9" : "h264";
const sourceRecipe = codec === "vp9" ? "source-vp9" : "source";
const children = [];
const logs = [];
let browser, page;
const result = {
  kind: "SYNTHETIC ONLY — no OBS, Windows, game or Twitch",
  checks: [],
  screenshots: [],
  errors: [],
};
const origin = "http://127.0.0.1:18793";
const delay = (ms) => new Promise((r) => setTimeout(r, ms));
async function start(name, args) {
  const log = await open(resolve(out, name + ".log"), "w", 0o600);
  logs.push(log);
  const child = spawn("just", args, {
    cwd: resolve("."),
    detached: true,
    stdio: ["ignore", log.fd, log.fd],
  });
  children.push(child);
  return child;
}
async function stop(child) {
  if (!child || child.exitCode !== null) return;
  try {
    process.kill(-child.pid, "SIGINT");
  } catch {}
  for (let i = 0; i < 20 && child.exitCode === null; i++) await delay(100);
  if (child.exitCode === null)
    try {
      process.kill(-child.pid, "SIGKILL");
    } catch {}
}
async function ready(url, expected) {
  for (let i = 0; i < 100; i++) {
    try {
      const r = await fetch(url, {
        redirect: "manual",
        signal: AbortSignal.timeout(500),
      });
      if (r.status === expected) return;
    } catch {}
    await delay(200);
  }
  throw Error(`Local service did not become ready: ${url}; inspect proof logs`);
}
const action = async (path) => {
  const r = await fetch(origin + "/proof/" + path, {
    method: "POST",
    headers: { "X-Renki-Proof": "1" },
  });
  if (!r.ok) throw Error("Proof action failed");
};
try {
  // Refuse port conflicts; never attach to or stop an existing process.
  for (const port of [18793, 18889]) {
    try {
      await fetch(`http://127.0.0.1:${port}`, {
        signal: AbortSignal.timeout(300),
      });
      throw Error(`Port ${port} already in use`);
    } catch (e) {
      if (e.message.includes("already in use")) throw e;
    }
  }
  await start("relay", ["streaming", "relay"]);
  await ready("http://127.0.0.1:18889/renki/whep", 405);
  await start("host", ["streaming", "proof-host"]);
  await ready(origin + "/login", 200);
  browser = await chromium.launch({ headless: true, channel: "chromium" });
  page = await browser.newPage({
    viewport: { width: 1440, height: 1000 },
    reducedMotion: "reduce",
  });
  result.codecs = await page.evaluate(
    () => RTCRtpReceiver.getCapabilities("video").codecs,
  );
  result.codec = codec;
  if (!result.codecs.some((c) => c.mimeType.toLowerCase() === "video/" + codec))
    throw Error(
      `This Chromium build does not support ${codec}. Use just check streaming proof-vp9 for the explicit VP9 synthetic variant; verify H264 in a capable browser separately.`,
    );
  let sessions = 0;
  page.on("request", (r) => {
    if (r.url() === origin + "/api/video/whep" && r.method() === "POST")
      sessions++;
  });
  result.signaling = [];
  page.on("response", async (r) => {
    if (r.url().includes("/api/video"))
      result.signaling.push({
        path: new URL(r.url()).pathname,
        status: r.status(),
      });
    if (
      r.url() === origin + "/api/video/whep" &&
      r.status() !== 201 &&
      !result.upstreamDiagnostic
    ) {
      result.upstreamDiagnostic = { pending: true };
      try {
        const config = JSON.parse(
          await readFile(".checks/streaming/panel.local.json", "utf8"),
        );
        const diagnostic = await fetch("http://127.0.0.1:18889/renki/whep", {
          method: "POST",
          headers: {
            "Content-Type": "application/sdp",
            Authorization:
              "Basic " +
              Buffer.from("panel:" + config.password).toString("base64"),
          },
          body: r.request().postData(),
          signal: AbortSignal.timeout(8000),
        });
        result.upstreamDiagnostic = {
          status: diagnostic.status,
          body:
            diagnostic.status === 201
              ? "Diagnostic negotiation succeeded (SDP withheld)"
              : (await diagnostic.text()).slice(0, 500),
        };
      } catch (e) {
        result.upstreamDiagnostic = { error: e.message };
      }
    }
  });
  page.on("pageerror", (e) => result.errors.push(e.message));
  // Expected missing map/diagnostics API responses in this deliberately minimal
  // host are separate from actual browser runtime errors.
  await page.goto(origin + "/login");
  await page.locator("input[name=token]").fill("local-synthetic-proof-token");
  await page.locator("button[type=submit],button").first().click();
  await page.locator("#camera").scrollIntoViewIfNeeded();
  const video = page.locator(".video-stage video");
  const frames = () =>
    video.evaluate((v) => v.getVideoPlaybackQuality().totalVideoFrames);
  const pixels = () =>
    video.evaluate((v) => {
      const c = document.createElement("canvas");
      c.width = 64;
      c.height = 36;
      const ctx = c.getContext("2d");
      ctx.drawImage(v, 0, 0, 64, 36);
      return Array.from(ctx.getImageData(0, 0, 64, 36).data);
    });
  let slatePixels;
  const pictureDifference = async () => {
    const p = await pixels();
    return (
      p.reduce((sum, n, i) => sum + Math.abs(n - slatePixels[i]), 0) / p.length
    );
  };
  const screenshot = async (name) => {
    const path = resolve(out, name + ".png");
    await page.locator("#camera").screenshot({ path });
    result.screenshots.push(path);
  };
  // With no source, MediaMTX's prerecorded slate must already decode.
  await expect.poll(frames, { timeout: 20000 }).toBeGreaterThan(30);
  await expect(page.locator(".standby-slate")).toBeVisible();
  await screenshot("no-source-standby");
  result.checks.push(
    "No publisher: encoded original slate decodes while panel stays covered",
  );
  slatePixels = await pixels();
  let source = await start("source", ["streaming", sourceRecipe]);
  await action("source/on");
  await expect(page.locator('[data-video-state="gameplay"]')).toBeVisible({
    timeout: 20000,
  });
  await expect.poll(pictureDifference, { timeout: 15000 }).toBeGreaterThan(15);
  if (source.exitCode !== null)
    throw Error("Synthetic publisher exited; inspect source.log");
  const initialMotion = await pixels();
  await delay(500);
  const laterMotion = await pixels();
  if (!laterMotion.some((n, i) => n !== initialMotion[i]))
    throw Error("Synthetic picture is not moving");
  result.syntheticPictureDifference = await pictureDifference();
  const gameplayFrames = await frames();
  await screenshot("connected-synthetic");
  result.checks.push(
    "Authenticated same-origin WHEP delivers moving 1280x720 synthetic " +
      codec,
  );
  // Hide game immediately on bot scope loss, without any WHEP renegotiation.
  await action("offline");
  await expect(page.locator(".slate-reason")).toHaveText("Tas is resting", {
    timeout: 5000,
  });
  await screenshot("tas-resting");
  await expect.poll(frames).toBeGreaterThan(gameplayFrames + 30);
  await action("online");
  await expect(page.locator('[data-video-state="gameplay"]')).toBeVisible({
    timeout: 7000,
  });
  result.checks.push(
    "Bot offline/online preserves video and requires later fresh frames",
  );
  await action("source/off");
  await stop(source);
  await expect(page.locator(".standby-slate")).toBeVisible({ timeout: 6000 });
  const before = await frames();
  await delay(4000);
  await expect.poll(frames).toBeGreaterThan(before + 60);
  await expect.poll(pictureDifference, { timeout: 8000 }).toBeLessThan(10);
  result.returnedSlateDifference = await pictureDifference();
  // Capture the encoded feed itself, separate from the panel fallback. Video is
  // hidden for safety but canvas readback proves the relay segment's actual image.
  const encoded = await video.evaluate((v) => {
    const c = document.createElement("canvas");
    c.width = v.videoWidth;
    c.height = v.videoHeight;
    c.getContext("2d").drawImage(v, 0, 0);
    return c.toDataURL("image/png").split(",")[1];
  });
  await writeFile(
    resolve(out, "encoded-standby.png"),
    Buffer.from(encoded, "base64"),
  );
  result.screenshots.push(resolve(out, "encoded-standby.png"));
  await screenshot("publisher-down");
  source = await start("source-restart", ["streaming", sourceRecipe]);
  await action("source/on");
  await expect(page.locator('[data-video-state="gameplay"]')).toBeVisible({
    timeout: 15000,
  });
  await expect.poll(pictureDifference, { timeout: 15000 }).toBeGreaterThan(15);
  if (source.exitCode !== null) throw Error("Restarted publisher exited");
  await screenshot("publisher-restored");
  result.checks.push(
    "Publisher loss -> encoded slate -> publisher restart uses the same WHEP session",
  );
  if (sessions !== 1)
    throw Error(`Expected one continuous WHEP session, observed ${sessions}`);
  result.sessionPosts = sessions;
  result.decodedFrames = await frames();
  // Kill relay to prove a separate network outage covers the picture and retries.
  await action("source/off");
  await stop(children[0]);
  await expect(page.locator(".standby-slate")).toBeVisible({ timeout: 10000 });
  await screenshot("relay-reconnecting");
  await start("relay-restart", ["streaming", "relay"]);
  await expect.poll(() => sessions, { timeout: 20000 }).toBeGreaterThan(1);
  const restartedBaseline = await frames();
  await expect
    .poll(frames, { timeout: 12000 })
    .toBeGreaterThan(restartedBaseline + 30);
  result.recoveredSessionPosts = sessions;
  result.recoveredDecodedFrames = await frames();
  result.checks.push(
    "Relay outage is covered and bounded reconnect decodes video after relay recovery",
  );
  await page.setViewportSize({ width: 390, height: 844 });
  await page.locator("#camera").scrollIntoViewIfNeeded();
  await screenshot("mobile");
  if (result.errors.length)
    throw Error("Browser runtime errors; inspect result.json");
  result.status = "passed";
} catch (error) {
  if (page)
    try {
      result.browser = await page.evaluate(() => ({
        caption: document.querySelector(".video-caption")?.textContent,
        video: (() => {
          const v = document.querySelector("video");
          return v
            ? {
                readyState: v.readyState,
                paused: v.paused,
                error: v.error?.message,
                tracks: v.srcObject?.getTracks().map((t) => ({
                  kind: t.kind,
                  readyState: t.readyState,
                  muted: t.muted,
                })),
                frames: v.getVideoPlaybackQuality().totalVideoFrames,
              }
            : null;
        })(),
      }));
      await page.screenshot({ path: resolve(out, "failure.png") });
    } catch {}
  result.status = "failed";
  result.failure = error.stack;
  process.exitCode = 1;
} finally {
  await browser?.close();
  for (const child of [...children].reverse()) await stop(child);
  for (const log of logs) await log.close();
  await writeFile(
    resolve(out, "result.json"),
    JSON.stringify(result, null, 2) + "\n",
  );
  console.log(
    `${result.status.toUpperCase()}: synthetic video proof; ${result.checks.length} checks; ${resolve(out, "result.json")}`,
  );
  if (result.failure) console.error(result.failure.slice(0, 1800));
}
