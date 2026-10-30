---
schema: 4
id: TKT-01M67DWFX7H0ZSGX4WWG2NVYCC
title: Validate Windows OBS freshness and persistent standby scenes
type: task
status: draft
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
claim: null
archive: null
created_at: 2026-10-30T22:11:32Z
updated_at: 2026-10-30T22:11:32Z
created_by:
  id: agent:codex/local-streaming
  name: ""
updated_by:
  id: agent:codex/local-streaming
  name: ""
extensions: {}
---

## Description

Follow the local-only proof from PR113 (Add local panel video and continuous standby slate), stacked above PR112 (Research safe OBS broadcasting and offline profile tooling). This is future Windows work, not VM access or deployment authorized by the local proof task. See docs/LOCAL-VIDEO.md and docs/STREAMING.md.

Implement a bounded authenticated obs-websocket v5 adapter in the VM's approved interactive desktop. Verify specific-window game capture settings against the installed OBS build; never broaden to desktop or arbitrary foreground capture. Establish fresh game-frame evidence that cannot be satisfied by OBS source activity, render FPS, packet arrival, a motionless screenshot or the standby slate. Write the scoped local capture assertion atomically and fail closed on controller/scope/capture uncertainty.

Start on the shared original Renki standby scene. Switch to gameplay only after several fresh verified checks; on game logout, source loss/freeze/black frames, world change or adapter failure switch to standby while retaining the video/output session. Keep OBS/relay lifecycle independent of bot/game cleanup and emergency body stop. Validate actual Windows codec/encoder/GPU costs, minimized/window recreation, RDP/console transitions, capture re-acquisition and scene reconciliation. Keep audio off initially. No Twitch or other external destination is required to close this slice.

## Acceptance criteria

- [ ] Actual fresh game capture is independently verified and assertions match current connection/world
- [ ] OBS switches safely to shared standby through game/scope/capture failures without stopping media/output
- [ ] Windows capture, encoder costs, desktop transitions and recovery evidence are retained; body stop remains independent
