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
updated_at: 2026-10-30T21:47:23Z
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

- [ ] Optional authenticated video uses bounded private signaling; JPEG diagnostics remain independent
- [ ] Shared original standby screen and player recovery hide untrusted or stalled game frames
- [ ] Loopback synthetic end-to-end evidence and fixture screenshots distinguish implemented behavior from Windows OBS follow-up
- [ ] Full CI and committed review pass; stacked PR documents architecture and remaining checks

## Implementation plan

Use native recvonly WebRTC with non-trickle ICE and same-origin authenticated WHEP POST/DELETE. Server fixes loopback upstream/path, refuses redirects/arbitrary locations, bounds SDP/timeouts/viewers and never forwards credentials or cookies. Configuration is startup-only and optional. Keep media transport alive through logout/world change; immediately cover unsafe content, clear freshness on scope change and require later decoded frames. Detect stalls by decoded frame counters and bound reconnect/backoff; do not equate frame arrival with game-capture truth.

Retain the Camera export and move its existing JPEG UI into collapsed diagnostic details, mounted only when requested. Share an original geometric calibration-card HTML/CSS slate between the panel and a loopback-only OBS-compatible static page. Provide pinned MediaMTX install/config/run and synthetic H264 publisher recipes without external outputs. OBS game health adapter remains unimplemented pending actual Windows freshness evidence; document that explicit boundary and required lifecycle tests.

Reuse the repository Playwright fixture suite, add lifecycle tests and authenticated server integration tests, and run actual loopback FFmpeg -> MediaMTX -> authenticated panel proxy -> Chromium proof. Retain artifacts and screenshots. Run just check ci then commit, just check review, address findings and push a PR stacked against t3/research-twitch-streaming. No subagents or VM access.
