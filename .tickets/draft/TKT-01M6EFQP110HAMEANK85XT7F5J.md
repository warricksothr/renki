---
schema: 4
id: TKT-01M6EFQP110HAMEANK85XT7F5J
title: "Explore goal kind: spec, validation and durable budgets"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
assignees: []
milestone: M7
parent: TKT-01M66PMASYEM3Z9KJMB2979CEM
origin: null
dependencies:
  - TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
  - TKT-01M62N2557R401S3E6JECSAPN4
  - TKT-01M6DCFSG0Z4W0T96SDDY6GH63
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

Rollout step 4, first half. Add a goal kind `explore`. Today internal/goal/store.go `validateRecord` accepts only `combat_trial` and `food_resupply`, and internal/goal/runtime.go `Spec`/`NewRecord` build only those.

The spec carries world, anchor (waypoint name, pinned position and `At`), radius (60 to 250 m, the existing record bounds), excluded zones, elapsed and work budgets, travel budget, attempt and distinct-failure limits, required new m2, and health/stamina/meal return thresholds. The record persists the original spec, the start instant used as the coverage `since`, the source id, budgets used and left, chosen and failed frontiers, legs and return state, within `MaxJSON` (16 KB).

Depends on TKT-01M6DCFSG0Z4W0T96SDDY6GH63 (Goals: lifecycle so stuck records never block new goals) because both change record lifecycle in the same file.

### Files
- internal/goal/store.go, internal/goal/runtime.go
- internal/app/goals.go and internal/app/goals_test.go (POST /api/goals)
- internal/inspection/inspection.go (implemented adapter list)
- docs/GOALS.md

### Verification
- `just check controller test-packages ./internal/goal ./internal/app ./internal/inspection`
- `just check ci`

## Acceptance criteria

- [ ] POST /api/goals accepts a valid explore spec and rejects each invalid field (radius outside 60-250 m, non-finite anchor, a budget at or below zero or above its cap, required m2 above the circle's area) with a reason; one test each
- [ ] Reopening the store restores the spec, start instant and budgets unchanged, and resume never resets a budget (test)
- [ ] A record with 32 failed frontiers and 16 legs stays under MaxJSON (test)
- [ ] internal/inspection reports explore as an implemented adapter and docs/GOALS.md documents the spec
