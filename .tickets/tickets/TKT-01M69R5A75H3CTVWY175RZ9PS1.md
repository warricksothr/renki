---
schema: 4
id: TKT-01M69R5A75H3CTVWY175RZ9PS1
title: Acquire missing food through bounded foraging and hunting
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - safety
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-milestones
  branch: t3/food-acquisition
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: aa39fe27533640f972b4ad549a88de71235da80c
  session: null
  claimed_at: 2026-10-31T19:50:40Z
  expires_at: null
archive: null
created_at: 2026-10-31T19:49:36Z
updated_at: 2026-10-31T20:00:18Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

Extend production food resupply after storage exhaustion with bounded normal resource collection, remembered foraging sites, explicit eligible wildlife hunting and verified home return. Preserve native permissions, current body ownership, injury guards, task receipts and durable attempt budgets. No manual supply staging is required.

## Acceptance criteria

- [x] Food and fuel resources use exact yield, fresh pack and scoped result evidence; no success-text-only gains
- [x] Foraging and hunting attempts and visits persist with finite limits, fresh eligibility and normal combat permissions
- [x] Food acquired outside home returns on the original floor before cooking, completion or escort handoff
- [ ] Focused replay checks and complete CI pass; guarded live results and remaining cases are recorded

## Implementation plan

Extend new food goals with a typed, optional acquisition checkpoint; old saved goals keep their admitted storage-only behavior. Search storage first, then exact live ready-food/raw drops and pickables with validated yields. Save resource and site attempts before dispatch (12 resources, six remembered food sites, at most two explicitly named ordinary wild Boar/Neck hunts). Resolve pickable products only by fresh measured inventory gain or a fresh matching nearby spawned drop; no animation/result text counts as inventory. A kill needs the exact native task, target and observed species death; it is independent from acquiring ingredients.

Remembered berry/mushroom/branch positions are hints within 80 m of pinned home, never stock or route evidence. Native dry/hostile-avoiding travel owns routing; a failed site consumes one visit and does not assert empty coverage. Current native permissions and combat guards remain authoritative. Hunting requires current attack permission, active food, stamina and eligible isolated wildlife. Keep existing strict injury stop; never reset injury to admit a continuation.

After every acquired yield, return to the original home level before cooking, finishing readiness or handing back to escort. Attempts, post-interact waits and uncertainty survive restart without replay. Keep summaries compact. Add replay checks for source spoofing/missing yields, partial pickup, products and auto-pickup, scoped home arrival, persisted exhausted visits, unsafe wildlife/current permissions and checkpoint failure. Run focused just checks, full just check ci, commit/review/push a PR stacked on food-resupply, then deploy the controller and supervise real production food goals. Record unavailable supplies or cold-fire cases honestly; no manual staging requests or fabricated items.

## Notes

**agent:codex/renki-milestones** at 2026-10-31T19:59:42Z

Implemented an optional typed acquisition checkpoint on new food goals; saved older food goals retain their original storage-only scope. After storage exhaustion, normal exact-yield drop/pickable collection can supply ready food, raw ingredients or fuel wood. Product RPC/auto-pickup handling requires fresh measured pack gain bounded by the observed yield, not success text. Short product waits, resource IDs, visits and two-hunt attempts are persisted before dispatch with the existing native task budgets and uncertain-action rules.

Remembered edible/branch sites within 80 m are travel hints only; six visits, consolidated in four-metre cells, do not assert area coverage. Twelve resource actions are allowed. Current native attack permission can admit only exact fresh isolated unstarred wild Boar/Neck targets; normal native equipment, ground, bystander and retreat checks still decide. Target-lost/unreachable hunts can advance to distinct prey, never retry the same ID. An exact task/target/species observed death is distinct from obtaining any ingredient. Existing injury baseline remains strict. Every collected yield outside home returns to the original floor before cooking/completion/escort continuation; a below-floor arrival cannot pass.

Added compact panel acquisition counts and --require-acquisition to just vm food-check. Corrected controller-only deployment to preserve an offline client while bringing back the independent panel. Focused just check controller test-packages ./internal/goal ./internal/app ./internal/chat passed in .checks/20261031T195547Z-S9C5dC.log; subsequent checks include the added distinct-prey negative recovery. Initial full gate caught stale panel output; rebuilt through just check panel build. Final just check ci is running in .checks/20261031T195717Z-pOcFBT.log. Live acquisition, combined raw-to-cooked fallback, cold-fire fueling and escort continuation are not yet claimed. Tas remains reserved for automated food trials; no manual setup requests.
