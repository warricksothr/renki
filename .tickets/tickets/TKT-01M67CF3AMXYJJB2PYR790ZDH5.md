---
schema: 4
id: TKT-01M67CF3AMXYJJB2PYR790ZDH5
title: Add local panel video and persistent standby scene
type: task
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - tooling
  - docs
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/local-streaming
  branch: t3/local-streaming-standby
  worktree: /home/sothr/.t3/worktrees/renki/t3-52079701
  commit: d7ee89f04e17a7d505173a5649b78e5fe1a39deb
  session: null
  claimed_at: 2026-10-30T21:47:23Z
  expires_at: null
archive: null
created_at: 2026-10-30T21:46:44Z
updated_at: 2026-10-30T22:10:49Z
created_by:
  id: agent:codex/local-streaming
  name: ""
updated_by:
  id: agent:codex/local-streaming
  name: ""
extensions: {}
---

## Description

Implement the explicitly authorized local-only streaming proof above PR112, preserving the parent 06d915b evidence. Add authenticated bounded WHEP signaling, an optional native browser player independent of diagnostic JPEG controls, shared original standby artwork and a standalone OBS slate, loopback MediaMTX tooling, meaningful tests and synthetic browser evidence. No VM, external destinations, deployment or merge. OBS capture freshness/scene switching must be scoped honestly if Windows validation is unavailable.

## Acceptance criteria

- [x] Optional authenticated video uses bounded private signaling; JPEG diagnostics remain independent
- [x] Shared original standby screen and player recovery hide untrusted or stalled game frames
- [x] Loopback synthetic end-to-end evidence and fixture screenshots distinguish implemented behavior from Windows OBS follow-up
- [ ] Full CI and committed review pass; stacked PR documents architecture and remaining checks

## Implementation plan

Use native recvonly WebRTC with non-trickle ICE and same-origin authenticated WHEP POST/DELETE. Server fixes loopback upstream/path, refuses redirects/arbitrary locations, bounds SDP/timeouts/viewers and never forwards credentials or cookies. Configuration is startup-only and optional. Keep media transport alive through logout/world change; immediately cover unsafe content, clear freshness on scope change and require later decoded frames. Detect stalls by decoded frame counters and bound reconnect/backoff; do not equate frame arrival with game-capture truth.

Retain the Camera export and move its existing JPEG UI into collapsed diagnostic details, mounted only when requested. Share an original geometric calibration-card HTML/CSS slate between the panel and a loopback-only OBS-compatible static page. Provide pinned MediaMTX install/config/run and synthetic H264 publisher recipes without external outputs. OBS game health adapter remains unimplemented pending actual Windows freshness evidence; document that explicit boundary and required lifecycle tests.

Reuse the repository Playwright fixture suite, add lifecycle tests and authenticated server integration tests, and run actual loopback FFmpeg -> MediaMTX -> authenticated panel proxy -> Chromium proof. Retain artifacts and screenshots. Run just check ci then commit, just check review, address findings and push a PR stacked against t3/research-twitch-streaming. No subagents or VM access.

## Notes

**agent:codex/local-streaming** at 2026-10-30T22:02:03Z

PR113 (Add local panel video and continuous standby slate) is open as a draft above PR112, preserving parent 06d915b. Implemented optional native video in the existing Camera export, lazy diagnostic JPEG UI, bounded authenticated loopback WHEP POST/DELETE, and a capture-health assertion contract separate from media transport. No game/bot/VM control lifecycle was modified. Shared original standby page is available both in-panel and on an independent static loopback slate server; MediaMTX v1.21.3 supports alwaysAvailableFile for encoded fallback.

Focused server and client checks passed, as did all ten new video fixture/viewport combinations. Visual inspection found mobile footer clipping, corrected before final verification. Bundled Playwright Chromium lacks H264; T3 preview advertises it. Keep the explicit VP9 synthetic automated run and actual T3 H264 evidence separate. The integration assertion was strengthened after an early run could count encoded slate frames as source motion: require actual picture differences, moving source pixels and a live publisher process, plus pixel similarity on return to slate. Earlier exploratory runs are not final acceptance.

Actual Windows OBS capture freshness and automatic scene switching remain unimplemented; production stays covered without a fresh matching health assertion. No Twitch/external destination, credentials/account, VM deployment or game access has occurred. See docs/LOCAL-VIDEO.md for security/ICE/recovery details and exact remaining Windows checks.

**agent:codex/local-streaming** at 2026-10-30T22:10:49Z

Actual H264 local proof passed through T3 preview: 1280x720 decode, encoded slate before source, moving synthetic FFmpeg source, bot offline covering with identical MediaStream, source killed -> slate advancing 1075 to1165 frames in3s, restored source at1511 frames with exactly one WHEP POST. Evidence .checks/streaming/proof-h264/result.json and matching PNGs. Stronger automated VP9 proof passed five checks including distinct moving source pixels, matching encoded fallback and independent relay-outage recovery (.checks/streaming/proof-vp9/result.json; log .checks/20261030T220241Z-bh8Jzk.log). Bundled Chromium lacks H264; explicit H264 proof recipe correctly reports that limitation. No VM, OBS, actual game, Twitch or external destination was used.

Full just check ci passed (.checks/20261030T220538Z-zyE8PK.log). Subsequent mobile visual review identified aspect-ratio/min-height width expansion; explicit width and a fitting assertion corrected it. All ten focused video cases passed again in .checks/video-ui-final, with reviewed screenshot copy visual-review-reviewed.json. Final complete gate is running before implementation commit. Keep AC4 pending until committed review and PR handoff.
