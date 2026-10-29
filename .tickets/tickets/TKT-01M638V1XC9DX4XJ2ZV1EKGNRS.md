---
schema: 4
id: TKT-01M638V1XC9DX4XJ2ZV1EKGNRS
title: "Follow: stuck outside a wall, the leader inside, path straight through"
type: bug
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-resume
  branch: t3/wall-avoidance
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 33d5c9ee55c3da7ea78be1f4d0bacbee756d9fd6
  session: null
  claimed_at: 2026-10-29T17:41:35Z
  expires_at: null
archive: null
created_at: 2026-10-29T07:26:24Z
updated_at: 2026-10-29T17:59:52Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:codex/renki-resume
  name: ""
extensions: {}
---

## Description

Seen on the VM, 2026-10-29 01:39-01:40, follow task t-1: Nimah stood inside the house at (-31.54, 36.09, 12.45). The bot stood outside the east wall at about (-28.1, 36.09, 11.55), 3.5 m away. The wall runs from (-28.6, 12.0) to (-29.4, 10.1), per the layout. Every re-path returned 2 waypoints, a straight line through the wall. The walker went through 9 recoveries (Jump, Strafe, Repath, Climb, three rounds) and gave up `stuck`, which sent it home.

Open question: why did the navmesh give a straight path through a wall? Either the navmesh does not include that wall, there is a gap the agent fits through and the bot does not, or GetPath returned a partial path that we treat as whole. Read the path trace's `end` (new: it is set when a path stops more than 1 m short) the next time this happens.

Follow should also not count it as stuck when it is already within its distance plus a little of the leader and cannot get closer.

## Acceptance criteria

- [ ] The cause of the straight path through the wall is known and written down
- [ ] A follower that is near the leader but cannot get closer waits, not recovers

## Implementation plan

### Evidence
The 17:39:47Z come-here attempt first had no navmesh path and drove a straight fallback toward the east wall. Later paths ended 21.9 m short on the house floor. GetPath currently allows partial paths, and Walker.Toward falls through to the original goal when the waypoint list ends, driving past the valid partial path and into a wall. This is distinct from raised-goal arrival.

### Approach
Validate every candidate path against the already measured, height-aware wall segments, including the final segment. Reject a route that crosses a known wall or stops far short instead of driving past its end. Use a bounded local A* fallback with 0.75 m cells, ground support, step-height limits, capsule clearance, water rejection and normal door handling. Budget search over physics frames; keep different floor levels separate. No stale collision map is persisted: current walls and physical probes validate each local plan. Keep navmesh for ordinary travel and preserve raised-goal recovery.

Add deterministic tests for the real house layout, doors/openings, stacked floors, enclosed/no-route goals, clearance and bounded search, and for stopping at partial path ends. Capture bounded path and probe-cost traces. Live-test walking out and returning before resuming the staged retreat and hauling work. just check ci and just check review gate landing/deployment.

## Notes

**agent:codex/renki-resume** at 2026-10-29T17:53:21Z

Cause confirmed from the 17:39:47Z–17:40:43Z audit window and installed game API: the first GetPath failed, so the walker drove a straight fallback into the wall. Later GetPath calls succeeded with partial paths ending 21.9 m short; requireFullPath was false, and Toward invented a direct segment to the original goal after the path ended.

Implemented height-aware validation against current measured walls and a bounded, supported local A* fallback (0.75 m cells, 4096 expansions, 48 m per axis, 8 s; eight expansions and a 2 ms target budget per physics frame). Physical probes check floor support at the end and midpoint, step height, body clearance and water; normal doors remain usable. Partial endpoints stop and replan instead of driving past them. Raised-goal detours now actually request complete navmesh paths and validate walls. Close followers blocked by a wall wait within the usual one-metre slack.

Six regressions pass, including all 71 measured house walls, stacked floors, clearance, backing away from a wall, bounded no-route search, partial endpoint steering and near-leader waiting. just check plugin test-core LocalRouteTests and just check ci passed (133 C# tests). Live wall/door/stair and performance checks remain to be observed after deployment. Chest-transfer work is safely retained in the stash named Hauling chest transfers in progress before wall routing fix; resume it after navigation and retreat verification.

**agent:codex/renki-resume** at 2026-10-29T17:59:52Z

Live fe9384a: come-here t-1 left via west doorway and reached Nimah east of house in 16 s (17:57:37Z–17:57:53Z), 1121 expansions, 238.69 ms CPU, 12.95 ms worst planning frame. Return-home t-2 failed after eight seconds of searching from 35 m away despite a wall-safe partial navmesh route to within 2 m horizontally, under the floor. Evidence: .checks/audit-window-qfLEXg.jsonl. Preserve useful wall-safe prefixes and stop/replan near their endpoint so local search can spend its bounded budget on the missing final leg. Live return verification remains outstanding.
