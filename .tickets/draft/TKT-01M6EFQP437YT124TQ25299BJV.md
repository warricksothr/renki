---
schema: 4
id: TKT-01M6EFQP437YT124TQ25299BJV
title: "Explore goal runner: legs, discovery and return proof"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - movement
assignees: []
milestone: M7
parent: TKT-01M66PMASYEM3Z9KJMB2979CEM
origin: null
dependencies:
  - TKT-01M6EFQP110HAMEANK85XT7F5J
  - TKT-01M6EFQNT4D9QAMN9E0S7EECRD
  - TKT-01M6EFQNYKRHT3AP5CEJNVVSFX
  - TKT-01M6EFQNR1F93A8ZYP95G4H2CP
  - TKT-01M6EFQN74X0EDYJAVBJJK2DY8
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

Rollout step 4, second half. The explore runner composes the implemented steps: prepare (TKT-01M6EFQNT4 (Check expedition readiness and report specific shortfalls)), choose a frontier (TKT-01M6EFQNMW (Rank exploration frontiers from measured coverage), TKT-01M6EFQNR1 (Remember failed frontier approaches with bounded revalidation)), one survey leg (TKT-01M6EFQNWT (Native survey task: walk one bounded leg and report it)), reassess, repeat while budgets allow, then return (TKT-01M6EFQNYK (Return to the expedition anchor within a reserved budget)) and report. A compact Starlark policy chooses among those runners and bounded waits, as internal/goal/combat.star does for combat; the host owns success, scope and budgets.

Success needs both: measured coverage since the goal's start inside the circle at or above the spec's m2 (TKT-01M6EFQN74 (Atlas: first-seen time and a measured-coverage query) query), and arrival proof at the pinned anchor (TKT-01M6EFQNYK). Returning with less is `partial` with the measured m2. No safe frontier is `partial` or `needs_plan`, not success. Missing or stale telemetry blocks another outbound leg.

### Why it matters to M7
This is the goal that goal 7.3's trials run.

### Files
- internal/goal/runtime.go, internal/goal/policy.go, internal/goal/combat.star (pattern), a new explore policy file
- internal/atlas (coverage query), internal/explore

### Verification
- `just check controller test-packages ./internal/goal ./internal/explore ./internal/atlas`
- `just check ci`

## Acceptance criteria

- [ ] Success requires measured coverage since start at or above the spec's m2 and arrival proof; tests show movement success alone, map knowledge alone, or a re-confirmed stale area cannot succeed
- [ ] Returning short of the m2 ends `partial` with the measured figure; no safe frontier ends without an outbound leg within one decision (tests)
- [ ] The host enforces elapsed, work, travel (sum of measured trail lengths, failed approaches included), attempt and failure budgets across all legs, and policy state cannot change them (test)
- [ ] A state snapshot older than the host's freshness limit blocks another outbound leg (test)
- [ ] An owner stop cancels the goal and its pending retry; another body order cancels the continuation (tests)
