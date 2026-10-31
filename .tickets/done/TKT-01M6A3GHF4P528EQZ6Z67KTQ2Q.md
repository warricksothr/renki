---
schema: 4
id: TKT-01M6A3GHF4P528EQZ6Z67KTQ2Q
title: Stock household food through bounded pressure-driven jobs
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - safety
assignees: []
milestone: M3
parent: TKT-01M6864352N8E28YCGHA19AN90
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-31T23:07:58Z
updated_at: 2026-10-31T23:28:44Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

Implement the authorized household food slice independently of tool crafting/repair. Extend durable food goals with optional per-prefab household storage targets, an explicit set of destination chests, and named forage markers. Normal personal resupply remains the first priority. Fresh task-matched chest observations determine known stock; ordinary one-meal withdrawal probes can inspect unknown chests without a new native task. Stock is not inferred from memory. Surplus is deposited through matched native transfer plus fresh pack evidence. Multiple ready-food collections may continue before returning according to remaining shortage, carry capacity, remaining meals, distance and danger; a committed delivery phase prevents oscillation. Cooking reuses existing native task/receipts, bound to home. Idle episodes use the existing behaviour lease/dwell/cooldown/run cap and a runner lifecycle checkpoint hook; interrupted or uncertain actions are never replayed. Diagnostics expose the pressure decision and shortages. Equipment repair and cultivation remain on their existing separate tickets.

## Acceptance criteria

- [x] Configure per-food household storage targets while retaining Tas personal reserve
- [x] Fresh unique chest receipts and matched deposits prove stock; memory and missing access cannot claim success
- [x] Bounded idle episodes select collection, cooking and committed delivery from shortages, capacity, distance and danger
- [x] Cancellation, restart uncertainty and edited scopes cannot repeat uncertain native actions
- [x] Focused regression checks, full gate and a bounded live household stock episode retain honest evidence

## Implementation plan

Implement the authorized household food slice independently of tool crafting/repair. Extend durable food goals with optional per-prefab household storage targets, an explicit set of destination chests, and named forage markers. Normal personal resupply remains the first priority. Fresh task-matched chest observations determine known stock; ordinary one-meal withdrawal probes can inspect unknown chests without a new native task. Stock is not inferred from memory. Surplus is deposited through matched native transfer plus fresh pack evidence. Multiple ready-food collections may continue before returning according to remaining shortage, carry capacity, remaining meals, distance and danger; a committed delivery phase prevents oscillation. Cooking reuses existing native task/receipts, bound to home. Idle episodes use the existing behaviour lease/dwell/cooldown/run cap and a runner lifecycle checkpoint hook; interrupted or uncertain actions are never replayed. Diagnostics expose the pressure decision and shortages. Equipment repair and cultivation remain on their existing separate tickets.

## Notes

**agent:codex/renki-milestones** at 2026-10-31T23:22:54Z

Implementation is on t3/household-food-stock above navigation PR #135. Adds optional stock targets to durable food goals, explicit unique destinations, current-run three-minute chest observations earned by normal one-meal probes, verified surplus deposits, source marker pinning, pressure-based multi-collection and committed delivery. Guarded idle episodes have distinct durable IDs, runner lifecycle checkpoints and real goal-stop routing to their owning behaviour. Household cooking returns precisely home before eating/healing. Panel controls and desktop/mobile request automation passed; focused goal/behaviour/app checks passed. Full gate is running; live household cooking/deposit trial remains pending. Broader repair-dependent pressure ticket remains draft with its dependency intact.

**agent:codex/renki-milestones** at 2026-10-31T23:28:44Z

Live household idle trial passed in .checks/food-20261031T232425Z. The temporary job started after its30s storage-marker dwell and created durable goal5508aae5f933be4f17357511. Nine native tasks withdrew three yellow mushrooms and two neck tails, ate one yellow mushroom, cooked two grilled tails, returned precisely to the home waiting point after each batch (t-5/t-8), ate one grilled tail and deposited the other through exact chest/task/inventory receipt t-9 into1:70031. Fresh household stock reached NeckTailGrilled1/1; two carried yellow mushrooms remained as the personal reserve. Final position[-35.45,36.09,7.58], three active foods,69.8/69.8HP, sheltered, idle, injurySeq0 throughout. The helper disabled its temporary definition and kept Tas safely online.

An earlier supervisor admission (.checks/food-20261031T232332Z) used an invalid idle repeat setting and was refused before starting work. Corrected the helper to trigger repeat/cooldown300s/maxRuns1, added a policy regression and retained the failed evidence. Cleanup now only disables a definition after a successful save, preserving the original error.

Focused goal/behaviour/app checks, desktop/mobile household UI checks, final full CI (.checks/20261031T232330Z-Ps7P0Q.log) and committed review (.checks/20261031T232635Z-57bHOm.log) passed. Code is committed as15b20c3 in PR#136, controller-only deployment .checks/20261031T232309Z-GCezrt.log. Native navigation remains e14d7b1. The deployed controller was built from these functional changes before commit; its version stamp is711795c-dirty. No client restart was required for household code deployment.

Remaining live evidence: outdoor multi-source household collection with several yields before a return, depleted sources, and capacity/danger-driven delivery. Those policies have focused regression coverage; this live trial proves idle admission, normal resupply, home cooking, precise waiting-point returns, stock deposit and safe completion. The broad repair-dependent production ticket remains draft with its original dependency intact.

## Summary

Implemented and deployed household food targets, guarded durable idle episodes, pressure-driven collection/delivery and panel controls in PR #136. Live idle home cooking and matched stock deposit passed with personal reserve, three active meals, full health and no injury. Outdoor multi-source production remains a separately documented live check; equipment repair stays on its existing blocked draft ticket.
