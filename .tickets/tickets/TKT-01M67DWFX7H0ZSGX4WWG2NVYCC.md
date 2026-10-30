---
schema: 4
id: TKT-01M67DWFX7H0ZSGX4WWG2NVYCC
title: Validate Windows OBS freshness and persistent standby scenes
type: task
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - vm
  - controller
  - tooling
assignees: []
milestone: null
parent: null
origin: null
dependencies:
  - TKT-01M67CF3AMXYJJB2PYR790ZDH5
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-milestones
  branch: t3/windows-obs-capture
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 2c18cf13ec31b7dd9e13c85ce960ade7f897e0ec
  session: null
  claimed_at: 2026-10-30T22:42:29Z
  expires_at: null
archive: null
created_at: 2026-10-30T22:11:32Z
updated_at: 2026-10-30T23:07:24Z
created_by:
  id: agent:codex/local-streaming
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

Follow the local-only proof from PR113 (Add local panel video and continuous standby slate), stacked above PR112 (Research safe OBS broadcasting and offline profile tooling). This is future Windows work, not VM access or deployment authorized by the local proof task. See docs/LOCAL-VIDEO.md and docs/STREAMING.md.

Implement a bounded authenticated obs-websocket v5 adapter in the VM's approved interactive desktop. Verify specific-window game capture settings against the installed OBS build; never broaden to desktop or arbitrary foreground capture. Establish fresh game-frame evidence that cannot be satisfied by OBS source activity, render FPS, packet arrival, a motionless screenshot or the standby slate. Write the scoped local capture assertion atomically and fail closed on controller/scope/capture uncertainty.

Start on the shared original Renki standby scene. Switch to gameplay only after several fresh verified checks; on game logout, source loss/freeze/black frames, world change or adapter failure switch to standby while retaining the video/output session. Keep OBS/relay lifecycle independent of bot/game cleanup and emergency body stop. Validate actual Windows codec/encoder/GPU costs, minimized/window recreation, RDP/console transitions, capture re-acquisition and scene reconciliation. Keep audio off initially. No Twitch or other external destination is required to close this slice.

## Acceptance criteria

- [x] Actual fresh game capture is independently verified and assertions match current connection/world
- [x] OBS switches safely to shared standby through game/scope/capture failures without stopping media/output
- [ ] Windows capture, encoder costs, desktop transitions and recovery evidence are retained; body stop remains independent

## Implementation plan

User explicitly authorized BLIGHT deployment and local OBS setup and will leave its console available. Install the tested merged controller/plugin while Tas stays offline, restart only the panel, and retain checksums/status evidence. Provision official pinned OBS and MediaMTX Windows releases through idempotent just recipes, with local credentials and independent interactive scheduled tasks.

Implement authenticated loopback-only obs-websocket v5 adapter with fixed operations, bounded payloads/timeouts, redacted diagnostics and private local config. Keep audio/external destinations off. Configure dedicated exact-Valheim game capture and shared standby scenes. Verify actual capture freshness using an opt-in game-rendered changing nonce: only match recently reported scoped nonces in screenshots of the exact OBS capture source, require progress over distinct nonces, reject stale/mismatched scope and switch to standby on uncertainty. OBS activity/FPS alone cannot grant gameplay. Exercise game offline/online, lost capture and adapter restart without stopping encoded output; preserve body-stop independence.

Validate focused protocol/freshness policy tests and Windows local capture/encoder evidence before enabling gameplay. Configure private reachable ICE and HTTPS only with measured remote playback. Document unexercised desktop transitions honestly. Run just check ci, commit and committed just check review; open and land a concrete deployment PR with evidence.

## Notes

**agent:codex/renki-milestones** at 2026-10-30T23:07:24Z

Owner explicitly authorized deployment/OBS preparation and left BLIGHT console available. Deployed main2c18cf1 panel first, then candidate plugin with opt-in changing game-frame beacon. Installed checksum-pinned OBS33.0.1 portable and MediaMTX1.21.3; websocket auth/private ACLs stay on VM; ICE18189 is tailnet-only and control/ingress loopback. Private Tailscale Serve HTTPS works with a loopback18780 port proxy. No external destination/account configured.

Actual Windows NVENC output verified baseline H264/720p30/3Mb/s/keyint60/no B frames. Found and corrected silent-AAC vs H264-only fallback track mismatch, advanced profile falling back to x264, and Studio preview being reset to standby. Actual captured game marker/nonces match current world/connection and grant playback only after progress. Successful sample: OBS~1.5% CPU,0.58ms render time,0 encoder-skipped frames; not a stress benchmark. Actual game-source PNG .checks/obs-windows/game-snapshot.png.

Live source-loss trial passed (.checks/20261030T230200Z-ZDWJ2O.log); adapter-kill watchdog trial passed (.checks/20261030T230206Z-f9uAVA.log); body stop returned200 while video continued; logout passed (.checks/20261030T230258Z-UlrPU4.log). Same browser MediaStream and unchanged WHEP request count2 through these checks (earlier relay provisioning created the second session). Logout evidence: menu/standby,7780 decoded frames. OBS output duration increased throughout. Tas stayed healthy; only natural food maximum decline, then logged out and game closed (.checks/20261030T230436Z-Xw5lal.log).

Deployment exposed EventSource CLOSED after proxy502 during controller restart: native EventSource does not always retry. Added bounded explicit CLOSED-source recovery while preserving native transient retries; tests cover duplicate/obsolete events and shutdown. Final combined CI/review/deployment follows. AC3 deliberately remains open: RDP/lock/unlock/minimize/GPU-loss and actual forced render-freeze transitions have not been live-validated; frozen-image policy is covered by decoded-image/scoped-telemetry tests. No manual desktop interruption or user credentials requested.
