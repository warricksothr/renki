---
schema: 4
id: TKT-01M6A62GWANGQWSB1GYCZ66G37
title: Validate outdoor household food delivery and fallback
type: task
status: blocked
status_reason: Outdoor delivery and home repair trials await safe recovery around the unchanged hostile corridor; both normal and guarded returns exhausted the native bounded search without injury. See the linked navigation follow-up and retained audits.
priority: high
due_on: null
labels:
  - controller
  - vm
assignees: []
milestone: M3
parent: TKT-01M6864352N8E28YCGHA19AN90
origin: null
dependencies:
  - TKT-01M6A6Y1XQGNZV4B01QK23J26E
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-31T23:52:44Z
updated_at: 2026-11-01T00:10:08Z
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
- [x] Depleted sources select a distinct eligible source without looping, covered by deterministic checks and live evidence where safe.
- [x] Capacity and declining nourishment commit a safe return before further remote collection, with meaningful regression coverage.
- [ ] Required checks pass and live outcomes and remaining repair work are recorded.

## Implementation plan

Run a bounded native household goal against current home storage with required outdoor collection and deposit receipts; retain exact failure evidence. Exercise distinct-source exhaustion and pressure decisions with regression scenarios, fixing the shared goal ordering or supervisor only when evidence requires it. Preserve current safety, ownership and explicit source/storage scope. Record live outcomes, run just check ci and committed review, then publish a stacked PR and deploy verified fixes before inspecting repair support.

## Notes

**agent:codex/renki-milestones** at 2026-10-31T23:57:21Z

The initial live trial is retained at `.checks/food-20261031T235251Z` (goal `78533cfb4e46dc9c1b6e2af6`, controller `50c3d64`). Tas normally withdrew/ate one grilled tail and two yellow mushrooms, healed to full fed health, but retained only one personal meal against reserve two. Both distinct configured source routes were attempted once and refused as unreachable, not proven depleted: t-6 berries, t-7 west mushrooms. t-8 returned to the correct home floor without injury; the supervisor reported needs_plan and shut down the game, leaving the panel online. There were zero collection/deposit receipts. This is failed production evidence, not an outdoor or depleted-site pass.

Code inspection found that stock pressure selection was only reachable while personal reserve/nourishment/recovery already passed. The fix checks return pressures after pending product reconciliation and before personal resupply/recovery, once stock work has been admitted through a stock observation; initial personal acquisition remains possible. A committed return persists if pressures improve en route. Regression scenarios cover low reserve, remaining nourishment below 300 seconds, recovery, weight, slots, continuing delivery, bootstrap acquisition, pending collection accounting and distinct depleted-site fallback with home return. Compact trial status now exposes stock decisions, attempts, collection and deposit evidence without full checkpoints.

Native station repair already exists and uses the ordinary game repair button with verified per-item durability gains. Multi-gather also returns to repair its axe and requires fresh restored durability before resuming. The next maintenance slice should extend/verify household tool and armour repair rather than add another primitive or silently enable unrestricted crafting/upgrades.

**agent:codex/renki-milestones** at 2026-11-01T00:07:18Z

### Deployed pressure check and remaining live blocker

Controller `717a924` passed `just check ci` (`.checks/20261031T235641Z-GVcSB3.log`), committed review (`.checks/20261031T235933Z-Q5iRIB.log`) and controller deploy (`.checks/20261031T235939Z-KWBKrD.log`). PR #137 is stacked on #136.

The west-only trial `.checks/food-20261031T235956Z` (goal `2a089c9eb3817fed0161dc8e`, reserve1, Mushroom target1) reached the west marker normally. t-3 interacted with a ripe mushroom, t-4 picked up exactly one observed mushroom and earned a fresh-inventory collection receipt, and t-5 ate it as the third food. The raised food maximum triggered a recovery return; the committed delivery state persisted as HP recovered above80%. This validates the live pressure transition, not delivery.

Return t-6 reused the recorded outward route until its next segment was blocked by a Greyling near [11,33.22,-9.26] (40m native clearance). A Boar near [-26.87,38.77,-87.74] also constrained alternatives (24m). The navmesh candidate was rejected for hostile_corridor, and the fully bounded local search exhausted4096 nodes with water, physical support, body/log-footprint and hostile refusals. Tas logged out without injury at roughly [-23.17,30.89,-30.41], about40m from home. There is no stock-deposit receipt. Full task and route evidence: `.checks/audit-window-KTZ3ii.jsonl`.

One guarded native recovery, `.checks/recovery-20261101T000427Z`, encountered the same essentially unchanged creatures and again exhausted4096 nodes. It logged out without moving or injury, at72.3/72.3 HP. Full evidence `.checks/audit-window-RA5Dxt.jsonl`. Do not repeat this unchanged attempt or claim a home return. The game is offline and the controller panel remains online. Safe recovery/detour routing is required before another production or repair trial; no combat, water, floor or hostile clearance was relaxed.

Deterministic checks cover distinct depleted sources followed by home return, six pressure conditions, initial personal bootstrap, pending collection accounting, and rejection of a claimed successful return away from home. Actual depleted-site and full delivery evidence remain unearned. Native return failures now retain their reason/detail in the goal. The supervisor can retain a healthy client after an exactly identified source-shortage result with a verified native return and fresh correct-floor shelter; that remains a failed production proof. It cannot retain a failed return or unrelated failure.

**agent:codex/renki-milestones** at 2026-11-01T00:08:15Z

in-progress to blocked: Outdoor delivery and home repair trials await safe recovery around the unchanged hostile corridor; both normal and guarded returns exhausted the native bounded search without injury. See the linked navigation follow-up and retained audits.

**agent:codex/renki-milestones** at 2026-11-01T00:10:08Z

Final full gate passed: `just check ci`, `.checks/20261101T000718Z-QcrM0B.log`. Focused goal tests passed in `.checks/20261101T000502Z-CcQRwJ.log`; food supervisor checks passed in `.checks/20261101T000304Z-wyyuNw.log`. The follow-up native-failure reporting and safe-retention fixes are included in PR #137; final committed review and clean controller deployment are the publication steps. AC1 remains unticked deliberately: positive outdoor stock delivery is not proved. AC4 stays unticked while the complete live sequence, including repair, remains blocked. The navigation follow-up was filed in draft and was not independently promoted.
