---
schema: 4
id: TKT-01M6EFHEK3PB5CSHWBHBW0P7AX
title: Routes home detour around remembered danger zones
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - movement
  - safety
assignees: []
milestone: M3
parent: TKT-01M62N2557R401S3E6JECSAPN4
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:55:08Z
updated_at: 2026-11-02T15:55:08Z
created_by:
  id: agent:claude-code/renki-plan-m3
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m3
  name: ""
extensions: {}
---

## Description

Goal 3.4 in docs/ROADMAP.md: a replay test shows routes preferring to avoid known danger zones, and one live trip detours around a zone recorded earlier that day.

Danger zones exist and are shown on the map (TKT-01M62N25C, Controller: danger zones from where the bot was hurt, done). They only change which tree or pickable is chosen first (`home.DangerRisk`, used by `internal/behaviour/conditions.go` and `internal/goal/food_*.go`). docs/MEMORY.md says so: "Explicit site travel, delivery, return-home and native navigation do not yet route around danger". `home.DangerRisk` says it "does not certify routes".

### Outcome

- A pure function in `internal/home` (beside `DangerZones` and `DangerRisk` in `internal/home/danger.go`) that, given a start, a destination and the current world's zones, returns either nothing (the straight trip does not cross a zone at or above a score threshold) or one or two intermediate waypoints that keep the straight legs outside those zones. Bound the detour: at most 1.5 times the straight distance and at most 60 m extra, else return nothing and say so. A detour waypoint is a hint for the native walker, not a safety guarantee: all native support, water and hostile checks still apply.
- `Keeper.GoHome` in `internal/home/home.go` uses it: when a detour is returned, it sends the trip as legs (`move_to` the waypoint, then home) and logs `danger detour` with the zones and waypoints. Read how `homeward` IDs and the chat brain's `TaskDone` follow a trip home before choosing between chaining legs in the controller and adding a `via` list to `move_to`. If you add `via`, it is a protocol change: `docs/BRIDGE-PROTOCOL.md`, `internal/protocol/task.go` (`MoveToArgs`), `plugin/Renki.Core/Protocol/TaskArgs.cs` and the plugin's move task in `plugin/Renki.Plugin/BodyTasks.cs`.
- A trip that does not cross a zone is sent exactly as today.
- Other travel (gather site, delivery, food trips) stays as it is; a later ticket can reuse the function.

### Files

- `internal/home/danger.go`, `internal/home/home.go`, and their tests in `internal/home`
- `docs/MEMORY.md` ("Injuries and danger"): replace the "do not yet route around danger" sentence for return-home.

### Verification

- A replay test in `internal/home` that loads recorded zones (injuries from a retained `memory.json` or audit, as a fixture in `internal/home/testdata`) and asserts the detoured legs stay outside every zone above the threshold, while a trip that avoids them is unchanged.
- `just check controller test-packages ./internal/home ./internal/chat ./internal/app`
- `just check ci`

## Acceptance criteria

- [ ] A pure function in internal/home returns bounded detour waypoints that keep straight legs outside zones above the threshold, or nothing with a reason
- [ ] A replay test with recorded zones shows the detour avoiding them, and a trip that does not cross a zone is unchanged
- [ ] GoHome sends the detour as legs and logs danger detour with the zones and waypoints; a test covers it
- [ ] docs/MEMORY.md says return-home now detours and other travel does not
- [ ] just check ci passes
