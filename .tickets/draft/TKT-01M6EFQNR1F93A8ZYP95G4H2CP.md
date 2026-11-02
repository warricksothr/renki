---
schema: 4
id: TKT-01M6EFQNR1F93A8ZYP95G4H2CP
title: Remember failed frontier approaches with bounded revalidation
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
parent: TKT-01M66PMAPGXNJHB49E7WMGEBYS
origin: null
dependencies:
  - TKT-01M6EFQNMW1J01MX78QAB91CZV
  - TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
  - TKT-01M62N2557R401S3E6JECSAPN4
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:58:32Z
updated_at: 2026-11-02T17:27:14Z
created_by:
  id: agent:claude-code/renki-plan-m7
  name: ""
updated_by:
  id: agent:claude-code/renki-plan
  name: ""
extensions: {}
---

## Description

Rollout step 2, second half. When a leg toward a frontier fails, record the frontier point, approach bearing, native failure reason, time and an evidence revision (the atlas first-received or hash state of the 16 m tiles around it). The ranker skips the identical approach until that evidence changes or a cooldown passes. One failed approach never marks a region unreachable: another bearing stays eligible.

Keep it bounded per goal and per world. The goal record's `Searches` ledger in internal/goal/store.go is the precedent: the host keeps it and policy state cannot reset it.

### Why it matters to M7
Goal 7.2 requires zero identical retries without new evidence, and goal 7.5's replay matrix depends on failures leading to a different choice.

### Files
- internal/explore/ (from TKT-01M6EFQNMW (Rank exploration frontiers from measured coverage))
- internal/goal/store.go (`SearchAttempt` precedent), internal/goal/areas.go

### Verification
- `just check controller test-packages ./internal/explore ./internal/goal`
- `just check ci`

## Acceptance criteria

- [ ] A failed approach is stored with point, bearing, reason, time and evidence revision; the ranker skips that identical approach until evidence in the surrounding 16 m tiles changes or the cooldown passes (test)
- [ ] Another bearing to the same frontier stays eligible after one failure; after 3 distinct failures the frontier is skipped for the rest of the goal (test)
- [ ] Repeated observation ticks with unchanged evidence do not re-enable a failed approach (test)
- [ ] Memory is bounded per goal and per world and a test shows the oldest entry evicted first
- [ ] Replay scenes cover no safe frontier, shared-map-only knowledge, stale coverage, an alternative approach and bounded memory

## Notes

**agent:claude-code/renki-plan** at 2026-11-02T17:27:14Z

Applied on 2026-11-02 by the planning pass: removed the dependency on TKT-01M6EFQNBJ (minimap explored cells), which is live. This ticket needs map knowledge only for its "shared-map-only knowledge" replay scene, so build that scene from a recorded fixture in the shape TKT-01M6EFQNBJ will report. M7's offline frontier and goal-runner chain then no longer waits on a VM slot.
