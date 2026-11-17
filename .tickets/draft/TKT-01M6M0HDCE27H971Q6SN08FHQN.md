---
schema: 4
id: TKT-01M6M0HDCE27H971Q6SN08FHQN
title: Prefer clear building interiors and centered stair approaches
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - movement
  - plugin
  - live
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-04T19:28:25Z
updated_at: 2026-11-04T19:28:25Z
created_by:
  id: agent:codex/explore-night
  name: ""
updated_by:
  id: agent:codex/explore-night
  name: ""
extensions: {}
---

## Description

The owner observed Tas clipping stair corners, double-door edges and posts at floor corners and proposes preferring the middle of open building pieces. Current Walker uses navmesh corners and 0.5m supported local-route samples, with no explicit stair-foot/center-of-opening obligation. During the supervised200m outing, ordinary warehouse return t-4 timed out by the stairs; scoped t-5 climbed via measured [-189.51,31.26,-471.72] -> [-193.01,32.97,-471.11] before a later route refusal. Checked descent t-7 succeeded. Long warehouse targets t-8/t-9 and t-10 demonstrate door approach sensitivity; east-exit t-12 refused a portal trigger and ended near the portal row. Receipts stay in .checks/task-{10y55ddm,ohq95jtx,_bygfryf,3solv4a4,3oj5cdhx,13mcheun,942goio1}.json.

Investigate supported free-space center/clearance preference, doorway entrance/through/exit hints and stair bottom/top connectors. Prefer safe interior routes without snapping to occupied geometric piece centers. Retain floor identity, actual capsule support/clearance, native door/ward/portal guards and no-fall/no-water checks. Assert rotated/narrow stairs, lower-floor voids, open double-door leaves, edge posts, occupied centers, offset doorways and owner takeover. Audit selected hint/clearance costs and actual route/receipt. This is filed follow-up, not an implementation claim.
