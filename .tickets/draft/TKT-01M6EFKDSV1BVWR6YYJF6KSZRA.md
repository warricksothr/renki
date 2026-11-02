---
schema: 4
id: TKT-01M6EFKDSV1BVWR6YYJF6KSZRA
title: "Controller: sort misplaced items into assigned chests while idle"
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

M5 goal 5.3 (docs/ROADMAP.md): assigned items reach their chests, including items already sitting in the wrong chest. TKT-01M62VXZ84 (Controller: assign every item to chests, with a loose-items chest) covers what the bot carries home. This ticket covers sorting: moving items that are already in a chest to the chest they are assigned to.

### What to build

A `sort` template on the behaviour engine, run through an idle trigger at home like `marker_cargo` (docs/BEHAVIOURS.md, "Rest while idle at home" and the idle triggers):

- From remembered container snapshots no older than a configurable age (default 10 min), find units of assigned items outside their chests. Don't-touch chests are never taken from.
- For each, withdraw from the wrong chest and deposit into the assigned chests through the existing `stow` routine, with matched withdraw and deposit receipts and a fresh pack observation after each. A full destination falls back to the next assigned chest, then reports the shortfall the way stow does.
- Runs only while Tas is idle at home and no player is within 3 m of the source chest (a player using a chest may be mid-change). Any owner order, retreat or new task interrupts through the engine's body lease. Items in the pack when interrupted are left for the next stow or cargo run rather than dropped.
- Bounded per run: at most 8 chest pairs and 6 minutes, with a cooldown of at least 5 minutes.

### Files

- `internal/behaviour/marker_cargo.go` and `internal/behaviour/stow.go`: the deposit path and receipts to reuse
- `internal/behaviour/idle.go`: idle triggers and dwell
- `internal/home/containers.go`: remembered snapshots and ages
- the assignment table TKT-01M62VXZ84 adds
- `docs/BEHAVIOURS.md`: a section for the template

### Verification

`just check controller test-packages ./internal/behaviour ./internal/home` with tests for: a misplaced stack moved to its chest, a don't-touch source skipped, a stale snapshot ignored, a player near the source chest delays the run, an owner order interrupts with items left in the pack, and a full destination reported once. `just check ci`. Live evidence comes from the assignments section of the M5 upkeep soak.

## Acceptance criteria

- [ ] A misplaced stack seen in a fresh snapshot is moved to its assigned chest with matched withdraw and deposit receipts, in a test
- [ ] Don't-touch chests are never taken from, and snapshots older than the age limit are ignored, in tests
- [ ] A player within 3 m of the source chest delays the run, and an owner order interrupts it with items left in the pack, in tests
- [ ] Each run is bounded to 8 chest pairs and 6 minutes, and docs/BEHAVIOURS.md documents the template
