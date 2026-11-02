---
schema: 4
id: TKT-01M6EFQNYKRHT3AP5CEJNVVSFX
title: Return to the expedition anchor within a reserved budget
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
parent: TKT-01M66PMAQSBK2TH076X7M5FGAM
origin: null
dependencies:
  - TKT-01M6EFQNWT6P9JJA3KVHFSQA2F
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

Rollout step 3, return. A controller step that decides when to turn back and proves arrival. Turn back while the remaining time and travel still cover the measured return (from TKT-01M6EFQN91 (Report and keep the ground the bot actually walked)'s trail lengths) plus a margin. Dispatch move_to to the pinned anchor. Arrival proof matches the combat goal's return in internal/goal/return.go: a succeeded receipt, then a state observed after the receipt within 3 m horizontally and 1.75 m vertically of the anchor (native `Arrival.Level` in plugin/Renki.Core/Movement/Arrival.cs).

If the return fails, try one alternative: the visited trail reversed, leaving out segments that descended more than a step, because an outbound drop does not prove reverse travel. Then stop with needs_plan and a diagnostic. If return becomes unsafe, native retreat wins and the goal keeps a partial diagnostic.

### Why it matters to M7
Goal 7.3 counts a run only with verified arrival at the anchor inside the elapsed budget.

### Files
- internal/goal/return.go (pattern), a new explore return file in internal/goal/
- plugin/Renki.Core/Movement/Arrival.cs (thresholds to match)

### Verification
- `just check controller test-packages ./internal/goal`
- `just check ci`

## Acceptance criteria

- [ ] The turn-back decision is a pure function, tested at budget boundaries, that fires while the remaining budget still covers measured return travel plus the margin
- [ ] Arrival proof requires a succeeded receipt and a later state within 3 m horizontally and 1.75 m vertically of the pinned anchor (test with a stale state rejected)
- [ ] A failed return tries at most one alternative (reversed trail without descending segments), then ends needs_plan with the reason; no further retry (test)
- [ ] An owner stop during return cancels without another dispatch (test)
