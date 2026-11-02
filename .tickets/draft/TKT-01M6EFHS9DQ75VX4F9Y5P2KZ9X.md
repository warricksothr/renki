---
schema: 4
id: TKT-01M6EFHS9DQ75VX4F9Y5P2KZ9X
title: "Goals: a failed panel start leaves no ready record"
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

Part 1 of TKT-01M6DCFSG0 (Goals: lifecycle so stuck records never block new goals), for goal 3.5 in docs/ROADMAP.md.

The panel's goal start (`internal/app/goals.go`, the `start` branch around lines 59-70) calls `a.Goals.Add(record)` before `goalAction` runs its freshness and busy checks. Pressing Start while the bot is busy returns an error to the page but leaves a `ready` record behind. `End` skips it and `terminal()` excludes it, so it never ages out, and enough of them fill `MaxRecords` (64) in `internal/goal/store.go`.

### Outcome

A failed panel start leaves no `ready` record. Either run the checks before `Add`, or cancel (or remove) the record when `goalAction` fails. A successful start is unchanged.

### Files

- `internal/app/goals.go` (`start` branch and `goalAction`)
- `internal/goal/store.go` if a remove or cancel helper is needed
- Tests beside the existing goal API tests in `internal/app`

### Verification

- `just check controller test-packages ./internal/app ./internal/goal`
- `just check ci`

## Acceptance criteria

- [ ] A panel start refused as stale or busy leaves no ready record; a test covers both
- [ ] A successful start is unchanged; just check ci passes
