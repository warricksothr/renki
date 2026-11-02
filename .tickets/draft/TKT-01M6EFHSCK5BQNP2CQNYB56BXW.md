---
schema: 4
id: TKT-01M6EFHSCK5BQNP2CQNYB56BXW
title: "Goals: evict expired and cancelled records, rotate on size"
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

Part 2 of TKT-01M6DCFSG0 (Goals: lifecycle so stuck records never block new goals), for goal 3.5 in docs/ROADMAP.md.

At `MaxRecords` (64), `Add` in `internal/goal/store.go` evicts only records that are terminal, have no `Pending`, and ended more than 15 minutes ago. Three kinds of record never get there: `suspended` records past their deadline (`Queue` refuses to resume them), any record with `Pending` including cancelled ones (`Cancel` does not clear `Pending`), and stale `ready` records. Separately, `persist` enforces 4 MB on every write but rotation only looks at the record count, so near 4 MB even `Cancel` and `End` checkpoints fail. `FoodWork` creates a record per household episode, so disconnects alone can fill the store.

### Outcome

- On `Open` and `Queue`, expired `suspended` and `ready` records move to `needs_plan` with a reason, then age out like other terminal records.
- A cancelled record's uncertain `Pending` intent moves to a capped side list (keep the newest N with their goal IDs), so the record itself becomes evictable.
- Rotation evicts by size as well as count, and a status-only transition (cancel, end) never fails on the size cap. Consider storing each policy source once, keyed by its hash.
- docs/GOALS.md states the actual retention rule (it currently says the limit refuses new goals, but the code rotates).

### Files

- `internal/goal/store.go` (`Add`, `persist`, `Open`, `Queue`, `Cancel`)
- `internal/goal/runtime_test.go` or a new `internal/goal/store_test.go`
- `docs/GOALS.md`

### Verification

- `just check controller test-packages ./internal/goal ./internal/app`, with tests for: 64 suspended-expired records then a new `Add`, cancelled with pending, a store near 4 MB accepting `Cancel` and `End`, and restart keeping the side list.
- `just check ci`

## Acceptance criteria

- [ ] Expired suspended and ready records move to needs_plan on Open and Queue and later age out
- [ ] Cancelled records with a pending intent become evictable; the intent survives in a capped side list across restart
- [ ] Rotation bounds the file size, and cancel and end never fail on the size cap
- [ ] docs/GOALS.md describes the actual retention rule
- [ ] just check ci passes
