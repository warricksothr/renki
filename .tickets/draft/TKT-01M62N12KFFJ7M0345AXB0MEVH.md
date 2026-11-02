---
schema: 4
id: TKT-01M62N12KFFJ7M0345AXB0MEVH
title: "Controller: no-chop zones the bot leaves alone"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - chat
assignees: []
milestone: M5
parent: null
origin: null
dependencies:
  - TKT-01M62N12JW8TFN12MAT8M1ADZ8
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T01:40:10Z
updated_at: 2026-11-02T15:59:16Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-plan-m5
  name: ""
extensions: {}
---

## Description

Later functionality, after M2's benchmark (Nimah, 2026-10-28): places the owner wants left alone (trees around the base, a garden, a scenic grove). Marked by command where the owner stands ("don't chop here", with a radius), kept per world, shown on the panel, and respected when choosing trees.

This is not on an M5 exit goal. It fits M5 because it protects the base while the bot works near it, and it is small. It waits on the M2 gather work (TKT-01M62N12JW (Controller: go chop wood)) so it does not collide with that ticket's changes to tree selection.

### What exists

- Named waypoints already support circular zones (a `Radius` over 0), are kept per world, are drawn by the panel and are edited from chat and the panel (internal/home/waypoints.go, docs/WAYPOINTS.md). Kinds are a fixed list: `place`, `storage`, `cooking`, `wood`, `stone`, `flint`, `berries`, `hunting`, `ore`.
- Tree choice for `gather` and `go chop wood` ranks candidates by distance plus a danger penalty (internal/behaviour/conditions.go, `targetCost`; internal/behaviour/gather.go; internal/behaviour/wood.go `planWood`).

### What to build

- A `nochop` waypoint kind (a zone, radius 3 to 60 m), so storage, drawing and editing come from the waypoint code.
- Chat for owners: `<name> don't chop here [radius]` marks a zone named after the speaker's position, `<name> chop here again` clears the zone the speaker stands in.
- Tree and log selection in `gather`, `planWood` and `collect` skips any standing tree whose position is inside a zone. Logs from a tree felled outside a zone that roll into one may still be split, since leaving them would block the chain.
- The panel draws `nochop` zones in their own colour.

### Files

- `internal/home/waypoints.go`: `WaypointKinds` and validation
- `internal/behaviour/gather.go`, `internal/behaviour/wood.go`, `internal/behaviour/conditions.go`: tree choice
- `internal/chat/waypoints.go`: waypoint chat commands
- `panel/src/controller.jsx`: waypoint drawing
- `docs/WAYPOINTS.md` and `docs/BEHAVIOURS.md`

### Verification

`just check controller test-packages ./internal/home ./internal/behaviour ./internal/chat` with tests for: a tree inside a zone never chosen, a tree just outside chosen, a rolled log inside a zone still split, and a non-owner refused. `just check panel ci` for the drawing. `just check ci`.

## Acceptance criteria

- [ ] An owner can mark and clear a no-chop zone where they stand
- [ ] Choosing trees to chop skips anything inside a zone
- [ ] The panel draws the zones
