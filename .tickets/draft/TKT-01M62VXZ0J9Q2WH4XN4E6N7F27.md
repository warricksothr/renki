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
assignees: []
milestone: null
parent: TKT-01M62VXYQB6ZDBQE4E6ERGN83C
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:48Z
updated_at: 2026-10-30T15:40:14Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:codex/renki-atlas
  name: ""
extensions: {}
---

## Description

The minimap knows which areas are explored. The plugin sends that, coarse
and only when it grows, so the controller can prefer routes over known
ground and avoid sending the bot across unexplored land to reach a goal.

This needs a protocol message (or a field on an existing one) for the
explored cells and a fixture. The controller keeps it per world.

## Acceptance criteria

- [ ] The plugin reports explored map cells when they change, with a protocol fixture
- [ ] The controller keeps the explored area per world and can say whether a straight route crosses unexplored land

## Notes

**agent:codex/renki-exploration** at 2026-10-30T15:25:32Z

Reused as the coverage foundation for TKT-01M66PFAYA1NTW2JHNRDKBK42E — Exploration goals after M2 and M3. docs/EXPLORATION.md proposes distinguishing minimap/shared knowledge, actual observation coverage, visited ground and directed route evidence, with provenance, age, height and incomplete-scan handling. A revealed minimap cell alone is not personal survey, object absence or proof of a safe traversable route.

The new exploration implementation tickets depend on both milestone epics and remain draft. This existing ticket retains its parent, acceptance criteria and draft status; no exploration runtime or automatic behavior is enabled by the design work.

**agent:codex/renki-atlas** at 2026-10-30T15:40:14Z

The durable storage proposal is tracked by TKT-01M66Q5SFWV99JCGJXVARVRJH7 — Durable versioned world atlas and portable mapping data; see docs/ATLAS.md. Its first slice persists loaded terrain and does not implement personal exploration coverage. Keep loaded map knowledge, minimap/shared knowledge, actual observed coverage, visited ground and directed route evidence distinct. Reuse stable world/epoch identity, provenance and truthful freshness from atlas capture; receiving a cached tile does not prove fresh observation. Existing parent, criteria and draft status remain unchanged.
