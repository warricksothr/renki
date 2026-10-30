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
  actor: agent:codex/renki-milestones
  branch: t3/progressive-home-routing
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 8d746974e69c49dc9878555b3114a1e327c0665e
  session: null
  claimed_at: 2026-10-30T17:24:39Z
  expires_at: null
archive: null
created_at: 2026-10-29T07:26:24Z
updated_at: 2026-10-30T17:30:03Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:codex/renki-milestones
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

Continue the owner's autonomous M2/M3 work by fixing the prerequisite long-distance return-home failure. The retained resupply audit shows a 375 m goal handed to the 48 m local planner with zero expansions or probes after the initial navmesh query failed. Keep local CPU, support, water and clearance bounds. For distant goals, permit a fully probed short frontier leg that makes measurable progress toward the actual destination, preserving the destination and replanning at the leg endpoint. Heights must follow live support rather than interpolating toward an unloaded floor. Wait briefly for asynchronously requested navmesh tiles even if the local search finishes without a route. Test distant sloped trips, obstacle detours, enclosed starts, unknown support/water, bounded work and final-floor arrival; then run just check ci, commit, just check review, land and deploy. Repeat just vm resupply-food and record actual travel, transfer, eating and cleanup evidence. Keep unrelated close-follow and terrain-mutation criteria open.

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

**agent:codex/renki-resume** at 2026-10-29T22:08:25Z

Live checks on 91b4835 after the startup fix:
- First in-world state seq117 at 22:02:57.890Z preceded inventory seq118 at 22:02:57.903Z. The unchanged 15-seed inventory was available without a reconnect or pack mutation.
- Stow t-1 at 22:03:12–22:03:14Z deposited exactly five BeechSeeds into 1:30592 and preserved ten. The chest's one ArmorRagsChest was unchanged. Withdraw t-2 at 22:03:33Z restored the pack to 15 seeds and the chest to its original contents.
- Initial collection move t-3 failed safely inside the house: local search exhausted 215 nodes /1720 probes at 22:03:45Z (47.72 ms total, 2.11 ms worst frame), with no item mutation. Explicit door interaction t-4 moved Tas from (-36.07,36.09,7.82) to (-37.69,36.09,9.52) and opened the west door. That combines opening and repositioning; it does not isolate the navigation cause.
- Retried stone collection then completed t-5 move to site, t-6 use Stone, t-7 take one Stone, t-8 approach chest, t-9 deposit exactly one Stone, t-10 return to site (22:04:58–22:05:12Z). Chest held Stone1 plus the unchanged rag armour. Tas stayed 25/25 HP. This is a controlled full loop after doorway staging, not proof of reliable unattended house exits.
- The next flint job used its pickable at 22:06:02Z and auto-pickup put Flint1 in the pack at 22:06:03.489Z. A stale perceived product caused pickup t-13 to arrive 97 ms later and be rejected invalid_target: no drop in range. The Flint is safe in the pack, but delivery stopped. No full flint-loop pass is claimed yet.

Follow-up t3/collection-pickup-race checks measured whole-yield gains before choosing stale product drops. A rejected missing product can be handled only while a known pickable's output is pending; fresh inventory evidence is still required. Added regressions cover both inventory-before-perception and request-versus-auto-pickup races, plus refusal without inventory evidence. Also handle a rejected vanished chest as an unmutated fallback, with permission rejection still terminal. Focused just check controller test-packages ./internal/behaviour passed. Gate/review/controller deploy and a new flint target follow; final live receipt will be attached to the follow-up PR. Background jobs remain uninstalled and check definitions are disabled between runs.

**agent:codex/renki-resume** at 2026-10-29T22:16:30Z

The owner's latest observation is repeated running underneath the base. The 22:14:31–22:15:14Z trial did complete: chest move t-2 failed after a 3199-node local search; the new one-shot home fallback t-3 replanned, entered through the west door, and t-4 deposited Flint2 with inventory/chest receipts. HP25, pack BeechSeeds15, chest Stone1/Flint2/ArmorRagsChest1. Evidence: .checks/audit-window-LtoncZ.jsonl. The fallback reduces job failure but does not establish good routing.

Code inspection found PathFollower.Next advances every waypoint using flat distance alone, including the final waypoint. LocalRoute keeps floor heights distinct, but following the resulting path can discard that distinction. Add height-aware waypoint arrival and stacked-floor regression checks before further live route tests. No successful-path persistence exists today; do not claim reuse of previous routes. Keep known-good entrance reuse as a separate navigation follow-up unless current evidence requires it.

**agent:codex/renki-resume** at 2026-10-29T22:26:17Z

Fresh one-target flint loop passed on the height-aware waypoint build with the bounded stow home fallback: 22:23:51–22:25:01Z, t-2 moved from home to (-28.15,30.43,42.11), t-3 used pickable 1:39745, t-4 took exactly Flint1, t-5's direct chest approach failed unreachable, t-6 went to configured home through the west door, and t-7 deposited Flint1. Task-matched receipt and newer pack observation confirmed chest Stone1/Flint3/ArmorRagsChest1 and pack BeechSeeds15. Every recorded state was 25/25 HP. No manual door staging occurred in this loop. Both temporary test definitions were deleted afterward. Evidence: .checks/audit-window-IM0XXq.jsonl.

This is a full bounded collection/delivery pass. It is not efficient/reliable direct chest routing: the direct approach still searched the underside and failed before the home fallback. Height-aware following prevents consuming waypoints from another floor; it does not add route memory. A reusable, live-validated entrance/successful-path hint is still needed, along with repeated unattended route checks. Tree cutting, mining, crafting, default loot, deliberate-drop detection, leader/work-site resumption and live multi-chest/full/ward checks remain open.

**agent:codex/renki-resume** at 2026-10-29T22:50:55Z

Implemented bounded, session-local memory of successful point-movement trails, floor-aware forward/reverse joins, live support/water/wall/capsule validation and immediate-leg revalidation. Point moves honor stopping reach beside solid goals. Follow/flee policies remain unchanged.

Live 2026-10-29 22:47:15–22:50:19Z, HP 25 throughout: outgoing t-1 succeeded (36 recorded points); home t-2 succeeded through doorway with normal planning; repeated outgoing t-3 reused route 1 and succeeded in 5.5 s (68 validation samples, 4.25 ms total, 1.85 ms worst slice). Chest t-4 validated reverse reuse (63 samples, 2.38 ms total, .39 ms worst slice) and succeeded, but first deployed version prematurely invalidated near a tight doorway turn and used a short local fallback. Source fix now counts consumed validated waypoints as progress on replay; stationary legs still trigger recovery. Retest final build before claiming uninterrupted reverse reuse. Initial local search: 98 expansions, 778 probes, 47.85 ms total; one 19.76 ms slice exceeds the 2 ms target, so do not claim a hard per-frame bound.

Evidence: .checks/audit-window-11vJwZ.jsonl, .checks/audit-window-lrLJ3u.jsonl, .checks/route-t3-events.json, .checks/route-t4-events.json. Real geometry mutations and longer unattended/reconnect/cancel checks remain staged. Navigation ticket AC2 remains open.

Owner supplied new flint axe in rag chest 1:30592; t-5 withdrew exactly one AxeFlint successfully at 22:50:19Z. Chest retains Stone 1, Flint 3 and ArmorRagsChest 1; other contents preserved.

**agent:codex/renki-resume** at 2026-10-29T22:55:14Z

Final waypoint-progress build live check 2026-10-29 22:54:47–22:55:03Z: level terrace (-46,33.9,16) outgoing t-5 reused route 3, succeeded in 3.7 s, 45 samples/1.61 ms total/.34 ms worst slice. Reverse chest t-6 reused route 5, crossed west doorway, closed door behind, succeeded in 4.1 s, 48 samples/1.84 ms total/.37 ms worst slice. Neither required fallback; HP stayed 25. Evidence .checks/route-level-out.json and .checks/route-level-return.json. Earlier steep exterior start t-4 legitimately stalled its replay and reverted to normal planning, which completed; .checks/route-corrected-return.json. Do not claim replay handles that slope or a hard 2 ms bound. Final source gate .checks/20261029T225019Z-McEZIK.log; committed review .checks/20261029T225100Z-67FUwU.log: no findings. Route hints still clear on reconnect.

**agent:codex/renki-milestones** at 2026-10-30T17:24:39Z

claim taken from agent:codex/renki-resume by agent:codex/renki-milestones

**agent:codex/renki-milestones** at 2026-10-30T17:30:03Z

PR #86 landed as 41110b0 after just check plugin test-core LocalRouteTests, just check ci (.checks/20261030T172544Z-89u0RN.log) and committed just check review (.checks/20261030T172624Z-lzd7cQ.log). Installed with checksum verification (.checks/20261030T172737Z-ocaU3K.log); initial install hit a transient controller.exe image lock despite process exit, and a later retry passed. Adding a non-mutating executable-write-access wait to stop-panel.ps1 so full package uploads do not race that lock. The new resupply run .checks/resupply-trial-20261030T172751293533Z is in progress: one native move_to has travelled about 87 m without damage, compared with the prior zero-probe refusal. No final home/food success claimed yet. Adding just vm trial-status to project compact ongoing position, observed travel, health, food and exact final outcomes from retained local evidence, reducing testing context without extra VM requests.
