---
schema: 4
id: TKT-01M6EFHSFVSFG7ETXW2J8HXRWB
title: "Goals: runtime updates refuse records that are no longer running"
type: bug
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - controller
assignees: []
milestone: M3
parent: TKT-01M6DCFSG0Z4W0T96SDDY6GH63
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:55:19Z
updated_at: 2026-11-02T15:55:19Z
created_by:
  id: agent:claude-code/renki-plan-m3
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m3
  name: ""
extensions: {}
---

## Description

Part 3 of TKT-01M6DCFSG0 (Goals: lifecycle so stuck records never block new goals), for goal 3.5 in docs/ROADMAP.md.

`Next` in `internal/goal/runtime.go` reads the record, evaluates the policy, then calls `Store.update` with closures that never re-check `Status`: the `need` path (around line 198), success (around 294) and dispatch (around 350), and in `internal/goal/food.go` the food updates (for example around line 544). An owner's stop from the panel that lands mid-step can get `Pending` or `Attempts` added to the cancelled record, or be turned into `succeeded` or `needs_plan`. docs/GOALS.md promises cancellation is never reset.

### Outcome

Every update closure called from `Next` and the food runner refuses (returns an error the runner treats as "stopped, do nothing") unless the record's `Status` is still `running`. A cancel during any step stays cancelled, and the order computed in that step is not dispatched.

### Files

- `internal/goal/runtime.go`, `internal/goal/food.go`, `internal/goal/store.go` (`update`)
- `internal/goal/runtime_test.go`, `internal/goal/food_test.go`

### Verification

- A test that cancels between the read and the update for each path (need, success, dispatch, food) and asserts the record stays `cancelled` with no new `Pending` and no dispatch.
- `just check controller test-packages ./internal/goal ./internal/app`, and the race suite through `just check ci`.

## Acceptance criteria

- [ ] Every Next and food update closure refuses unless Status is running
- [ ] Tests cancel mid-step on the need, success, dispatch and food paths and show the record stays cancelled with no dispatch
- [ ] just check ci passes
