# Broadcasting Renki's game view

**Local implementation update:** [LOCAL-VIDEO.md](LOCAL-VIDEO.md) supersedes the
local-viewing, standby and outage-lifecycle parts of this original research.
The loopback slice adds authenticated WHEP/player, shared original slate and
local relay tooling. [WINDOWS-VIDEO.md](WINDOWS-VIDEO.md) documents the subsequent
OBS adapter, real game capture and private HTTPS deployment. External destinations
remain future work. The text below records the PR #112 research baseline.

Proposed design, researched 2026-10-30 against base `3ccd07d` (PR #109).
No VM, account, capture setting or service was accessed or changed. This branch
adds offline configuration/diagnostic tooling only. OBS installation, a running
adapter, an overlay and Twitch output remain future work with operator setup.

## Recommendation and alternatives

Use **OBS Studio on the gaming VM**, in the same interactive Windows desktop
and GPU as Valheim. Start with a specific-window Game Capture source, a static
offline slate, H.264 CBR output at **1280×720, 30 fps, 3000 Kbps, 2-second
keyframes**, and audio off. Prefer a hardware H.264 encoder if the VM actually
exposes one with adequate headroom. Treat these as starting targets for a local
recording benchmark, not verified BLIGHT capabilities or Twitch certification.
Use the built-in Twitch service configuration when broadcasting is authorized.
OBS 28+ includes obs-websocket; do not install an additional websocket plugin.

- **OBS + obs-websocket v5:** Windows Game Capture is intended for DirectX/OpenGL
  games and avoids the diagnostic path's CPU JPEG encoding. OBS owns composition,
  scaling, encoding, Twitch output/reconnect and browser sources. A small local
  adapter can expose bounded controls and diagnostics without putting broadcast
  dependencies in the game plugin. Game hooking, minimized rendering, GPU access,
  session changes and encoder support still need deployment checks. Prefer this
  over maintaining a second compositor and encoder supervisor.
- **FFmpeg:** a good smaller service when the exact capture device/encoder and
  a fixed composition have already been proven. Windows `gdigrab` supports
  window-title capture; D3D11 `ddagrab` is a filter-based desktop-duplication option
  in suitable builds, not a substitute for game hooking. Direct desktop capture
  risks unrelated windows; GDI may be slow or black for GPU games. HTML/map
  composition, audio routing, process/window reacquisition, crash recovery,
  reconnect, rate diagnostics and secret-safe ingest need additional tooling.
  Hardware encode depends on the FFmpeg build and VM driver too. Keep it as a
  fallback after capture benchmarking, not an automatic emergency switch.
- **Browser compositor:** a dedicated Preact/canvas overlay fits OBS Browser
  Source. A browser alone cannot silently acquire the Windows game view: screen
  capture uses permission/session-dependent APIs, and a browser-to-Twitch path
  also needs encoding and an ingest bridge. Animating the current JPEG feed in a
  canvas at 30 fps still repeats old images. A separate WebRTC/game-video feed
  would be a new media system. Use a browser for the optional overlay only.

Do not use Display Capture as an automatic fallback. Select Window Capture
explicitly if Game Capture fails approved local testing. Restrict matching to
Valheim's executable/window and test title changes after relaunch; never select
"any fullscreen application" or whichever window is foreground. Source setting
keys and encoder IDs vary by OBS build: inspect locally before implementing
configuration writes rather than fabricating portable identifiers.

## What exists today

`plugin/Renki.Plugin/CameraReporter.cs` captures the rendered framebuffer at end
of frame, including the HUD, downsizes on GPU, then performs `ReadPixels` and
JPEG encoding. `internal/protocol/screenshot.go` defaults to 5-second captures,
960-pixel width, quality 70, disabled until enabled. Bounds include 1–300 seconds,
1280×720 output, 512 KiB/frame, one capture in flight and a native ceiling of
one capture start/second. A latest-only bridge slot prioritizes control traffic.

This is visual telemetry for an agent, **not a video/audio transport**. At the
default cadence, five seconds of action are missing between images; even the
maximum 1 fps cannot show smooth movement. Making it a stream would add bridge,
GPU readback and CPU/JPEG costs while losing motion/audio. Leave it unchanged
for diagnosis and stream actual game frames separately. OBS captures the same
rendered third-person/HUD view the bot uses, not an invented first-person view.

`GET /api/camera` and `/api/camera.jpg` are authenticated, memory-only and scoped
to the current connection/world; old pinned images return 409. The Preact
`CameraSession` rejects asynchronous responses from an older generation and
clears images on logout/world changes. Retain that discipline in future overlay
data. A screenshot cannot prove task success or OBS capture health.

`just vm map-snapshot` assembles private scoped map/atlas data and the production
renderer into read-only HTML. It is useful for operator diagnosis, but the bundle
contains world identity, positions, named markers and player/container knowledge.
It is not a public broadcast asset. Existing UI automation is described in
[PANEL-UI-TESTING.md](PANEL-UI-TESTING.md); reuse its fixture discipline and renderer
only after introducing a public projection. This proposal does not depend on
the parallel panel layout branch.

## Scenes, recovery and a separate read-only overlay

Create an OBS scene collection dedicated to broadcasting, with these stable
names (the offline tool uses the capture source name):

- `Renki Offline`: static safe slate with no desktop, game audio or panel data.
- `Renki Gameplay`: `Renki Game Capture`, optionally game-only audio, and a future
  `Renki Overlay` Browser Source. Layer a slate behind capture as a visual aid;
  an opaque black/frozen capture can hide it, so this is not health detection.

OBS and the future adapter should start on the offline slate, output stopped.
Run OBS via a dedicated interactive logon task, separate from `RenkiStart` and
`RenkiPanel`, so a game restart need not restart the encoder. Do not attach this
to current trial cleanup or deploy recipes. OBS must not be launched directly
in an SSH/noninteractive session. [VM.md](VM.md) documents the persistent desktop,
display/dummy-plug and RDP-to-console requirements already needed by Valheim.

The future adapter should track explicit states `offline`, `waiting-capture`,
`ready`, `reconnecting`, `fault`, and owner-controlled broadcast intent. Default
to the slate after startup, controller loss, logout, connection/world change or
capture failure. Clear the overlay immediately on scope change; never carry a
cached map from the previous world into the next one. Require matching current
scope plus several successful capture checks before returning to gameplay.
Reacquire the approved executable after a restart with bounded backoff. After
repeated failures, remain on the slate and report a fixed reason; never restart
the game or broaden capture automatically. Stream network reconnect is separate
from game availability; keeping the slate on air briefly is reasonable. Game downtime should preserve output on the slate; only explicit output intent
or a separate destination/output failure should affect that session. Restarting OBS must not silently
restore a previously running stream: the operator must re-arm the session.

`GetSourceActive` only reports whether a source is in Program or showing in OBS
UI. `GetStats.activeFps` reports OBS rendering, not incoming game frames. Neither
proves a healthy game hook, and a still image can be a legitimate idle game.
Use a dedicated private capture probe/local preview and process/window presence
alongside controller scope and capture evidence. Do not count the slate/overlay
as a game frame. Low-rate `GetSourceScreenshot` checks may help diagnose blank
capture but still cannot conclusively distinguish frozen from motionless play;
they must not enter ordinary logs or public responses. Before unattended
operation, prove a reliable freshness signal for the chosen capture method or
require operator re-arming after failure. This is an unresolved prerequisite,
not functionality implemented here.

The overlay should be a **separate loopback listener and browser entry point**,
not `/` on the authenticated control panel and not a CSS-hidden control panel.
A projection inside the controller can emit only public enum status, observation
age and optionally approved local terrain. Start with status only. Do not proxy
`/api/status`, audit events, chat, inventory, player profiles or raw map exports.
The overlay server must have no game command handlers or credential access.
Use GET-only resources, strict Host/origin handling and CSP, no remote fonts or
analytics, OBS Browser Source page permissions set to no access, and no panel login cookie/token in
the page URL, storage, HTML or JS. Bind to loopback in the VM; a remotely hosted
compositor would need separate transport/auth design. There is no overlay
listener or public endpoint in this slice.

An optional map must be opt-in and built from an explicit allowlist: approved
terrain around an anonymous centered bot, no world UID/name, player names,
absolute coordinates, owners, chest contents, graves or named locations. Even
terrain silhouettes can identify a base; offer cropping, delay and disabling it,
and seek owner agreement on what may be broadcast. Status should be fixed
categories such as moving/working/offline rather than task arguments or free
text. Hide stale dynamic content and label retained terrain as historical.
Future UI acceptance needs fixture-only desktop/mobile/OBS-size screenshots,
scope changes, stale/fetch errors and proof that forbidden fields and mutation
routes are absent. No UI was added in this branch, so no new UI screenshots are
claimed.

## Agent control and secret boundaries

Use an authenticated **local adapter** as the only agent-facing OBS client.
OBS's websocket password grants broad OBS control, not scoped read-only access;
do not hand it to the game plugin, overlay or ordinary agent tools. Enable
authentication and restrict TCP 4455 to local access using verified listener
configuration/firewall rules; do not assume the default server is loopback-only
or publish it through Tailscale. A later remote agent can invoke the bounded
adapter via the existing SSH trust boundary without receiving OBS credentials.

Suggested future contract (not live commands in this PR):

- `status`: fixed states, last safe reason, observation age, actual video
  settings and interval deltas for bytes/frame counters; no raw OBS response.
- `prepare(profile)`: validate, select offline slate, check capabilities and
  require stopped streaming/recording/virtual-camera outputs before changing
  video settings. Encoder/audio/source mappings are local, verified setup.
- `arm`/`start`: only an owner with a session-scoped broadcast capability can
  authorize going live after privacy and capture preflight. Agents have status
  and preview rights by default; game-control permission does not grant stream
  start rights. Avoid toggle operations, which are unsafe to retry.
- `scene(offline|gameplay)` and `stop`: idempotent desired state, with gameplay
  gated on health/arming. Independent emergency stop remains available to the
  owner. Reconcile bounded request IDs/timeouts against actual output state
  before retrying; do not treat an OBS acknowledgement as delivery to Twitch.

For websocket v5, negotiate Hello/Identify RPC 1 with authentication, subscribe
only to required state events, discover `GetVersion.availableRequests`, and
handle disconnects. `GetStreamStatus` and `GetStats` support useful counters;
calculate deltas over time and reset baselines after reconnect/restart. OBS
render lag, encoding lag, network reconnect and blank game capture are different
failures. `StartStream`, `StopStream`, `SetCurrentProgramScene` and
`SetVideoSettings` exist, but this PR never sends them. General encoder writes
are not assumed portable across advanced/simple output modes or hardware.

Keep Twitch account linking/stream keys in OBS's per-user configuration, secured
with the Windows account/file ACLs. Prefer owner-operated account setup. Keep
the websocket password in a protected local secret store/file readable only by
the adapter identity. Neither secret belongs in repo, CLI arguments, process
logs, tickets, screenshots or URLs. Never fetch/log `GetStreamServiceSettings`,
which may contain a stream key. Do not dump OBS config, input settings, raw
websocket errors or browser-source URLs into agent diagnostics. Local OBS logs
and private probes still need access control and retention limits.

Privacy also applies to actual video/audio: Valheim HUD/chat, nameplates,
server/login screens, Steam notifications and other players' voice can expose
information even when the overlay is safe. Review a private recording, crop or
disable those surfaces where practical, default microphone/desktop audio off,
and allow game-only audio only after verifying app-specific routing. Do not
capture an authenticated panel window. Streaming permission and other players'
privacy decisions remain operator prerequisites.

## Offline implementation and checks

`broadcast.example.json` is a secret-free target profile, not an importable OBS
scene collection. `scripts/broadcast.py` uses only Python's standard library and
does not connect to anything, execute subprocesses, read environment secrets,
write settings or start OBS. Use:

```
just check broadcast-test
just broadcast-plan
just broadcast-plan --profile broadcast.example.json --response scripts/fixtures/broadcast/obs-response.json
```

For local experimentation, copy the example to `broadcast.local.json` (already
gitignored) and use `just broadcast-plan --profile broadcast.local.json`.
Fields are required and unknown fields rejected at every level. Version 1
accepts 720p/1080p at 30/60 fps, video bitrate 2500–6000 Kbps, fixed 2-second
keyframes, AAC 96–160 Kbps, game/window capture, hardware/software H.264 preference,
audio off/game-only and an overlay boolean. These deliberately limited **project
bounds** are not a promise of ingest, encoder, audio or overlay availability.
There is no location to put a token/key/URL or arbitrary OBS setting. The tool
cannot verify actual bitrate/encoder/keyframes from the diagnostic batch.

The plan contains recommended settings and an inert v5 RequestBatch (op 8)
with only five read requests. A future transport may send it only **after**
authentication/capability negotiation; it is not a connection implementation.
The optional `--response` consumes a local synthetic RequestBatchResponse (op 9)
with matching batch/request IDs and types. It emits fixed labels, finite numbers
and booleans, excluding error comments, version/platform strings and unknown
payload fields. `captureHealth` remains `unverified` even if the source is in
Program. Exit 0 means the local file was valid, including responses containing
OBS request failures; automation must inspect each `ok` and the actual states.
Exit 2 means validation/input failure, with a bounded error excluding input data
and paths. Do not save raw live response envelopes in the repo or check logs.

`just check ci` includes `broadcast-test`, covering target bounds/type rejection,
unknown/secret fields, duplicate/malformed/oversized JSON, correlation errors,
reconnect/video mismatch/missing capability, failed OBS requests, output
allowlisting and no network/process use. All fixture observations are synthetic.
The tests do not prove OBS compatibility on BLIGHT or successful Twitch output.

## Operator prerequisites and later acceptance

Before a separately authorized deployment:

1. Identify the guest Windows/OBS versions, actual passed-through GPU/driver,
   hardware H.264 encoder availability/session limits and headroom. Confirm a
   persistent console desktop/display; record whether RDP, lock, disconnect,
   minimization and privilege differences break capture. Do not infer NVENC
   from "has a GPU". Software x264 competes with the game/controller CPU.
2. Install OBS in that desktop, create only the dedicated scenes/sources and
   prove Game Capture, then Window Capture if necessary. Do a local recording
   at 720p30 before any account access. Benchmark game/task latency, render and
   encode skipped-frame deltas, CPU/GPU load, disk and output size. Cap game FPS
   if approved to leave GPU headroom; do not promise hardware encoding removes
   GPU contention. Test the unchanged diagnostic camera concurrently.
3. Exercise game exit/relaunch, logout, world change, stale controller, missing
   overlay, OBS crash, GPU/session loss and network outage. Verify slate privacy,
   source reacquisition and bounded recovery. Resolve the capture-health signal
   before enabling unattended gameplay transitions.
4. Implement/test the local adapter and status-only overlay with synthetic
   fixtures first. Validate allowlist/privacy, no browser controls, no secret in
   errors, unreachable websocket from other hosts, ownership/arming and stop.
5. An owner configures Twitch credentials locally and checks current ingest
   guidance, account permission, sustained outbound bandwidth plus headroom and
   service-selected ingest. A later explicitly authorized bandwidth/ingest test
   and playback review must confirm bitrate, keyframes, resolution, motion,
   reconnect, audio/privacy and latency. Do not assume local recording proves
   remote delivery or automatic transcoding. Adjust bitrate/resolution/fps only
   between sessions with measured evidence.

No part of that list is complete merely because the offline tests pass.

## Sources

Primary documentation read 2026-10-30; links describe platform capabilities,
not measurements of the gaming VM:

- [OBS Game Capture](https://obsproject.com/kb/game-capture-source) and
  [Window Capture](https://obsproject.com/kb/window-capture-sources): source
  selection, matching behavior and capture tradeoffs.
- [OBS Browser Source](https://obsproject.com/kb/browser-source): local/URL
  sources, viewport/FPS, source lifecycle and page permissions.
- [OBS encoding performance](https://obsproject.com/kb/encoding-performance-troubleshooting):
  rendering/encoding resource contention.
- [obs-websocket README](https://github.com/obsproject/obs-websocket/blob/master/README.md)
  and [v5 protocol](https://github.com/obsproject/obs-websocket/blob/master/docs/generated/protocol.md):
  included since OBS 28, authentication, default port 4455, RPC/batches and the
  exact diagnostic fields used by the tool. Revalidate against the installed
  version; master documentation can evolve.
- [FFmpeg input devices](https://ffmpeg.org/ffmpeg-devices.html#gdigrab) and
  [ddagrab filter](https://ffmpeg.org/ffmpeg-filters.html#ddagrab): Windows capture
  choices and build requirements.
- [Twitch broadcast ingestion](https://dev.twitch.tv/docs/video-broadcast/):
  broadcasting/ingest and stream-key trust boundary.
  [Broadcasting guidelines](https://help.twitch.tv/s/article/broadcasting-guidelines?language=en_US)
  returned a client-rendered help shell in this research environment; its current
  numerical recommendations were not verified. Check the rendered current page
  before account setup; the example limits above are conservative project policy.
