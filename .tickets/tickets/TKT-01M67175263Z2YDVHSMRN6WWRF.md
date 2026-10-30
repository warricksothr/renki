---
schema: 4
id: TKT-01M67175263Z2YDVHSMRN6WWRF
title: Capture bounded game-camera JPEGs for visual debugging
type: task
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - controller
  - protocol
  - tooling
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-milestones
  branch: t3/game-camera
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 89b0d3bc93b65813b7c1fd720cddd334e111c169
  session: null
  claimed_at: 2026-10-30T18:30:37Z
  expires_at: null
archive: null
created_at: 2026-10-30T18:30:10Z
updated_at: 2026-10-30T18:44:50Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

The owner requests an updated JPEG of the bot's rendered game view every five seconds, configurable with a ceiling of one capture per second, for the panel and automated visual debugging. Implement native Unity capture at a bounded resolution/quality and one capture in flight. Use a latest-only media slot, authenticated retrieval with scoped metadata, runtime controls and no embedded image blobs in ordinary audit/model context. Clear images on session/world changes and report unavailable/offline state honestly. Provide a repeatable agent-friendly capture recipe and live verify on the VM without owner assistance. Keep procedure design PR #92 unmerged and continue M2/M3 after this slice.

## Acceptance criteria

- [ ] Rendered camera JPEG capture defaults to five seconds and never starts more than once per second, with bounded dimensions/bytes and no accumulating queue
- [ ] Authenticated latest-image API and panel show capture time, scope and task; stale session images do not appear current
- [ ] Runtime enable/interval/quality/size controls validate bounds; image blobs stay outside routine audit and model context
- [ ] Core, protocol, session, panel and live VM checks verify capture and debugging retrieval with no owner setup
- [ ] Users and agent tools export the actual panel map renderer with matching scoped data; read-only standalone rendering works without an open panel

## Implementation plan

Add a bounded screenshot message and shared capture config; Unity end-of-frame GPU downscale/readback/JPEG with a hard capture scheduler; latest-only low-priority bridge slot; strip image bytes from audit; validate and scope latest media in session; authenticated JPEG/metadata endpoint and panel view; runtime controls and agent-friendly capture recipe. Verify scheduler/transport/validation/scope/auth plus live image on the VM, then land and continue M2/M3 route and gathering fixes.

## Notes

**agent:codex/renki-milestones** at 2026-10-30T18:44:50Z

Implementation and fixture checks pass: just check ci (.checks/20261030T184419Z-Hlr3s2.log). Actual panel-renderer fixture rendered through native HTML preview without console errors. Live VM verification remains open; no completion claimed before deployment.
