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
  branch: t3/escort-nimah
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 2220901608b9f4e9e503a1d56296477e35cbdd0e
  session: null
  claimed_at: 2026-10-31T16:00:05Z
  expires_at: null
archive: null
created_at: 2026-10-30T22:11:32Z
updated_at: 2026-10-31T16:17:03Z
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

User reported persistent standby while Tas was in-world. Read-only BLIGHT diagnostics show OBS and relay still running, RenkiCapture Ready with LastResult 1, watchdog Running, and a stale successful verification timestamp. The verifier loop's runtime exit-1 path is failure to atomically publish capture health. Make publication failure revoke Program to standby/reset freshness and retry on subsequent loop ticks instead of terminating the verifier. Add an end-to-end loop regression with a failed health write followed by recovery, and a permanently failing writer that remains contained/alive. Add bounded Task Scheduler restart settings for the capture verifier/watchdog, and expose task last-result/time in existing redacted status tooling. Keep exact-window capture, changing-frame evidence, scoped health expiry and silent local output intact. Run focused checks and just check ci, commit/review, redeploy only the adapter/tasks, and verify fresh offline status plus independent watchdog/OBS/relay continuity. Tas is safely offline at 3.2 HP following a failed unguarded recovery; no game restart for video proof until a continuous recovery guard is active. Preserve untested desktop/minimize/GPU-loss acceptance as open.

## Notes

**agent:codex/renki-milestones** at 2026-10-30T23:07:24Z

Owner explicitly authorized deployment/OBS preparation and left BLIGHT console available. Deployed main2c18cf1 panel first, then candidate plugin with opt-in changing game-frame beacon. Installed checksum-pinned OBS33.0.1 portable and MediaMTX1.21.3; websocket auth/private ACLs stay on VM; ICE18189 is tailnet-only and control/ingress loopback. Private Tailscale Serve HTTPS works with a loopback18780 port proxy. No external destination/account configured.

Actual Windows NVENC output verified baseline H264/720p30/3Mb/s/keyint60/no B frames. Found and corrected silent-AAC vs H264-only fallback track mismatch, advanced profile falling back to x264, and Studio preview being reset to standby. Actual captured game marker/nonces match current world/connection and grant playback only after progress. Successful sample: OBS~1.5% CPU,0.58ms render time,0 encoder-skipped frames; not a stress benchmark. Actual game-source PNG .checks/obs-windows/game-snapshot.png.

Live source-loss trial passed (.checks/20261030T230200Z-ZDWJ2O.log); adapter-kill watchdog trial passed (.checks/20261030T230206Z-f9uAVA.log); body stop returned200 while video continued; logout passed (.checks/20261030T230258Z-UlrPU4.log). Same browser MediaStream and unchanged WHEP request count2 through these checks (earlier relay provisioning created the second session). Logout evidence: menu/standby,7780 decoded frames. OBS output duration increased throughout. Tas stayed healthy; only natural food maximum decline, then logged out and game closed (.checks/20261030T230436Z-Xw5lal.log).

Deployment exposed EventSource CLOSED after proxy502 during controller restart: native EventSource does not always retry. Added bounded explicit CLOSED-source recovery while preserving native transient retries; tests cover duplicate/obsolete events and shutdown. Final combined CI/review/deployment follows. AC3 deliberately remains open: RDP/lock/unlock/minimize/GPU-loss and actual forced render-freeze transitions have not been live-validated; frozen-image policy is covered by decoded-image/scoped-telemetry tests. No manual desktop interruption or user credentials requested.

**agent:codex/renki-milestones** at 2026-10-31T16:09:52Z

Current read-only BLIGHT diagnostics confirm RenkiCapture Ready/LastResult1 with stale successful observedAt04:58 local, while RenkiCaptureWatchdog, OBS and relay remain Running. Program is Renki Offline and output remains active. The verifier runtime fatal path is an atomic health-publication failure; a transient Windows sharing violation is plausible but the original filesystem error was not retained, so its exact cause is unconfirmed.

Implemented contained publication failure: revoke gameplay, reset freshness, switch Program to standby and keep retrying. Added mock OBS loop regressions for transient and persistent write failures. Added bounded Task Scheduler failure restarts for verifier/watchdog and exposed last-run/result in redacted status. Exact-window/changing-frame/scope verification and independent OBS/relay lifecycle are retained. Focused Go and streaming checks passed. Full CI/review/deployment pending. Tas is safely logged out at3.2HP; no unguarded game restart for capture proof. AC3 desktop/GPU transition evidence remains open.

**agent:codex/renki-milestones** at 2026-10-31T16:17:03Z

Repair committed ate3c444d in PR130 and deployed with just check streaming windows-adapter (.checks/20261031T161240Z-hQDpkd.log). Full just check ci passed147s (.checks/20261031T160856Z-p3v3EI.log); committed just check review HEAD~2..HEAD passed with no findings (.checks/20261031T161134Z-HbjG78.log).

Actual Windows proof: RenkiCapture and RenkiCaptureWatchdog now Running with new process IDs6004/11768 and fresh status, while OBS18488 and relay16200 retain their original processes. In-world verifier observedAt2026-10-31T11:14:35.0061537-05:00 reports Verified changing game frames; exact Valheim source active, Program Renki Gameplay, silent local outputActive true. Browser API content gameplay, matching world scope1:DizzysHellheim:827096491,1280x720 readyState4,8428 decoded frames, same original MediaStream and unchanged one WHEP negotiation. Screenshot visually inspected: /home/sothr/.t3/userdata/browser-artifacts/browser-screenshot-blight-raccoon-grue-ts-net-mvwlqjz6-068edd28.png.

Tas healed under native recovery guard and was logged out/stopped safely at29.7HP. Browser returned to Tas is resting/standby while sameStream stayed true,10465 decoded frames and one WHEP negotiation (.checks/video-offline-after-recovery.json). OBS output duration increased from62070866ms offline after deployment to62137466ms inworld. No output/relay restart or broader capture source was needed. Original exit1 was confirmed; its precise filesystem cause remains unproven. AC3 desktop/minimize/GPU loss/encoder contention trials remain open. This repair does not establish those transitions.

## Summary

Core implementation delivered in PR114 (Add verified Windows OBS video and independent standby recovery) and deployed on BLIGHT at70c6e6f. Approved portable OBS33.0.1/MediaMTX1.21.3, private authenticated adapter/watchdog, changing in-game beacon, NVENC baseline720p30 local output, matched silent-AAC fallback, private ICE and Tailscale HTTPS are working. No external destination/account configured.

Full just check ci passed (.checks/20261030T230745Z-Cngr2g.log), including42 UI scenarios; committed just check review passed with no findings (.checks/20261030T231034Z-9SPtP6.log). Final installed package/checksums passed (.checks/20261030T231105Z-fiXKME.log); adapter deploy .checks/20261030T231122Z-GnqMBw.log. Core source-loss/watchdog/body-stop/logout proof is retained in the prior note and docs/WINDOWS-VIDEO.md.

The deployed panel recovered automatically from real controller downtime/proxy502 using CLOSED-source recreation: header returned from reconnecting to no plugin, then showed connecting and in-world normally on the next game launch. Actual final GUI displayed gameplay with visible video and matching current52.4/52.4 health. Live screenshot /home/sothr/.t3/userdata/browser-artifacts/browser-screenshot-blight-raccoon-grue-ts-net-mvvl92zw-d0102123.png. Final logout showed menu/standby, same MediaStream,3510 decoded frames and unchanged single WHEP request. OBS output remained active. Final game checks .checks/20261030T231216Z-W46LEn.log and .checks/20261030T231329Z-VQ9tkr.log; game then stopped and panel remains available independently.

AC1 and AC2 are satisfied. AC3 remains unchecked for actual RDP/lock/unlock/minimization/GPU-loss/forced game-render-freeze tests and sustained encoder contention. Returned this partially completed ticket to ready with AC3 open, rather than declaring those desktop transitions verified. No user action currently needed; do not lock the working console without a recovery method. Claim released at handoff; remaining tests can resume later.
