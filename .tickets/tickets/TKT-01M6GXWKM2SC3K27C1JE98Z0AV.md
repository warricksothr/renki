---
schema: 4
id: TKT-01M6GXWKM2SC3K27C1JE98Z0AV
title: "Travel: allow checked shallow-water wading without swimming"
type: bug
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - movement
  - plugin
  - protocol
  - live
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/home-survey-delivery
  branch: t3/map-home-location
  worktree: /home/sothr/.t3/worktrees/renki/t3-b02dd361
  commit: de0457496e394e5962c124ed0ddb6553948eece1
  session: null
  claimed_at: 2026-11-03T14:44:43Z
  expires_at: null
archive: null
created_at: 2026-11-03T14:44:23Z
updated_at: 2026-11-03T14:44:44Z
created_by:
  id: agent:codex/home-survey-delivery
  name: ""
updated_by:
  id: agent:codex/home-survey-delivery
  name: ""
extensions: {}
---

## Description

During the owner-observed old-house to new-home return on 2026-11-03, native t-66 made supported southbound progress but stopped at [-47.32,30.94,-117.91] with bounded dry bank departure time_limit. Reference water is 30; conservative dry planning requires another clearance buffer and does not model supported shallow wading. Evidence .checks/task-mbs1nked.json and .checks/new-home-bank.json. HP stayed full and injurySeq stayed 16702.

The owner explicitly requested distinguishing safe walkable shallow water from swimming. Add an explicit checked-wading travel mode. Preserve strict dry survey/household contracts. Use native swimmer-depth configuration with a conservative reserve and current local liquid heights; validate actual support, slopes, continuous short steps, body/obstacle/portal clearance and hostile exclusion. Grounded immersion alone is not swimming. Actual swimming, missing support, invalid depth configuration and excessive depth must stop this mode without enabling swimming/shore fallback. Existing direct stop/newer-order behavior applies.

Validate with core boundary and captured-bank regressions, wire argument agreement, real plugin build and full gate/review. Deploy under the lease, then attempt the changed southbound route with actual health, support, immersion and native receipt evidence. New-home arrival and a free-bed claim remain separate live outcomes; do not invent either.

## Acceptance criteria

- [ ] Explicit wading mode validates identically in Go and C# and preserves strict dry scoped movement.
- [ ] Planning and live execution admit supported shallow steps using current liquid height and native swimming threshold reserves.
- [ ] Missing support, discontinuous footing, excessive depth, swimming, unsafe slopes, obstacles, portals and hostile corridors remain refused.
- [ ] Focused regressions, the full gate and committed review pass.
- [ ] Changed native southbound travel has retained receipts and health/support/water evidence, or an exact remaining refusal is recorded.

## Implementation plan

Add explicit move_to wade:true, requiring supported dry/hostile-avoiding coordinate travel and disallowing scoped familiarization or combat recovery. Advertise a native feature and refuse dispatch to older plugins. Derive a maximum 0.8 m supported wading allowance from the local character swim depth minus the game's 0.4 m swim offset and a further 0.3 m reserve. Account for settling and validate current liquid heights at supported endpoints, midpoints, short prefixes and actual body footing. Wading bypasses strict dry bank/shore recovery only while grounded, not swimming and inside its measured limit; it never dispatches swimming input. Apply the same mode to local/navmesh/reused-route probes and reset it on every goal. Preserve default dry semantics and survey admission. Add captured-bank, depth/wave, support and protocol regressions, then build/gate/commit/review/deploy and supervise one changed native return. Record partial travel and any unavailable bed separately.
