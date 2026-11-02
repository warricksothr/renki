---
schema: 4
id: TKT-01M6DDBX5TPR0VD0RQ2CT7NY3M
title: "Idle routines: stale bedside arrival, no-op runs exhaust MaxRuns"
type: bug
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
assignees: []
milestone: null
parent: TKT-01M6DCF2ABDDNG3QF369NDPBT5
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T05:57:55Z
updated_at: 2026-11-02T05:57:55Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-review
  name: ""
extensions: {}
---

## Description

Two defects in the idle routines added by #152/#153: bedside rest and household food work.

### 1. The bedside wait can judge arrival on a stale state

`internal/behaviour/marker_rest.go:144-152`: after the move's success receipt, the run waits only for a state newer than `r.beforeSeq`, the frame from when the order was sent. It never compares against `result.ReceiptSeq`. Commit 75db0ae added exactly this check to food (`internal/goal/food.go:210`).

The engine calls `Next` with the view from the start of the tick, which is often the last state before the native arrival, about 2.1–2.5 m from the bed. `besideBedOnFloor` then fails with "bedside move lacks observed arrival on the bed floor", which counts a run and starts the 300 s cooldown.

Fix: wait while `v.State.Seq <= max(r.beforeSeq, result.ReceiptSeq)`, and add a test with a state from before the receipt.

### 2. Runs that do nothing use up MaxRuns

`internal/goal/food_work.go:69-71` and `marker_rest.go:110-112,138-140` return `nil, nil` when nothing is needed: already fed with a reserve, or already beside the bed. `engine.go` treats that as a success, `finish` increments `i.completed`, and `MaxRuns: 100` is checked against that count, which resets only on restart.

Personal resupply is exhausted after about 8.75 h of idling and day-mode bedside rest after about 11.7 h. After that the food gap these routines close comes back silently.

Fix: give the engine a "nothing to do" outcome that does not count as a run (or admit only when work is needed), with an engine test. Also update the "100 attempts per controller session" wording in the docs.

## Acceptance criteria

- [ ] Bedside rest waits for a state newer than the move's receipt, covered by a test
- [ ] A run that sends no order does not count toward MaxRuns, covered by an engine test
- [ ] The docs describe what counts as an attempt
