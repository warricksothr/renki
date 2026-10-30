---
schema: 4
id: TKT-01M66QF9HTVA5EEF2ASPEQYA65
title: Show saved and stale atlas layers in the map
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - controller
assignees: []
milestone: null
parent: TKT-01M66Q5SFWV99JCGJXVARVRJH7
origin: null
dependencies:
  - TKT-01M66QF9G576JFCQ0JBP3RWR2E
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-30T15:39:51Z
updated_at: 2026-10-30T16:04:17Z
created_by:
  id: agent:codex/renki-atlas
  name: ""
updated_by:
  id: agent:codex/renki-atlas
  name: ""
extensions: {}
---

## Description

Expose bounded saved-atlas viewport/overview queries and panel rendering as proposed in docs/ATLAS.md. Accumulate terrain beyond the hot cache, allow explicit saved-world selection while offline and show observation age, source and resolution. Distinguish live connection, remembered/stale, imported and unknown evidence. Keep current canvas initially; persisted backdrop does not establish observed coverage, walkability or route safety.

## Acceptance criteria

- [x] Panel restores accumulated terrain across reconnect/restart and can select a saved world while offline
- [x] Hover/styling communicates age, provenance, resolution and stale/unknown status without presenting saved data as live
- [x] Viewport/result budgets and world/connection changes during requests are checked; live task scope remains separate from saved-world selection

## Implementation plan

Keep /api/map and live task scope intact. Add authenticated read-only saved-world and bounded terrain viewport endpoints, stable world keys in live snapshots, explicit saved-atlas selection, stale shading and oldest-corner/source/resolution hover details. The live view merges atlas terrain with session-only coverage holes. Saved views clear live actors/objects and block waypoint actions; generation checks reject in-flight stale responses. Exercise offline reads, viewport budgets, auth and browser selection/panning.

## Notes

**agent:codex/renki-atlas** at 2026-10-30T16:04:16Z

Implemented authenticated /api/atlas/worlds and bounded /api/atlas/map, stable keys on live snapshots and saved-world selection. Terrain hover includes oldest contributing measurement time, source and spacing; stale cells fade. Saved selection clears live actors/objects and has no waypoint task scope. Browser fixture passed offline viewing and a delayed live response across saved selection, with zero browser errors. Viewports are capped at 512 m per side; coarse overview rendering remains deferred. Auth/viewport/offline API checks and just check ci passed. Live deployment evidence follows after merge.

## Summary

Implemented authenticated /api/atlas/worlds and bounded /api/atlas/map, stable keys on live snapshots and saved-world selection. Terrain hover includes oldest contributing measurement time, source and spacing; stale cells fade. Saved selection clears live actors/objects and has no waypoint task scope. Browser fixture passed offline viewing and a delayed live response across saved selection, with zero browser errors. Viewports are capped at 512 m per side; coarse overview rendering remains deferred. Auth/viewport/offline API checks and just check ci passed.
