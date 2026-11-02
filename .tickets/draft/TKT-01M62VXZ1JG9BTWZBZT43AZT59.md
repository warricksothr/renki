---
schema: 4
id: TKT-01M62VXZ1JG9BTWZBZT43AZT59
title: "Deliver to the nearest storage: home, a ship or a waypoint"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
assignees: []
milestone: M7
parent: TKT-01M62VXYQB6ZDBQE4E6ERGN83C
origin: null
dependencies:
  - TKT-01M62VXYZMVEP74QVV9SRKDWGH
  - TKT-01M62VXZ0J9Q2WH4XN4E6N7F27
  - TKT-01M62VXYWJHB5BC9VGCBM0PVXD
  - TKT-01M62VXYY6F094WXQS3FBQ57RF
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:48Z
updated_at: 2026-11-02T16:15:35Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-plan
  name: ""
extensions: {}
---

## Description

Deliver-when-full goes home at first. Later it picks the best storage it
can reach: the base, the ship the bot came on (a ship has a hold), or a
waypoint with chests close by.

Each candidate is scored on distance, the room it has (from chest memory),
whether the way crosses unexplored land (strongly avoided), and danger
zones. When nothing is good enough, the bot says so rather than taking a
long, risky trip.

### Status and fit (2026-11-02)

This is not on an M5 exit goal, and it depends on TKT-01M62VXZ0J (Know which parts of the map the bot has explored), which is in M7. It belongs with the expedition work in M7, where the bot is far enough from home for the choice to matter.

What exists: `gather` with `returnHome` and `marker_cargo` deliver home or to a `storage` waypoint only (internal/behaviour/gather.go, internal/behaviour/marker_cargo.go). Remembered chests carry positions, contents and ages (internal/home/containers.go), and danger zones carry scores (internal/home/danger.go). Ships are not reported as containers today.

### Files

- `internal/behaviour/gather.go` (`deliver`, `goHome`) and `internal/behaviour/stow.go`
- `internal/home/containers.go`, `internal/home/danger.go`, `internal/home/waypoints.go`
- the explored-area layer from TKT-01M62VXZ0J

### Verification

`just check controller test-packages ./internal/behaviour ./internal/home` with tests for: a storage waypoint chosen over a farther home, a route over unexplored land avoided, a danger zone avoided, and a "nothing good enough" message. `just check ci`.

## Acceptance criteria

- [ ] Deliver-when-full can choose the ship's hold or a waypoint's chests instead of home
- [ ] A route over unexplored land or through a danger zone is avoided where another will do
- [ ] When no storage is good enough, the bot says so

## Notes

**agent:claude-code/renki-plan** at 2026-11-02T16:15:35Z

Applied on 2026-11-02 by the planning pass: milestone set to M7, to match the exit goal in docs/ROADMAP.md this ticket serves. This supersedes any earlier note that only proposed the move. The owner can move it back.
