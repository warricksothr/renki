---
schema: 4
id: TKT-01M6A62GWANGQWSB1GYCZ66G37
title: Validate outdoor household food delivery and fallback
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - vm
assignees: []
milestone: M3
parent: TKT-01M6864352N8E28YCGHA19AN90
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-milestones
  branch: t3/outdoor-household-validation
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 50c3d6433ce5f06ac753ddab72f77137aa3fd5f8
  session: null
  claimed_at: 2026-10-31T23:52:51Z
  expires_at: null
archive: null
created_at: 2026-10-31T23:52:44Z
updated_at: 2026-10-31T23:57:21Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

Exercise the authorized outdoor household-food sequence with real collection and storage receipts, then depleted-source fallback and return pressures. Fix gaps exposed by bounded trials without relaxing safety or treating existing chest stock as new production.

## Acceptance criteria

- [ ] An outdoor household trip has verified collection and stock deposit receipts, or its precise failure and follow-up fix are recorded.
- [ ] Depleted sources select a distinct eligible source without looping, covered by deterministic checks and live evidence where safe.
- [ ] Capacity and declining nourishment commit a safe return before further remote collection, with meaningful regression coverage.
- [ ] Required checks pass and live outcomes and remaining repair work are recorded.

## Implementation plan

Run a bounded native household goal against current home storage with required outdoor collection and deposit receipts; retain exact failure evidence. Exercise distinct-source exhaustion and pressure decisions with regression scenarios, fixing the shared goal ordering or supervisor only when evidence requires it. Preserve current safety, ownership and explicit source/storage scope. Record live outcomes, run just check ci and committed review, then publish a stacked PR and deploy verified fixes before inspecting repair support.

## Notes

**agent:codex/renki-milestones** at 2026-10-31T23:57:21Z

The initial live trial is retained at `.checks/food-20261031T235251Z` (goal `78533cfb4e46dc9c1b6e2af6`, controller `50c3d64`). Tas normally withdrew/ate one grilled tail and two yellow mushrooms, healed to full fed health, but retained only one personal meal against reserve two. Both distinct configured source routes were attempted once and refused as unreachable, not proven depleted: t-6 berries, t-7 west mushrooms. t-8 returned to the correct home floor without injury; the supervisor reported needs_plan and shut down the game, leaving the panel online. There were zero collection/deposit receipts. This is failed production evidence, not an outdoor or depleted-site pass.

Code inspection found that stock pressure selection was only reachable while personal reserve/nourishment/recovery already passed. The fix checks return pressures after pending product reconciliation and before personal resupply/recovery, once stock work has been admitted through a stock observation; initial personal acquisition remains possible. A committed return persists if pressures improve en route. Regression scenarios cover low reserve, remaining nourishment below 300 seconds, recovery, weight, slots, continuing delivery, bootstrap acquisition, pending collection accounting and distinct depleted-site fallback with home return. Compact trial status now exposes stock decisions, attempts, collection and deposit evidence without full checkpoints.

Native station repair already exists and uses the ordinary game repair button with verified per-item durability gains. Multi-gather also returns to repair its axe and requires fresh restored durability before resuming. The next maintenance slice should extend/verify household tool and armour repair rather than add another primitive or silently enable unrestricted crafting/upgrades.
