---
schema: 4
id: TKT-01M6A62GWANGQWSB1GYCZ66G37
title: Validate outdoor household food delivery and fallback
type: task
status: blocked
status_reason: Outer-trip recovery admitted healthy Tas but physical routing found no safe progress leg amid seven loaded hostile envelopes and the coast. Requires the linked route-feasibility follow-up before resupply, repair or another outdoor delivery trial.
priority: high
due_on: null
labels:
  - controller
  - vm
  - live
assignees: []
milestone: M5
parent: TKT-01M6864352N8E28YCGHA19AN90
origin: null
dependencies:
  - TKT-01M6A6Y1XQGNZV4B01QK23J26E
  - TKT-01M6A9EVRB6KY9J8K1G9BV4BZT
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-31T23:52:44Z
updated_at: 2026-11-02T16:15:35Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:claude-code/renki-plan
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

Continue outdoor household validation after the verified home return. The next trial stopped on a targeting Neck about96m from home without injury or receipts, outside the local startup-return radius. Add an explicitly selected `work_trip` one-shot return profile; leave the default local order unchanged. Cap work trips at200m home radius,400m travelled and240s, and require stronger admission: at least90% maximum health (and50HP), three active foods whose shortest duration exceeds the240s return allowance by30s, and35 stamina. Require at least two active foods throughout; preserve every existing injury, water, threat, authority, arrival-floor and shelter gate. Unknown profiles must fail closed.

Expose selection through the existing arming/supervisor recipes, record the profile/limits in native traces, and give the supervisor an explicit420s overall deadline for the selected work-trip profile (including game join/cleanup allowance). Test local compatibility, profile validity, exact new bounds, stronger reserves, terminal outcomes and supervisor forwarding/cleanup. Run focused checks and the full gate, commit and review, install while offline, then attempt one work-trip return under the supervisor. Do not run the incompatible local order or repeat an unchanged failed return. Only after safe home recovery inspect normal food/repair readiness and decide whether another production trial is justified by changed source or supply observations.

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

**agent:codex/renki-milestones** at 2026-11-01T00:41:14Z

The navigation dependency is now done: PR #138 /29db3b8 passed a guarded111m return to the correct home floor without injury; see that ticket's retained recovery evidence.

Retried the positive outdoor test with `just check vm food-check --storage 1:70031 --stock-target Mushroom=1 --sources 'west mushrooms' --require-acquisition --require-deposit --keep-online`. Evidence `.checks/food-20261101T003807Z`; exact audit `.checks/audit-window-R7JtyI.jsonl`. Goal432d0ad89e3af334119b78a4 reached roughly[-61.65,33.37,-85.6] (about96m from home,157m observed travel) but entered needs_plan: nearby threat interrupts household work. A live Neck was targeting Tas about22m away; the unchanged household guard interrupted the source trip. No damage, collection or deposit was observed; final67.9/67.9HP, grounded and dry. The supervisor logged out and stopped the game; the panel remains online. Do not count this as outdoor production proof.

The body is now beyond the existing80m local startup-return radius. Do not retry that order or silently enlarge it. The next implementation should add an explicitly selected, capped work-trip recovery profile with stronger admission reserves, the same injury/water/threat/authority checks, explicit diagnostics, and tests for its independent distance/time/travel bounds. Local default recovery must remain unchanged. This allows a healthy interrupted outer trip to return autonomously without changing the pinned home, using cheats, or asking a person to clear creatures. Positive outdoor delivery and repair remain unverified.

**agent:codex/renki-milestones** at 2026-11-01T00:52:28Z

The explicit work-trip profile is implemented in9b422d6 / PR #139 https://github.com/warricksothr/renki/pull/139. Focused37-case RecoveryReturnTests, supervisor tests, plugin build, full `just check ci` (`.checks/20261101T004543Z-SLizhL.log`) and committed `just check review t3/hostile-return-detours..HEAD` passed. The first gate caught dictionary whitespace; the corrected gate passed. Installation verified package/DLL checksums.

One live `just check vm recover-home DizzysHellheim --work-trip` attempt is retained at `.checks/recovery-20261101T004838Z`, audit `.checks/audit-window-BDzhju.jsonl`. Native trace proves explicit work_trip admission with240s/400m/200m bounds at[-61.67,33.37,-85.61]. The body stayed grounded and dry at67.6/67.6HP, without movement or injury. Native routing found no safe progress leg after3228 expansions/25626 probes against seven loaded hostile envelopes and water/body/support refusals, then logged out. The supervisor stopped the game, restored the panel and cleared the one-shot order. This verifies profile admission and safe failure, not a home return. No unchanged retry was made.

Retain inactive graph diagnostics in RouteSearch instead of dropping them when no estimate becomes active. This does not start an additional search or change physical permission; its regression checks one factory call and a terminal unchanged failure. Further live navigation, personal resupply from remembered mushroom chest1:70040, workbench repair at1:70046, and a changed source trial remain pending. Current gear is worn (axe76, rag chest121 durability) but still usable; memory is a hint and must be refreshed through normal game interactions.

Filed TKT-01M6A9EVRB6KY9J8K1G9BV4BZT — Resolve a multi-threat coastal return bottleneck in draft, with exact threats/positions and required route-feasibility work. The local80m recovery defaults and all hostile/physical/survival checks are unchanged. AC1/4 remain unticked: there is still no outdoor stock deposit and repair has not been tested.

**agent:codex/renki-milestones** at 2026-11-01T00:52:28Z

in-progress to blocked: Outer-trip recovery admitted healthy Tas but physical routing found no safe progress leg amid seven loaded hostile envelopes and the coast. Requires the linked route-feasibility follow-up before resupply, repair or another outdoor delivery trial.

**agent:codex/renki-milestones** at 2026-11-01T00:54:57Z

Final diagnostic follow-up gate passed: just check ci, .checks/20261101T005228Z-EP3pwl.log; focused guidance and plugin lint also passed. Native admission/safe refusal are verified, while outdoor deposit and repair remain explicitly unearned.

**agent:claude-code/renki-plan-m3** at 2026-11-02T15:58:15Z

### Milestone review: serves no M3 goal

M3 grooming, 2026-11-02: outdoor household food delivery serves none of M3's goals 3.1-3.5 in docs/ROADMAP.md. It belongs with its parent TKT-01M6864352 (Pressure-driven work, storage and maintenance goals) in M5. Proposed for M5; the owner decides. It stays blocked on TKT-01M6A9EVRB (Resolve a multi-threat coastal return bottleneck). Live work needs the VM lease, TKT-01M6EF5DTG.

**agent:claude-code/renki-plan** at 2026-11-02T16:15:35Z

Applied on 2026-11-02 by the planning pass: milestone set to M5, to match the exit goal in docs/ROADMAP.md this ticket serves. This supersedes any earlier note that only proposed the move. The owner can move it back.
