# Local game video and standby

This is a local proof of concept stacked above PR #112. It adds native browser
video to the existing `Camera` component without depending on layout PR #110.
No recipe deploys to the gaming VM, starts Valheim, contacts Twitch or configures
an external output. Audio is off. Diagnostic JPEG/map APIs remain independent.

## Architecture and ownership

The intended production path is **OBS specific-window Game Capture → one H264
encode → private MediaMTX relay → browser WebRTC**. The authenticated controller
proxies only WHEP signaling. It does not proxy, decode, transcode or composite
video. A later destination is a separate optional relay output; it is disabled
and unnecessary for local viewing. Avoid a second compositor/encoder service.

OBS owns an interactive-desktop session independent of Tas and the controller.
Its output stays connected while it switches between the original Renki slate
and a verified game scene. Game logout, game restart and controller restart must
not stop OBS or a destination session. No new lifecycle hooks were added to
bot/game start, stop, trial cleanup or emergency body stop.

The relay can also loop a prerecorded copy of the same slate when its publisher
disappears, using v1.21.3 `alwaysAvailable` / `alwaysAvailableFile`. This protects
local readers through **publisher loss**, whereas an OBS scene switch protects
both its continuing local output and a future destination from **game loss**.
Relay availability is not proof of game capture. A publisher that is alive but
frozen/black must be switched to slate by the capture adapter; the relay cannot
infer that from packets. Destroying the relay itself requires browser reconnect.

The original code-native card uses a geometric Renki mark, calibration rings,
muted green and cream, subtle scan lines and a slow indicator with reduced-motion
support. It uses no Fallout assets, characters, logos, remote fonts or analytics.
`panel/src/standby.jsx` and `standby.css` are shared by the panel fallback and the
standalone OBS page. The encoded offline segment is rendered from that exact
standalone page, rather than a separately maintained picture.

## State, trust and recovery

Transport stays alive even while content is covered. The panel distinguishes:

- **standby**: Tas is resting, capture is unverified, health expired, or video is
  unconfigured. Do not show a cached game picture beneath an uncertain scope.
- **connecting**: an enabled viewer is negotiating its initial session.
- **gameplay**: current server-side capture assertion matches the current
  connection/world and at least two later decoded frames have arrived.
- **reconnecting**: frames stopped or transport failed; cover the video first.
- **unavailable**: five unsuccessful attempts exhausted the recovery budget.
  The Retry video button resets it; viewing needs no additional approval.

Status is polled every second with a two-second request timeout. A server-side
capture assertion lasts at most three seconds and the browser trusts a received
status for at most 2.5 seconds. An adapter silently dying is covered after those expiry windows; include up to
two seconds of signaling delay and one polling tick (at most roughly 8.5 seconds
with timely browser scheduling). A received logout/scope-change event covers
immediately. The next one-second check after a three-second decoded-frame gap
covers the picture even if ICE is connected. After eight seconds from negotiation, a failed connection or five
seconds without decoded frames starts recovery. Negotiation is bounded to twelve
seconds, including host ICE gathering; signaling itself is bounded to eight.
Retries back off 1/2/4/8 seconds and stop after five consecutive attempts. A
working decoded stream lasting over fifteen seconds resets that failure budget.
Unmount closes the peer and sends a bounded session DELETE. ICE teardown also
releases readers if HTTP cleanup fails.

Neither `RTCPeerConnection.connectionState`, decoded video, OBS `GetSourceActive`
nor OBS renderer FPS proves fresh **game** frames. Standby frames also decode.
The controller therefore defaults to covered content until an independent local
capture adapter provides a fresh assertion. A real Windows adapter is **not
implemented or validated** in this slice. Optional configured playback without
such an adapter remains safely on the slate. The synthetic proof supplies a
clearly labelled test assertion; never use that writer with real game capture.

## Configuration and adapter boundary

Add a startup-only block under `panel` in the controller's private config:

```json
{"video":{"port":18889,"password":"PRIVATE_RELAY_READ_PASSWORD","healthFile":"ABSOLUTE_PRIVATE_PATH/health.local.json"}}
```

Omit it or use port zero to disable video. The upstream is always
`http://127.0.0.1:<port>/renki/whep`; browsers cannot select another URL or path.
The relay username is fixed to `panel`. The password stays in server config,
is redacted from controller startup logging, and is never returned by the video
status endpoint or embedded in assets. The panel's existing login cookie/bearer
auth covers all signaling. Only POST SDP and DELETE fixed UUID sessions are
accepted, with the panel action header; no general proxy, redirects, arbitrary
Location, OPTIONS, PATCH, WHIP publishing, raw relay errors or upstream cookies.
SDP bodies/answers are capped at 64 KiB, request timeouts at eight seconds, and
concurrent signaling operations at four. GET status bypasses those slots. Native
non-trickle host ICE avoids a dependency on a browser player SDK. Ephemeral ICE
credentials necessarily travel in the authenticated SDP; long-lived relay read
credentials do not. Upstream proxy environment variables are ignored.

The adapter atomically replaces the configured local health file, at most 4 KiB:

```json
{"gameplay":true,"connection":7,"worldUid":"CURRENT_UID","observedAt":"2026-10-30T22:00:00Z"}
```

It must set `gameplay:false` on uncertainty, scene/slate selection, absent
capture, wrong window, stale observations or scope change. A true assertion must
refer to the **actual current capture**, independently prove fresh game frames,
and carry current controller connection/world identity; copied controller
online status is insufficient. UTC timestamps in the future, older than three
seconds, malformed/oversize/missing files and mismatched scope fail closed.
Keep the path on a local regular filesystem, writable only by that adapter and
readable by the controller; do not use a pipe, device or network mount.

Windows follow-up must implement the authenticated obs-websocket v5 connection,
verified specific-window source mapping, reliable fresh-capture evidence and
scene reconciliation. Start on `Renki Offline`; return to `Renki Gameplay` only
after several verified checks in the new scope. On uncertainty select the slate,
retain OBS output, and report a fixed safe reason. Never broaden to desktop or
any foreground window, restart the game, or stop output just because Tas logs
out. Validate scene changes, minimized/window recreation, RDP/session changes,
OBS reconnect, source black/freeze and new-world buffering on the actual VM.
`GetSourceScreenshot` alone cannot distinguish a motionless game from a freeze.
The offline tools from PR #112 remain offline; they do not perform these writes.

## Loopback recipes

Prerequisites: Linux amd64 for the pinned binary helper, FFmpeg with libx264,
Node dependencies and the locked Chromium (`just panel ui-install`). No Docker,
external accounts, API tokens or stream keys are required. Installation downloads
the public MediaMTX v1.21.3 release and checks its pinned SHA-256. Everything else
runs locally. Run the following in separate terminals where indicated:

1. `just streaming install` puts the verified binary under `.checks/streaming/`.
2. `just streaming prepare` builds the panel/shared slate, renders a 1280×720 PNG,
   encodes a silent 30 fps baseline H264 MP4, and writes private mode-0600 configs.
3. `just streaming relay` runs MediaMTX in the foreground.
4. `just streaming source` publishes moving FFmpeg test bars over local RTSP.
5. `just streaming proof-host` serves an authenticated **synthetic-only** panel
   on 127.0.0.1:18793 with token `local-synthetic-proof-token`. It is a test host,
   has no game bridge, and cannot execute body commands. Only the proof runner
   toggles its simulated capture assertion; the host alone stays on standby.
6. `just streaming slate` serves the standalone original page at
   `http://127.0.0.1:18794/`. Use this local URL as an OBS Browser Source, at
   1280×720, with page permissions set to no access. It serves no command APIs,
   panel data or credentials. Run it independently of controller/game lifecycle.

`just check streaming proof` owns and cleans up its own relay, publisher,
synthetic host and Chromium. It refuses occupied test HTTP ports. Do not run it
alongside the manual instances above. Detailed logs, screenshots, decoded counts
and WHEP-session evidence are retained in `.checks/streaming/proof/` (H264) or `.checks/streaming/proof-vp9/` (VP9). This is an
integration check requiring the downloaded relay; the portable CI gate does not
download or require a media server. CI runs server/client lifecycle tests,
loopback configuration checks and UI fixture scenarios instead. Do not run
multiple panel recipes that reinstall dependencies concurrently.

Generated relay config permits anonymous **publish only from loopback**, with
read access limited to the server-side panel credential. RTSP TCP binds
127.0.0.1:18554, optional OBS RTMP ingress 127.0.0.1:11935, WHEP HTTP
127.0.0.1:18889 and ICE UDP/TCP 127.0.0.1:18189. Only path `renki` exists, with
four readers, no publisher takeover, forwarding, recording, HLS, SRT, MoQ, public
API or telemetry listeners. Static slate and proof host also bind loopback.
Any local process can publish; this assumes a trusted local user environment.

## Private LAN/Tailscale and later outputs

This checked-in configuration is intentionally loopback-only. It does **not**
make remote LAN/Tailscale playback work just because panel HTTP is reachable.
For that later deployment, keep WHEP HTTP loopback behind panel auth, bind ICE
listeners specifically to the host's approved LAN/Tailscale address and advertise
that reachable address in `webrtcAdditionalHosts`. Allow only intended viewers
through the host/tailnet firewall to the chosen ICE UDP/TCP port. Do not publish
relay signaling/ingress/API or OBS websocket to the internet. If OBS and relay
are on different hosts, separately design authenticated private ingest; do not
copy the anonymous loopback publisher permission onto a LAN listener.

Use HTTPS for the panel on real LAN/tailnet hostnames, including a valid browser
certificate. Loopback HTTP is a browser secure-context exception; plain remote
HTTP is not a portable WebRTC deployment. TLS at the panel does not tunnel ICE:
media still needs its own reachable private candidate. There is no public STUN
or TURN dependency in this proof. Restrictive networks may need private TURN
with short-lived scoped credentials and a reviewed deployment configuration;
this player intentionally does not forward arbitrary upstream ICE-server Link
headers or promise general NAT traversal.

Start with OBS H264 baseline/constrained baseline, yuv420p, no B frames, 720p30,
3 Mb/s and two-second keyframes; validate actual browser/hardware codec support
before choosing another profile. The relay remuxes/packetizes once-encoded media.
Each browser adds a distinct WebRTC reader: four viewers at 3 Mb/s require roughly
12 Mb/s outbound plus overhead. Measure actual encoder/GPU/frame costs on Windows;
this Linux synthetic run is not that benchmark. Hidden tabs can throttle decode;
recovery must still cover stale media when a tab returns. Other browser engines
and native mobile remain unverified. Future Twitch output is separately configured
and disabled here, and must preserve the slate during game downtime rather than
owning Tas lifecycle. No Twitch delivery, live OBS capture or Windows claims are
made by this proof.

## Versioned primary references

Verified against MediaMTX **v1.21.3**, rather than unversioned config examples:

- [Complete versioned configuration](https://github.com/bluenviron/mediamtx/blob/v1.21.3/mediamtx.yml):
  `authInternalUsers`, loopback listener/ICE options, `maxReaders`,
  `alwaysAvailableFile`, `record` and `forward`.
- [Versioned WHEP implementation](https://github.com/bluenviron/mediamtx/blob/v1.21.3/internal/servers/webrtc/http_server.go):
  POST `/<path>/whep`, 201 SDP answer, relative UUID Location, DELETE session.
- [Versioned native reader](https://github.com/bluenviron/mediamtx/blob/v1.21.3/internal/servers/webrtc/reader.js)
  and [v1.21.3 release artifacts](https://github.com/bluenviron/mediamtx/releases/tag/v1.21.3).
- [OBS websocket v5 protocol](https://github.com/obsproject/obs-websocket/blob/5.6.3/docs/generated/protocol.md)
  for the future capture/scene adapter; see PR #112 research in [STREAMING.md](STREAMING.md).

### Browser codec limitation in this development environment

The locked Playwright Chromium build on this host advertises VP8/VP9/AV1 but no
H264, including its full Chromium channel. Accordingly `just check streaming
proof` fails explicitly on the missing H264 capability; it must not silently
switch codecs or claim H264 acceptance. `just check streaming proof-vp9` is the
explicit VP9 synthetic variant, with `prepare-vp9` / `source-vp9` recipes; it
encodes the slate and test source as VP9 once and the relay still never transcodes.
It uses the same ports/config and must not run concurrently with the H264 setup.
T3's collaborative browser on this host advertises H264. After rerunning
`just streaming prepare`, use the manual H264 relay/source/proof-host recipes and
that browser to check actual H264 decode and lifecycle. Keep evidence for the two
codecs separate. Real OBS/Windows acceptance still requires the actual encoder,
browser and capture-freshness integration; a successful VP9 test is not H264 proof.

## Retained implementation evidence (2026-10-30)

The strengthened `just check streaming proof-vp9` run passed all five integration
checks, including moving source pixels distinct from the slate, a matching
encoded slate after publisher loss, one WHEP POST through source restart, and a
new decoding session after a separate relay outage. Its artifact is
`.checks/streaming/proof-vp9/result.json`; full run log is
`.checks/20261030T220241Z-bh8Jzk.log`.

The **H264** run used T3's collaborative browser against the actual authenticated
Go handler, same private relay and silent 1280×720 FFmpeg H264 publisher. At no
source, 132 slate frames decoded. Source playback reached 462 frames; Tas offline
covered the picture at 910 while preserving the same MediaStream and one WHEP
POST. After killing the publisher, the encoded slate advanced from 1075 to 1165
frames over three seconds and matched the initial slate (mean 8-bit channel
difference 0.012). Restarted H264 source reached 1511 frames, changed the picture
back to test bars, and retained that same stream and one WHEP POST. This is actual
local H264 media continuity, **not OBS/Windows game capture or Twitch proof**.
Metrics/capabilities and screenshots are retained under
`.checks/streaming/proof-h264/`, with `result.json`, `connected.png`,
`publisher-down.png`, `publisher-restored.png`, `no-source.png` and
`tas-resting.png`. These are private local artifacts, not shipped product assets.

For repeating the manual synthetic checks, `just streaming proof-state` accepts
`source-on`, `source-off`, `offline` and `online` for the synthetic-only host.
Start a source before `source-on`, assert it is actually moving, and set
`source-off` when stopping it. This simulated health assertion is for the test
host only; production must never infer capture freshness from these toggles.
