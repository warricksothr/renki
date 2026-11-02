---
schema: 4
id: TKT-01M6EFKDQQZ5PMC5YAY1ZAVJJW
title: "Panel: edit chest assignments, loose and don't-touch chests"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
assignees: []
milestone: M5
parent: TKT-01M62VXZ2FWDX3D4Z6XX1MB5A6
origin: null
dependencies:
  - TKT-01M62VXZ84SA9TSJC6NJNPV8KJ
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:56:13Z
updated_at: 2026-11-02T15:56:13Z
created_by:
  id: agent:claude-code/renki-plan-m5
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m5
  name: ""
extensions: {}
---

## Description

M5 goal 5.3 (docs/ROADMAP.md): assigned items reach their chests. TKT-01M62VXZ84 (Controller: assign every item to chests, with a loose-items chest) adds the assignment table and its chat command. Owners also need to see and edit the table in the panel, where the chests are on the map.

### What to build

A chest-assignments editor in the panel:

- The table: each item or selector (`wood`, `ore`, `food`, an exact prefab) with its ordered chests, the loose chest, and the chests marked don't-touch.
- Pick a chest by clicking it on the map. Chests come from remembered containers (`GET /api/memory/containers`) and current perception, as the hover in TKT-01M64ZZY (Panel: show remembered chest contents on hover) already shows.
- Add, reorder and remove assignments; mark or clear loose and don't-touch. Saving goes through the controller endpoint TKT-01M62VXZ84 adds, with `X-Renki-Act`.
- A rejected save shows the reason and changes nothing.

### Files

- `panel/src/controller.jsx`: map hover for remembered chests and the existing control sections
- `panel/ui/scenarios.mjs` and `panel/ui/fixtures/`: browser scenarios
- `internal/app/memory.go`: the containers endpoint the editor reads
- `docs/PANEL.md`: the new section

### Verification

`just check panel ci` with UI scenarios for listing, adding, reordering and removing an assignment, marking loose and don't-touch, and a rejected save. `just check ci`. No game is needed.

## Acceptance criteria

- [ ] The panel shows the assignment table, the loose chest and don't-touch chests
- [ ] An owner can add, reorder and remove an assignment by picking chests on the map, covered by a UI scenario
- [ ] Marking and clearing loose and don't-touch saves through the controller, covered by a UI scenario
- [ ] A rejected save shows the reason and changes nothing
