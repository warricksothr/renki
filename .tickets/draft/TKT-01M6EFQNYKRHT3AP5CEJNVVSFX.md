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
  - TKT-01M6EFQN91YGXHVP9NF32BK21W
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:58:32Z
updated_at: 2026-11-02T17:28:48Z
created_by:
  id: agent:claude-code/renki-plan-m7
  name: ""
updated_by:
  id: agent:claude-code/renki-plan
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

## Notes

**agent:claude-code/renki-plan** at 2026-11-02T17:28:48Z

Applied on 2026-11-02 by the planning pass: this ticket now depends on TKT-01M6EFQN91 (Report and keep the ground the bot actually walked) instead of TKT-01M6EFQNWT (native survey task). The return sends move_to to the pinned anchor and, as its one alternative, follows the walked trail reversed. Both come from N91, and neither needs the survey task, so the return can be built offline.
