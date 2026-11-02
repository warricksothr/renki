---
schema: 4
id: TKT-01M6EFQNT4D9QAMN9E0S7EECRD
title: Check expedition readiness and report specific shortfalls
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
assignees: []
milestone: M7
parent: TKT-01M66PMAQSBK2TH076X7M5FGAM
origin: null
dependencies:
  - TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
  - TKT-01M62N2557R401S3E6JECSAPN4
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:58:32Z
updated_at: 2026-11-02T15:58:44Z
created_by:
  id: agent:claude-code/renki-plan-m7
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m7
  name: ""
extensions: {}
---

## Description

Rollout step 3, preparation. A controller function decides whether Tas may start, or continue, an expedition from the current snapshot, and names each shortfall. It pins the anchor waypoint's position and its `At` time (internal/home/waypoints.go) at start, so a later waypoint edit cannot silently redirect an active expedition.

The first benchmark uses supplied gear and food. This ticket does not craft, repair or resupply; it only reports what is missing.

### Why it matters to M7
Goal 7.3 starts each run from an observed safe anchor with supplies. An honest refusal here is how a run fails safely instead of walking out hungry.

### Files
- internal/goal/ (new explore readiness file beside internal/goal/food_stock.go, which already reads meals and inventory)
- internal/home/waypoints.go, internal/protocol/inventory.go, internal/behaviour/health.go

### Verification
- `just check controller test-packages ./internal/goal`
- `just check ci`

## Acceptance criteria

- [ ] Ready only when in the named world, within 3 m horizontally and 1.75 m vertically of the pinned anchor, HP and stamina at or above the spec thresholds, the spec's meal count active or carried, and free inventory slots at or above the spec
- [ ] Each failed check yields its own shortfall code and message (for example not_at_anchor, low_health, missing_food, no_space, anchor_changed), one test each
- [ ] A waypoint edit after start makes the check return anchor_changed instead of following the new position (test)
- [ ] No crafting, repair, resupply or combat setting change is attempted
