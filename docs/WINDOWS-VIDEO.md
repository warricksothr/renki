# Private Windows game video

This builds on [the loopback proof](LOCAL-VIDEO.md) with authenticated local OBS
control, verified game capture, persistent standby and private tailnet viewing.
It configures no Twitch or other external destination. An owner must authorize
installing software and changing VM configuration; BLIGHT deployment was
explicitly authorized on 2026-10-30.

## Setup and operation

Keep the bot account logged in on its console desktop with a display available.
Use existing SSH access; OBS needs no additional remote login. As in [VM.md](VM.md),
leave RDP through `C:\renki\release-rdp.ps1`. Do not lock a working console to
test recovery unless an operator can unlock it.

1. `just check streaming windows-setup` downloads checksum-pinned official
   OBS Studio 33.0.1 portable and MediaMTX 1.21.3 releases. Both ends verify the
   archives. It prepares a dedicated installation under `C:\renki\streaming`,
   an original standby page and matching encoded fallback. Existing ordinary
   OBS installations/settings are not used. Setup refuses running OBS.
2. `just check streaming windows-start` starts the independent interactive OBS
   task. `just check streaming windows-adapter` builds, verifies and installs
   the adapter and independent watchdog and creates the two scenes. Updating
   it replaces only those two processes; OBS, relay and game are independent.
3. Before the first output, `just check streaming windows-action codec` selects
   NVENC baseline H.264, 720p30, 3 Mb/s, two-second keyframes and no B frames.
   It refuses active output and changes only the dedicated portable profile.
   Verify encoder availability on the actual GPU/driver. Use
   `just check streaming windows-action local-stop` before changing the encoder.
4. `just check streaming windows-action relay` starts the private relay;
   `just check streaming windows-action local-start` selects standby and starts
   only `rtmp://127.0.0.1:11935/renki`. Microphone, desktop and game audio inputs
   are disabled or muted. OBS still emits a **silent AAC track**. The fallback
   MP4 must match its H.264/AAC tracks or MediaMTX rejects the publisher and OBS
   repeatedly reconnects. Browser negotiation requests video only.
   `just check streaming windows-slate` replaces this matching fallback and
   restarts the relay; existing readers reconnect.
5. While Valheim is stopped, `just check streaming windows-action beacon`
   enables the opt-in in-game marker in BepInEx config. Install the plugin
   containing `CaptureBeacon`, then use the usual game lifecycle recipes.
6. `just check streaming windows-action enable` adds only the private video
   block to controller config, retaining a pre-video backup. Restart separately;
   to keep Tas offline use `just check vm install` then `just check vm panel-start`
   instead of a game-starting deploy.
7. `just check streaming windows-action https` exposes the authenticated panel
   through private Tailscale Serve HTTPS. A loopback-only Windows port proxy on
   18780 forwards to the existing local tailnet address; Serve requires a
   loopback upstream. The original panel address remains usable.

`just check streaming windows-status` reports redacted state and OBS metrics.
`just check streaming windows-action snapshot` and
`just check streaming windows-action game-snapshot` retrieve allowed source
images into `.checks/obs-windows/`. JPEG/map agent APIs and live diagnostic
capture-rate controls stay independent of OBS.

Tasks `RenkiOBS`, `RenkiRelay`, `RenkiCapture` and `RenkiCaptureWatchdog` use the
existing interactive bot account. They are independently installed/startable;
this slice does not enable automatic output at Windows logon. Bot logout,
controller restarts, body stop and trial cleanup do not stop them. OBS starts
on `Renki Offline`; `local-start` also explicitly selects it.

## Capture evidence and containment

Optional `state.captureProof` / panel status contains a 64-bit random nonce and
UTC generation time, rotating every half-second in an active world session.
Unity draws a tiny black/white strip in the real game frame: a 16-bit sentinel
plus nonce at virtual 1280x720 coordinates. It is visible and is not an
authentication credential. No plugin GPU readback or JPEG encoding is added.

Once per second the adapter samples the exact OBS `Renki Game Capture` source
at 1280x720. It verifies fixed `Valheim:UnityWndClass:valheim.exe` window-only
capture and exact-title matching, and refuses extra inputs in the two encoded
scenes. Marker decoding requires strict contrast and agreement. It compares
against fresh telemetry before/after capture and requires two distinct nonces
in one stable connection/world. A held image cannot continually renew health.
Wrong window, blank/old image, malformed proof, changed scope or uncertain
controller status revokes trust. OBS source activity/FPS remain diagnostics.

Studio preview must remain on `Renki Gameplay` while Program is on standby,
otherwise OBS may stop the hook and return black source screenshots. Program
selection avoids redundant changes and restores the private preview afterward.

The adapter atomically publishes the scoped three-second health assertion. A
separate process watches expiry and selects OBS standby if the adapter dies,
retaining encoded output. The panel independently expires assertions and detects
stalled decoded frames. See LOCAL-VIDEO.md for polling/expiry windows; these are
bounded checks, not instantaneous protection during desktop/GPU failure.

A temporary failure replacing the health file revokes Program to standby and
resets freshness verification; the adapter retries rather than exiting. The
capture verifier and watchdog tasks also restart after an unexpected process
failure (up to five attempts, one minute apart). `just streaming windows-status`
includes task state, last-run time, last result and the verifier observation
time. A running OBS output with an old observation time means the verifier
needs attention; OBS frame rate alone cannot establish healthy game capture.

OBS websocket v5 is authenticated; firewall rules block network access to 4455
while loopback works. Passwords stay in ACL-restricted local files, never in
diagnostics. The adapter has no inbound listener and sends the panel credential
only to a literal IP belonging to this VM. Redirects/environment proxies are
disabled; requests, image sizes and decoded dimensions are bounded. Raw OBS
errors and settings are not logged.

Relay control/RTSP/RTMP bind loopback. Only ICE UDP/TCP 18189 binds the local
tailnet IP, with firewall access limited to the tailnet range. Read credentials
stay server-side. HTTPS does not tunnel ICE: clients also need private access
to that media address. Nothing enables public STUN/TURN, relay control, Funnel
or an external broadcast.

## Evidence and limits

BLIGHT's RTX 3090/driver encoded baseline NVENC with no B frames. Actual 1280x720
game video reached the authenticated panel over Tailscale HTTPS. During the short
game check OBS reported about 1.5% CPU, 0.58 ms average frame render time and zero
encoder-skipped frames. This is a short sample, not a stress/headroom benchmark.
Game capture and its marker were visually inspected. Tas remained healthy and
was logged out after testing.

`just check streaming windows-action trial-source-loss` temporarily selects
only a nonexistent exact Valheim window, verifies adapter standby, then restores
the fixed game window. `just check streaming windows-action trial-watchdog`
kills only the adapter, verifies independent standby with increasing output
duration, then restarts it. Both passed on Windows; gameplay recovered. Body stop
and logout also passed while OBS output and the same browser MediaStream
continued. WHEP requests did not increase across those four checks. Earlier
relay/encoder provisioning restarts required renegotiation and are separate.

Actual RDP/lock/unlock, minimization, GPU loss and forced game-render freezes are
not yet live-validated. Frozen-image rejection is tested with decoded images and
scoped telemetry. Do not infer those unexercised transitions or sustained encoder
headroom from the console test. No Twitch/external delivery has been tested.

Deployment also exposed a recovery bug: a proxy's 502/503 could permanently close
EventSource after controller restart. The panel now recreates CLOSED sources
with bounded backoff while retaining native transient retries. Tests cover
duplicate errors, obsolete events and cancellation.
