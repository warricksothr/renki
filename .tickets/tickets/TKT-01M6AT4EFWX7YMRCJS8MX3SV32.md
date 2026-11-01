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
updated_at: 2026-11-01T06:36:16Z
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

**agent:codex/renki-milestones** at 2026-11-01T06:03:27Z

Commit44e46a0 implements pantry policy, scoped hunting and controls. Full just check ci passed (.checks/20261101T055852Z-XtkC54.log); committed just check review9ef2f6f..HEAD passed without findings (.checks/20261101T060151Z-k1Pq9y.log). Live proof remains open.

The final saved state is44/53.5HP, outside on dry terrain, with three active foods. Default startup healing cannot wait outdoors and default startup-return admission requires50HP. Added an explicit nearby startup profile instead of disabling RecoveryGuard or changing default profiles:40HP/80%, two foods with120s remaining,25stamina,8m hostile clearance;90s/60m measured travel/30m home radius, no combat, immediate injury/water/session logout. Focused RecoveryReturn and supervisor checks pass; full gate/review and field return pending.

**agent:codex/renki-milestones** at 2026-11-01T06:11:31Z

Deployed6222030 with verified package/native/controller checksums. The first nearby launch was rejected before game mutation because vm recovery-return still allowed only local/work_trip; fixed the recipe and added an actual just launcher integration test with fake SSH/SCP (no network). Commitb47828d also covers a complete simulated hunt/loot/return/two-cook/two-deposit refill using distinct protocol receipts; focused checks and full CI pass (.checks/20261101T060723Z-ANJKFF.log), committed review passed without findings.

Native nearby return .checks/recovery-20261101T060723Z reached the bank departure but refused all16 candidates as water/support policy. Tas logged out at44/53.5HP without observed damage or travel. Audit .checks/audit-window-w0y1ua.jsonl: body[-53.51,30.72,11.48], supportY30.69, currentwater30.0–30.02; final reported candidate descended to30.66. No production or route success. Retaining the complete first16-probe bank sweep in bounded diagnostics to distinguish mathematical clearance failures on the uphill bearings; current single-per-stage sample concealed those. Do not resend the unchanged trial or mark route/pantry live acceptance complete.

**agent:codex/renki-milestones** at 2026-11-01T06:26:06Z

Build5b882a8 replay .checks/recovery-20261101T062326Z (audit .checks/audit-window-dSWtzj.jsonl) isolates the first bank issue: executing/dry are true, installed and loaded core identity is reported, and all eligible uphill hits are0.5011–0.5020m from the body. That exceeds the0.501m physical short-step guard despite safe rising support/water clearance. Request49cm instead of50cm so the measured1–2mm hit offsets fit inside the unchanged bound; no water/uphill/body/hostile checks relaxed. Added synthetic returned-hit-offset regression and retained captured-height fixture. Native proof pending corrected deployment.

Also fixed the confirmed-hunt/perception race: the exact kill receipt may settle for its already charged two-second loot wait while perception still contains that target. Check/Next allow no extra creature or action, stale/foreign receipt or renewed expiry. Focused goal checks pass; full CI/review next. Tas remains offline at44HP; no pantry live production claimed.

**agent:codex/renki-milestones** at 2026-11-01T06:31:58Z

Build9edf05b native nearby return passed: .checks/recovery-20261101T062940Z (supervisor .checks/20261101T062940Z-zjo5iH.log). Tas climbed off the saved bank, travelled37.3m, reached actual sheltered floor[-37.13,36.09,9.13], recovered from44HP to52.7/52.7HP and remained online without observed injury. The49cm request correction preserves the original physical step/water/support limits. Full just check ci and committed just check review9ef2f6f..HEAD passed before deployment.

Pantry field trial now running for one real Mushroom from north mushrooms into1:70040 with acquisition/deposit evidence required. Repeated cooking/hunting/refill acceptance remains open. Separately retaining at most six named-area dispatch timestamps in the food acquisition checkpoint so rolling16-outcome history cannot erase15-minute cooldowns; regression replaces the full history and reopens the store before checking expiry.

**agent:codex/renki-milestones** at 2026-11-01T06:36:16Z

Build9edf05b outdoor trial .checks/food-20261101T063143Z reached north mushrooms, earned exact interact/pickup receipts for one Mushroom and began return without injury. That item filled the personal reserve, so no pantry deposit was earned. Return then stalled at support30.8m: audit .checks/audit-window-pEgTQ2.jsonl shows repeated bank entry/completion at[-52.1,30.86,10.73], with the original8s budget expiring despite only one probe. A later wave briefly set inWater, correctly ending work. Final saved state near[-52.2,30.79,10.75],50.1HP, no observed damage; offline with panel restored.

Correcting the boundary oscillation: once bank recovery starts, require both body and footing at least0.9m above current water before handing back to ordinary0.8m planning. This adds10cm settling/hysteresis clearance without weakening dry execution or renewing the original8s/3m budget. Captured boundary regression added. Continue with accessible reserve chest1:70031 included in household scope so scarce gathered food need not all fund bootstrap reserves.
