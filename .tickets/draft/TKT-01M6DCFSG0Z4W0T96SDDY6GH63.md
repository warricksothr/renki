---
schema: 4
id: TKT-01M6DCFSG0Z4W0T96SDDY6GH63
title: "Goals: lifecycle so stuck records never block new goals"
type: bug
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - controller
assignees: []
milestone: M3
parent: TKT-01M6DCF2ABDDNG3QF369NDPBT5
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T05:42:34Z
updated_at: 2026-11-02T05:42:34Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-review
  name: ""
extensions: {}
---

## Description

Durable goal records have no full lifecycle, so the store can end up refusing every new goal. That would stop household, food and combat goals until someone edits goals.json by hand.

### 1. Records that never qualify for eviction

At `MaxRecords` (64), `internal/goal/store.go` `Add` only evicts records that are terminal, have no `Pending`, and ended more than 15 minutes ago. Three kinds of record never get there:

- `suspended` records past their deadline. `Queue` refuses to resume them, so they never reach a terminal status.
- Any record with `Pending`, including `cancelled` ones, because `Cancel` does not clear `Pending`.
- `ready` records left by a failed panel start. `internal/app/goals.go:59-70` calls `Goals.Add` before `goalAction`'s freshness and busy checks, so pressing Start while the bot is busy leaves a `ready` record that `End` skips and `terminal()` excludes.

`FoodWork` creates a record for every household episode, so disconnects alone can fill the store.

### 2. The size cap is never rotated against

`store.go` `persist` enforces 4 MB on every write, but rotation is triggered only by the record count. Near 4 MB, even `Cancel` and `End` checkpoints fail.

### 3. An owner stop can be overwritten mid-step

`runtime.go` `Next` reads the record, evaluates the policy, then calls `update` with closures that never re-check `Status` (dispatch around :356-372, success around :291, `need` around :197, `food.go:544`). A panel stop that lands mid-step can have `Pending`/`Attempts` added to the cancelled record, or be turned into `succeeded` or `needs_plan`. That breaks the docs/GOALS.md promise that cancellation is never reset.

### Fix

- Run the freshness and busy checks before `Add`, or cancel the record when `goalAction` fails.
- On `Open` and `Queue`, move expired `suspended` and `ready` records to `needs_plan`, then let them age out.
- Retain an uncertain intent in a capped side list instead of keeping its record forever.
- Rotate on size as well as count. Consider storing each policy source once, keyed by its hash.
- Make every `Next`/`foodNext` update closure fail unless `Status == "running"`.
- Update the docs/GOALS.md sentence about the limit refusing new goals, because the code does rotate.

## Acceptance criteria

- [ ] A failed panel start leaves no ready record behind
- [ ] Expired suspended/ready records and cancelled records with a pending intent become evictable
- [ ] Rotation also bounds the file size, and status-only transitions cannot fail on the size cap
- [ ] Runtime update closures refuse to change a record that is no longer running
- [ ] docs/GOALS.md describes the actual retention rule
