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
  branch: t3/navigation-cache
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: bfe3f88dd5fc07bad7e0bd7f6725e63b12dd523c
  session: null
  claimed_at: 2026-10-29T17:41:35Z
  expires_at: null
archive: null
created_at: 2026-10-29T07:26:24Z
updated_at: 2026-10-29T18:35:05Z
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

- [x] The cause of the straight path through the wall is known and written down
- [ ] A follower that is near the leader but cannot get closer waits, not recovers

## Implementation plan

### Evidence
The 17:39:47Z come-here attempt first had no navmesh path and drove a straight fallback toward the east wall. Later paths ended 21.9 m short on the house floor. GetPath currently allows partial paths, and Walker.Toward falls through to the original goal when the waypoint list ends, driving past the valid partial path and into a wall. This is distinct from raised-goal arrival.

### Approach
Validate every candidate path against the already measured, height-aware wall segments, including the final segment. Reject a route that crosses a known wall or stops far short instead of driving past its end. Use a bounded local A* fallback with 0.75 m cells, ground support, step-height limits, capsule clearance, water rejection and normal door handling. Budget search over physics frames; keep different floor levels separate. No stale collision map is persisted: current walls and physical probes validate each local plan. Keep navmesh for ordinary travel and preserve raised-goal recovery.

Add deterministic tests for the real house layout, doors/openings, stacked floors, enclosed/no-route goals, clearance and bounded search, and for stopping at partial path ends. Capture bounded path and probe-cost traces. Live-test walking out and returning before resuming the staged retreat and hauling work. just check ci and just check review gate landing/deployment.

### Cached navigation geometry
The owner authorized cached obstacle boxes/circles with mark-and-sweep refresh, nearer-first and travel-corridor priority. Keep the cache in the plugin and bounded in space/count/work per frame. Discover nearby/ahead/far colliders at different cadences; cache static upright box/capsule primitives, leave meshes, moving objects and doors to live physics. Transform/shape/enabled changes dirty records, destroyed objects are swept, and session/world changes clear the cache. Primitive bounds are conservative early blockers only; live support, water and capsule queries remain authoritative for uncached space. Use a spatial hash rather than testing every obstacle. Memoize exact repeated support samples within each planning physics step without merging floors; resample next step because floors and meshes can change independently of indexed solids. Revalidate candidate cached blockers against live collider stamps before rejecting a leg, so a delayed sweep cannot preserve a demolished wall. Spread discovery over frames, with the oldest due near/ahead/far pass selected to avoid starvation. Remove allocating All physics calls in the planner in favour of bounded nonallocating queries that fail closed on saturation. Log cache hits, actual physics calls, bounds rejections, cache maintenance and planning CPU; compare the same exact house exit/return. Verify geometry corners/height/backing out, mutations/removal, update priority and bounded caches in Core tests. Preserve normal navmesh and all existing movement safety.

## Notes

**agent:codex/renki-resume** at 2026-10-29T17:53:21Z

Cause confirmed from the 17:39:47Z–17:40:43Z audit window and installed game API: the first GetPath failed, so the walker drove a straight fallback into the wall. Later GetPath calls succeeded with partial paths ending 21.9 m short; requireFullPath was false, and Toward invented a direct segment to the original goal after the path ended.

Implemented height-aware validation against current measured walls and a bounded, supported local A* fallback (0.75 m cells, 4096 expansions, 48 m per axis, 8 s; eight expansions and a 2 ms target budget per physics frame). Physical probes check floor support at the end and midpoint, step height, body clearance and water; normal doors remain usable. Partial endpoints stop and replan instead of driving past them. Raised-goal detours now actually request complete navmesh paths and validate walls. Close followers blocked by a wall wait within the usual one-metre slack.

Six regressions pass, including all 71 measured house walls, stacked floors, clearance, backing away from a wall, bounded no-route search, partial endpoint steering and near-leader waiting. just check plugin test-core LocalRouteTests and just check ci passed (133 C# tests). Live wall/door/stair and performance checks remain to be observed after deployment. Chest-transfer work is safely retained in the stash named Hauling chest transfers in progress before wall routing fix; resume it after navigation and retreat verification.

**agent:codex/renki-resume** at 2026-10-29T17:59:52Z

Live fe9384a: come-here t-1 left via west doorway and reached Nimah east of house in 16 s (17:57:37Z–17:57:53Z), 1121 expansions, 238.69 ms CPU, 12.95 ms worst planning frame. Return-home t-2 failed after eight seconds of searching from 35 m away despite a wall-safe partial navmesh route to within 2 m horizontally, under the floor. Evidence: .checks/audit-window-qfLEXg.jsonl. Preserve useful wall-safe prefixes and stop/replan near their endpoint so local search can spend its bounded budget on the missing final leg. Live return verification remains outstanding.

**agent:codex/renki-resume** at 2026-10-29T18:05:10Z

Live dd9ef44 return-home t-2 succeeded at 18:03:59Z after a safe navmesh prefix, an off-level reposition and short local searches (434 expansions / 84 ms and 364 / 68.18 ms). Ended at (-36.5,36.09,8.26), 25/25 HP. First order immediately after restart still found no usable navmesh and exhausted local search; GetPath builds tiles asynchronously and unused tiles expire after 30 s. Recheck for newly usable complete paths or useful safe prefixes once per second during planning, preserving the existing search and deadline when no path is usable. Close-follower waiting and cold-start retry remain live checks; implementation tests and just check ci pass.

**agent:codex/renki-resume** at 2026-10-29T18:16:46Z

Live a04704d: exact east-side walk t-6 started at 18:14:37Z, used local wall-aware routing through west door, recovered at the door, adopted a newly usable navmesh during local search at 18:14:52Z and succeeded at 18:15:08Z. Final pos (-2.84,33.19,9.5), 25/25 HP. Return-home t-7 succeeded at 18:15:44Z; final (-37.03,36.09,7.92), 25/25 HP, still 15 seeds. This exercises the asynchronous-navmesh recheck and both directions through the actual house. Evidence .checks/audit-window-naIMBv.jsonl. Close-follow waiting through a wall, cancel during planning, newly built/demolished walls and a blocked door remain staged checks; only the root-cause AC is checked.

**agent:codex/renki-resume** at 2026-10-29T18:35:05Z

Implemented the owner-requested mark/refresh/sweep cache on t3/navigation-cache. Static upright box and capsule solids persist in a spatial hash, with bounded near/ahead/far discovery, nearer-first and forward-corridor rebuild priority, at most 4096 records, 128 discovered records processed, 16 rebuilds and 32 sweep records per rendered frame. Candidate cached blockers validate their current collider stamp before rejection; changed bounds are dropped before rebuilding, and rebuilds stamp the geometry actually measured. Doors, rigidbodies, tilted/sheared primitives and meshes remain live physics decisions. Exact support samples share work within a planning physics step only, preserving floor levels and seeing changed/missing floors next step. Planner physics queries are nonallocating and fail closed on saturated buffers. World/session resets clear the cache and retained buffers.

NavigationCacheTests adds 14 cases covering rotated/rounded geometry, circles, height/backing-out, index moves/removals/deduplication, stale-bound rejection, refresh priority, bounded support memory and floor lifetime. An identical deterministic detour uses fewer simulated live queries; actual Unity CPU savings are not yet measured. just check plugin build and just check plugin test-core NavigationCacheTests passed; just check ci passed all 158 C# tests including interop plus the controller race suite and both builds. docs/TESTING.md now names cache limits, trace fields and remaining staged checks.

Live testing remains deferred at the owner's request: repeat the exact house exit/return, changed walls/floors/doors, stop during planning and near-leader waiting, and compare planner plus maintenance CPU. Do not close the remaining follower criterion from Core tests. M3 retreat implementation remains staged for a controlled Greyling check; M2 next implementation is the ready behaviour-engine ticket TKT-01M62VXYSN4RQV75HTA3BSD162 (Controller: behaviour engine: templates, instances, triggers, repeat), followed by chest/work-site memory and loot/stow orchestration. The milestones remain incomplete.
