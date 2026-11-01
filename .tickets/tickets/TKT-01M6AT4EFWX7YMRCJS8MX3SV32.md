---
schema: 4
id: TKT-01M6AT4EFWX7YMRCJS8MX3SV32
title: Keep the pantry stocked through idle food production
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - safety
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
  branch: t3/pantry-maintenance
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 9ef2f6fa33a1c3e7853c075798a72b211221a3f8
  session: null
  claimed_at: 2026-11-01T05:43:31Z
  expires_at: null
archive: null
created_at: 2026-11-01T05:43:19Z
updated_at: 2026-11-01T05:58:52Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

Implement the user-approved keep-the-pantry-stocked routine: reliable checked base departure/return, stock low/high thresholds, bounded wildlife hunting and source rotation, base cooking/fuel/deposit, panel policy and reasons, and repeatable supervised idle replenishment. Build on household_food rather than a separate planner. Preserve player orders, native permissions, actual inventory receipts and safe cancellation; no spawned supplies or manual clearing as the normal test loop.

## Acceptance criteria

- [ ] Checked base departure and return handle retained route failures without lowering general travel clearances
- [ ] Idle household policy replenishes low stock to a high target with bounded food-specific hunting and source cooldowns
- [ ] Native hunting, ingredient pickup, cooking and pantry deposits earn separate scoped receipts
- [ ] Panel and tools configure pantry policy and expose current decisions and blockers
- [ ] Full CI and committed review pass; supervised repeated replenishment outcomes are retained

## Implementation plan

First fix retained route failures: preserve short-lived negative evidence for physically refused slope approaches and report numeric water/support clearance. Add a separate bounded bank-exit policy only for a grounded, non-swimming, out-of-water body below the normal forecast buffer: fresh measured footing, strictly improving dry clearance, no downhill/wading/jump/run, same body/hostile checks, finite probes/time/distance. Normal0.8m planning buffer remains unchanged; this is live measured recovery with a stricter-than-base actual dry margin, not a general planning waiver. Validate captured positions on explicitly synthetic banks and then the saved native shore return.

Then extend existing household_food with low/high stock thresholds, opt-in food-specific hunts, bounded multi-area wildlife scouting and persistent source cooldown feedback. Reuse native kill/pickup/cooking/transfer receipts, travel leases and current permissions. Ordinary combat requires an exact owned hunt budget, never a generic injury-baseline reset. Configure through panel/API and expose reasons. Validate focused checks, full just check ci, committed just review, publish/link a PR above146, deploy checksum-verified builds and supervise repeated idle-to-pantry episodes. Keep unsuccessful field criteria open; no manual staging/clearing or unchanged failed trial loops.

## Notes

**agent:codex/renki-milestones** at 2026-11-01T05:58:52Z

Pantry implementation now adds low/high refill hysteresis, explicit household Boar/Neck hunting by missing recipe, pinned hunting markers, center-first multi-area search with bounded interior samples and 15-minute persisted visit cooldowns. Exact current hunt task/target may spend the same lifetime ten-damage allowance as food travel; successful matching kill acknowledges injury but earns no ingredient credit. Added panel/tool policy controls and atomic rotation of old resolved goal records (never pending/active). Focused controller checks and bank core checks pass; CI/review and native proof pending.

Native trial .checks/food-20261101T055009Z failed returning through a newly loaded group of Greydwarfs/Shaman. Final audit recorded44HP; no production. Corrected bank recovery to run before flee preflight and compare equal settling allowances on gradual inclines. Native return and repeated hunt/loot/cook/deposit remain unearned; all field criteria remain open.
