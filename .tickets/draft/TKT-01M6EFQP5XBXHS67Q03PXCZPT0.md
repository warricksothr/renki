---
schema: 4
id: TKT-01M6EFQP5XBXHS67Q03PXCZPT0
title: Replay scenarios for exploration goals
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
  - TKT-01M6EFQP437YT124TQ25299BJV
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:58:33Z
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

docs/EXPLORATION.md ("Rollout and evidence") lists the scenarios an exploration goal must survive: repeated or stale frames, shared-map-only knowledge, incomplete scans, stacked floors, water and cliffs, unavailable supplies, declining health, blocked or changed return routes, no useful frontier, budget exhaustion, checkpoint failure, uncertain restart, world change, owner stop and owner takeover. Write one named replay test for each against the explore runner, using the fake-view style of the existing food goal tests (for example internal/goal/food_test.go).

Each must end in a materially different allowed choice, a partial result or needs_plan, never an identical retry, and budgets must survive restart.

### Why it matters to M7
This is goal 7.5's evidence.

### Files
- internal/goal/ (new explore replay test file), internal/explore/testdata/

### Verification
- `just check controller test-packages ./internal/goal`
- `just check ci`

## Acceptance criteria

- [ ] The 15 scenarios above each have one named test, listed in the ticket summary
- [ ] Each test asserts the outcome is a different allowed choice, partial or needs_plan, and that no identical dispatch repeats
- [ ] Restart and checkpoint-failure tests show the uncertain step is re-observed, not replayed or counted as discovery, and budgets are not reset
