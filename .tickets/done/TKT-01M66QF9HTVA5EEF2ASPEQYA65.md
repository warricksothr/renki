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
updated_at: 2026-10-30T16:29:09Z
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

**agent:codex/renki-atlas** at 2026-10-30T16:22:33Z

Deployed 5f94daf after PR #79 (persistence), #80/#81 (installer lock/error handling). Full package checksum verification passed. Live DizzysHellheim world UID 827096491, namespace local/epoch 1: captured 841 canonical samples across 81 derived viewport tiles. All 841 retained with unchanged height/biome values and no older measurement times after controller reconnect, game shutdown plus offline panel restart, and a new game process rejoin. Persistent observer source stayed stable; the producer stream stayed the same across controller restart and changed across game restart as intended. Fresh unchanged confirmations advanced all 841 measurement times. No nearby actors were reported at initial capture; Tas rejoined at full current health 59/59. No terrain edits or harvesting were performed; localized changed-value/border/expiry behavior is proven by replay/storage checks, not a live landscaping edit.

Added read-only just vm atlas-check, which captures full evidence under .checks, prints bounded counts/version/health and compares canonical samples, ages and border consistency against a baseline. The helper checks live identity when in-world, supports saved/offline reads and never sends tasks or changes settings. Added regression checks for missing samples, old times and conflicting border evidence; all are in just check ci.

Evidence: .checks/atlas-live-5eh3p64k.json (initial), .checks/atlas-live-y811eqrm.json (controller reconnect), .checks/atlas-live-nz2zjdd8.json (offline panel), .checks/atlas-live-c0qhp5x5.json (new game process). CI: .checks/20261030T161948Z-JuSjpP.log. Final shutdown/offline confirmation follows. Portable export/import remains the fourth draft child; no map publication is enabled.

**agent:codex/renki-atlas** at 2026-10-30T16:29:09Z

Final native shutdown/offline proof retained all 841 samples: .checks/atlas-live-yhj_ei_8.json. Tas is logged out, Valheim stopped, and the controller/panel is online. The user additionally requested automatic offline terrain/fog, so the page now defaults to Follow Tas: it preserves the last live atlas when he leaves, loads the newest saved atlas on a fresh offline page, dims known terrain and keeps unknown cells dark. Rejoining automatically restores live mode unless the user pinned a saved world. Live actors/objects remain cleared for saved views; request generations reject delayed live responses.

Browser checks using the recorded native offline atlas passed initial offline fallback (81 derived tiles), live return, explicit saved-world pinning and a delayed live response across logout, with zero browser errors. Added a rendering regression proving fog dims known cells and never fills unknown cells. just check ci passed with the new read-only VM evidence helper/oracle checks (log .checks/20261030T162752Z-ID5jSz.log). The final controller-only UI deployment follows this commit; portable sharing remains a separate draft child.

## Summary

Implemented authenticated /api/atlas/worlds and bounded /api/atlas/map, stable keys on live snapshots and saved-world selection. Terrain hover includes oldest contributing measurement time, source and spacing; stale cells fade. Saved selection clears live actors/objects and has no waypoint task scope. Browser fixture passed offline viewing and a delayed live response across saved selection, with zero browser errors. Viewports are capped at 512 m per side; coarse overview rendering remains deferred. Auth/viewport/offline API checks and just check ci passed.
