---
schema: 4
id: TKT-01M62VXZ0J9Q2WH4XN4E6N7F27
title: Know which parts of the map the bot has explored
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - controller
  - protocol
  - live
  - movement
assignees: []
milestone: M7
parent: TKT-01M62VXYQB6ZDBQE4E6ERGN83C
origin: null
dependencies:
  - TKT-01M6EFQN74X0EDYJAVBJJK2DY8
  - TKT-01M6EFQN91YGXHVP9NF32BK21W
  - TKT-01M6EFQNBJCQP8BGKX658XVNBS
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:48Z
updated_at: 2026-11-02T15:59:32Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-plan-m7
  name: ""
extensions: {}
---

## Description

The minimap knows which areas are explored. The plugin sends that, coarse
and only when it grows, so the controller can prefer routes over known
ground and avoid sending the bot across unexplored land to reach a goal.

This needs a protocol message (or a field on an existing one) for the
explored cells and a fixture. The controller keeps it per world.

### Extended for M7 (docs/EXPLORATION.md, "What Tas knows about an area")
Exploration needs three coverage kinds kept apart, each with provenance: map knowledge (minimap or shared), measured terrain coverage (atlas samples first received during a window), and visited ground (where Tas actually walked, with elevation). Loaded terrain is not personal visual coverage, and a shared map reveal is not discovery. This ticket stays under its current parent and is now the umbrella for three one-PR children:

- TKT-01M6EFQN74 (Atlas: first-seen time and a measured-coverage query): the number M7 goal 7.3 measures
- TKT-01M6EFQN91 (Report and keep the ground the bot actually walked): visited ground and measured travel
- TKT-01M6EFQNBJ (Report minimap explored cells as map knowledge): this ticket's original scope and its two original criteria

### Files
- internal/atlas/store.go, internal/app/atlas.go, plugin/Renki.Core/Movement/RouteMemory.cs, plugin/Renki.Plugin/BodyTasks.cs, plugin/Renki.Plugin/TerrainReporter.cs, internal/protocol/terrain.go, internal/protocol/task.go

### Verification
Done when the three children are done. Offline: `just check controller test-packages ./internal/atlas ./internal/app ./internal/protocol ./internal/session` and `just check plugin test-core RouteMemoryTests`. Live: the minimap child's capture after `just vm deploy`.

## Acceptance criteria

- [ ] The plugin reports explored map cells when they change, with a protocol fixture
- [ ] The controller keeps the explored area per world and can say whether a straight route crosses unexplored land
- [ ] Measured terrain coverage (TKT-01M6EFQN74) and visited ground (TKT-01M6EFQN91) are done, and map knowledge is never counted as measured coverage

## Notes

**agent:codex/renki-exploration** at 2026-10-30T15:25:32Z

Reused as the coverage foundation for TKT-01M66PFAYA1NTW2JHNRDKBK42E — Exploration goals after M2 and M3. docs/EXPLORATION.md proposes distinguishing minimap/shared knowledge, actual observation coverage, visited ground and directed route evidence, with provenance, age, height and incomplete-scan handling. A revealed minimap cell alone is not personal survey, object absence or proof of a safe traversable route.

The new exploration implementation tickets depend on both milestone epics and remain draft. This existing ticket retains its parent, acceptance criteria and draft status; no exploration runtime or automatic behavior is enabled by the design work.

**agent:codex/renki-atlas** at 2026-10-30T15:40:14Z

The durable storage proposal is tracked by TKT-01M66Q5SFWV99JCGJXVARVRJH7 — Durable versioned world atlas and portable mapping data; see docs/ATLAS.md. Its first slice persists loaded terrain and does not implement personal exploration coverage. Keep loaded map knowledge, minimap/shared knowledge, actual observed coverage, visited ground and directed route evidence distinct. Reuse stable world/epoch identity, provenance and truthful freshness from atlas capture; receiving a cached tile does not prove fresh observation. Existing parent, criteria and draft status remain unchanged.
