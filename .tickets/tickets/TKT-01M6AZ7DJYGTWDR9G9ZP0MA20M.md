---
schema: 4
id: TKT-01M6AZ7DJYGTWDR9G9ZP0MA20M
title: Traverse known portals with verified arrivals
type: task
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - movement
  - controller
assignees: []
milestone: null
parent: null
origin: null
dependencies:
  - TKT-01M6AYP8TQRJCX0ZAQEY0GBDCV
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/portal-deploy
  branch: t3/renki-portal-travel-startup
  worktree: /home/sothr/.t3/worktrees/renki/t3-6c0da0c0
  commit: 501ca8d3ca755d7de1888f92f3a1e27e92bd6d7e
  session: null
  claimed_at: 2026-11-04T16:21:07Z
  expires_at: null
archive: null
created_at: 2026-11-01T07:12:19Z
updated_at: 2026-11-04T16:21:45Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/portal-deploy
  name: ""
extensions: {}
---

## Description

Build on observed per-world portal links as explicit route transitions. Revalidate the exact link and native inventory/server restrictions, use normal approach/entry controls, verify teleport completion and actual arrival, clear local geometry after transport and expose changed-link, forbidden-cargo, unknown-destination, cancellation and arrival-timeout outcomes. Track observed exits and return-route knowledge without treating remote portal coordinates as safe landing proof. Preserve cargo such as owner-reserved ore. Deliberate entry needs a narrowly scoped exception to the final native trigger guard, never a global bypass.

## Acceptance criteria

- [ ] Portal travel is an explicit scoped transition distinct from walking and tag editing
- [ ] Link changes, native item/server restrictions and cancellation cannot cause unintended travel
- [ ] Arrival location and return knowledge are observed and persisted separately from source link metadata
- [ ] Repeatable native tests verify traversal and restoration without spawned supplies

## Implementation plan

Add an explicitly ordered interact portalTo transition that pins native source/destination IDs, stays separate from tag editing and ordinary walking, and advertises a feature checked before dispatch. Reuse the exact-link walking/trigger passage with an ordered mode: bounded hostile-avoiding approach, live link/cargo/access revalidation, observed native teleport, then grounded three-dimensional proximity to the loaded expected destination before success. Record the actual arrival and separately observed return link in the task/audit receipt; do not infer remote safety or general route chains. Cover protocol rejection, cancellation, changed links, native refusal and false-arrival cases with focused checks, then the complete gate and committed review before deployment. Cross the owner-selected SpawnCamp link only after fresh observations. Keep native repeatability and durable map-level route knowledge unchecked until earned.

## Notes

**agent:codex/home-survey-delivery** at 2026-11-03T15:51:12Z

Concrete return case from the new-home continuation: automatic timer Loot t-38 pursued Resin, remained active after repeated portal_cast input refusals, escalated stuck Jump/Strafe and entered darkforest at2026-11-03T15:36:32Z. Source near[-188.09,30.97,-459.47] is1:161081; remote portal1:161078 at[415.39,31.65,-1176.34] reports connected target1:161081 with its known destination position. Tas is currently on the remote portal platform, saved home unchanged near[-204.39,32.97,-467.43]. Ordinary deliberate movement into the return trigger is correctly refused; no traversal allowance was granted. Audit .checks/audit-window-FfbfFj.jsonl, blocked probe .checks/task-o4knlg4c.json, fresh link .checks/new-home-before-portal-deploy-map.json.

TKT-01M6H1A2XPHCAR4F016AE4F8K0 (End the body task when the final portal guard refuses movement) closes the separate continuation hole: final guard failure terminates the body task before later recovery ticks. It does not implement this ticket's controlled traversal. Existing Loot has been disabled through the normal API, preserving its definition. The owner supplied Raspberry and Tas recovered from the earlier storage injury. Keep this ticket draft; known destinations still are not safe landing/return receipts, and intentional portal travel needs its own admitted transition rather than an exception to all ordinary movement.

**agent:codex/home-survey-delivery** at 2026-11-03T16:00:45Z

Supersedes the healthy remote-platform snapshot in the preceding note. On deployed cc5f323, Greydwarf 1629975892:102 hit at 15:54:52.701Z (11.99 damage), 15:54:54.821Z (12.28) and 15:55:04.879Z (11.44). Automatic flee t-2 started within 0.28s of the first hit, failed unreachable at 15:54:55.186Z; replacement flee t-5 remained active until death at 15:55:04.915Z near [413.36,31.65,-1177.66]. No direct probe interrupted these retreats. Respawn at 15:55:22.962Z was at the old base [-31.08,36.55,15.19], followed by automatic strict-dry return t-7, which failed unreachable at 15:55:44.553Z. Latest resumed observation at 15:56:28Z: idle 25/25HP near [-46.32,33.79,10.13], no active/carried food, empty pack and no equipped gear; fresh local perception had no characters. Loot remains disabled. Exact retained events/states/inventory: .checks/audit-window-M3BtxI.jsonl; fresh state/map .checks/new-home-resume-runtime.json and .checks/new-home-resume-map.json. This is a death/failed escape, not a safe portal arrival or return, and does not earn survey, bed or survival acceptance.
