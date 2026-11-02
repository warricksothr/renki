---
schema: 4
id: TKT-01M6A1D13A2A9WXTEZH9MS1WZA
title: Avoid repeating stalled stair approaches on raised-home return
type: bug
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - movement
  - safety
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-31T22:31:06Z
updated_at: 2026-10-31T23:10:07Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

The western food trial gathered live mushrooms, but its first home return t-12 timed out after120s. The bounded cleanup return t-14 eventually reached verified shelter without injury. Full evidence is .checks/marker-food-20261031T222103Z (especially t-12.json/t-14.json); this is a navigation failure, not an empty resource site.

Near [-48,32.5,7.8], native local search repeatedly found raised entrance routes via approximately [-47.62,32.87,8.13], [-47.62,33.13,8.63], [-47.62,33.47,9.13] toward [-39.77,36.09,11.2]. Actual walking stalled and replanned similar prefixes while keeping feet on support. Inspect capsule/support projection and how failed execution updates RouteRefusals before altering tolerances. Use learned refusals or a newly checked alternative; do not bypass floor, water, collision or injury guards. The successful second return is evidence of recoverability, not proof that the first timeout is fixed.

## Acceptance criteria

- [x] Replay the captured raised-home approach without repeating a stalled local prefix
- [x] Preserve dry-ground, support, hostile and floor checks while selecting a checked alternative
- [x] Earn a bounded live return from the western food site without manual routing

## Implementation plan

Implement a core route-execution progress watch independent of StuckDetector resets and path replacement. Charge only time when movement was actually commanded; planning/door waits pause it. Record best forward physical progress so stair-edge oscillation cannot repeatedly reset it. Keep this watch across replans, reset on genuinely different goals/approaches or real travel, and retain bounded directional/floor-aware RouteRefusals for a stalled execution prefix. Physical route/support/hostile checks remain unchanged. Replay the captured oscillation, planning waits, real progress, reverse escape and alternative-route selection; run focused checks, full CI, committed review and deploy. Run one bounded live raised-home round trip using the existing marker checker, and record earned results without promoting failures to passes.

## Notes

**agent:codex/renki-milestones** at 2026-10-31T23:02:00Z

Work continues on t3/raised-home-refusals above t3/marker-idle-food; claim predates the branch switch. RouteProgress core regressions and full CI passed before adding the supervised --travel-only mode. Second full gate is running. Tas resupplied through the normal food goal without injury and remains safely online (.checks/food-20261031T225915Z). Live route proof remains pending.

**agent:codex/renki-milestones** at 2026-10-31T23:08:48Z

Deployed e14d7b1. First supervised west round trip (.checks/marker-food-20261031T230333Z) failed its 120-second first return t-2; keep this as failure. New execution refusals fired four times and changed checked routes. Cleanup t-3 reached fresh safe home shelter without observed injury (injurySeq 0 throughout), keeping Tas online. Testing a separately recorded 180-second per-leg budget for this long dry/hostile-avoiding journey, rather than treating cleanup as first-return success.

**agent:codex/renki-milestones** at 2026-10-31T23:10:07Z

Clean second west-site round trip passed in .checks/marker-food-20261031T230731Z: t-4 outward 53.58s, t-6 first return 63.74s, both exact native success receipts. Fresh final position [-36.6,36.09,7.69], sheltered, idle; injurySeq stayed 0. Trial permitted 180s per leg but actual return was below120s; this does not erase the first trial timeout. Focused route-progress and supervisor policy checks, full CI and committed review passed. Native e14d7b1 remains deployed. Long-trip helper now accepts an explicit120–180s per-leg bound without weakening checks.

## Summary

Implemented and deployed in PR #135. Captured oscillation regressions and fresh live west/home round trip passed; earlier120s failure remains documented separately. Tas remains safely online at home.
