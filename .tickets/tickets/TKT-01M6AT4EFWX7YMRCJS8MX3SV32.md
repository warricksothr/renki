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
updated_at: 2026-11-01T10:51:00Z
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

Preserve the current walking, water, slope, task and bank limits. Extract the existing support-hit selection into a pure tested policy without changing which hit wins. Retain bounded per-query failure evidence (missing hits, slope rejection, footing band, selected height band, saturation), numeric normals/heights and collider identity through exact-query memo hits. Attach that evidence to navmesh/live/local refusals and snapshot unfinished local/replay search when bank recovery interrupts it.

Add a reusable just vm route-benchmark supervisor: validate existing marker routes, require same-world fresh safe-home observations and native outward/return receipts for each finite round trip, stop on failure/command/scope change, retain compact metrics and evidence. Recovery of the currently offline shore position is a prerequisite, never an automatically repeated failed trial. Tests cover support selection on stacked floors/slopes and cached failures, benchmark false success, unsafe home, player interruption and cleanup. Run focused recipes, full just check ci, commit/review, deploy together, then one instrumented supervised nearby return. Use measured evidence to decide the next route correction; no unchanged retry or manual rescue dependency. Keep pantry and ten-return field criteria open until earned.

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

**agent:codex/renki-milestones** at 2026-11-01T06:41:55Z

Build148dd78 nearby return .checks/recovery-20261101T063927Z stopped at admission before movement; .checks/audit-window-zRuhkE.jsonl retained50.1/53.7HP, dry ground, no nearby perceived creatures and food durations109/406/893s. Caller incorrectly used the minimum across every active meal, requiring the optional third meal to satisfy the two-meal120s nearby reserve. Core helper now measures the second longest duration for nearby/local admission, while work_trip retains all three. Nonfinite/negative/missing durations fail closed. Captured regression checks admission/shortage and unchanged work-trip requirements. One bounded refusal trace now reports exact native reserves to reduce future diagnosis context. Tas remains offline; no further field proof earned.

**agent:codex/renki-milestones** at 2026-11-01T06:48:33Z

Build86b47f5 nearby return .checks/recovery-20261101T064503Z passed the corrected meal admission but stalled on its first physically valid bank candidate: audit .checks/audit-window-AGi7Y3.jsonl shows0.4m initial travel to[-51.92,31.01,10.58], then only centimetre bobbing until the original8s budget expired. No water/threat/probe refusal or damage; exactly one sampled candidate. The retained target lacked execution feedback.

BankExit now abandons a held candidate after1s without3cm closer approach, continues the existing bearing sweep and retains original8s/3m/128-probe limits. True short-step progress resets only the per-candidate stall timer; bobbing does not. Added exact no-progress/real-progress and total-budget regressions and one bounded native stall trace. Next native return and pantry proof remain pending; offline, about50.1HP.

**agent:codex/renki-milestones** at 2026-11-01T07:07:03Z

Continuation: the dirty bank build passed nearby return in .checks/recovery-20261101T065742Z (actual sheltered home-floor arrival, no injury). The following idle pantry trial .checks/food-20261101T065917Z attempted six distinct sources and travelled over350m but earned no food collection, cooking or deposit. Its home return failed on the shore and the supervisor logged out; final46.4HP with no observed injury. Audit .checks/audit-window-9BYSr2.jsonl shows repeated planning-buffer recovery followed by native sliding on support normals around0.62. Installed Character.ApplySlide uses a38-degree player threshold; ordinary dry support planning previously accepted normals down to0.35. Ordinary dry endpoint/midpoint samples now require a walkable normal of at least0.8; actual footing and bounded bank recovery remain separately measured. This is a new route-admission correction; do not repeat the old failed route unchanged. Pantry field criteria remain open.

**agent:codex/renki-milestones** at 2026-11-01T07:21:42Z

Reviewed/deployed e19ce26 native return .checks/recovery-20261101T071741Z remains incomplete:0.9m observed travel, no injury, final46.2/46.2HP, then native logout. Audit .checks/audit-window-E2ZUAi.jsonl shows walking-slope admission now rejects sliding navmesh samples. The bank reached walkable footing near[-51.81,31.08,10.45], but planning waited long enough for a wave-water change to30.122m to re-enter recovery after the original aggregate8s deadline. No budget was renewed, margin waived, body puppeted or home relocated. Pantry production remains unproved. Next change needs stable bank handoff across planning/wave variation under original bounds; no unchanged retry. Portal and building collision work is deployed and documented separately.

**agent:codex/renki-milestones** at 2026-11-01T07:33:54Z

Captured bank handoff now reserves20cm of liquid variation beyond the existing10cm settling allowance before holding for route planning. The same conservative planning margin applies when admitting a new wait near the shore; ordinary walking keeps its unchanged0.8m buffer. The regression uses initial[-51.81,31.08,10.45]/30.98m footing and later[-51.9,31.01,10.45]/30.92m footing at30.122m liquid height, rather than attributing all loss to the wave alone. All original8s/3m/128-probe bounds and fresh dry/body/hostile checks remain. Focused BankExit checks and full just check ci pass. Reviewed deployment and one changed-policy native return follow; no production or return result claimed yet.

**agent:codex/renki-milestones** at 2026-11-01T07:37:52Z

Changed-policy return .checks/recovery-20261101T073423Z failed without injury after 0.7m travel; audit .checks/audit-window-qejxWc.jsonl. Recovery reached body 31.20m / footing 31.11m above water 30m. Two navmesh candidates failed endpoint support; footing settled to 31.10m while planning. Completion and waiting re-entry both used 1.1m, so tiny settling re-entered recovery after its original deadline. Correcting this with distinct 1.1m completion / 0.9m waiting thresholds (ordinary travel remains 0.8m), captured-position/wave regression, and explicit refusal reason diagnostics. No budget reset or physical guard waiver. Tas is offline and panel remains available. Pantry production and native portal-trigger avoidance remain unproved.

**agent:codex/renki-milestones** at 2026-11-01T07:45:44Z

Reviewed d5e74f6 passed just check ci (.checks/20261101T073806Z-VxuFrP.log) and committed just check review 4337eb4..HEAD (.checks/20261101T074045Z-twhKvO.log), then was pushed and checksum-verified on BLIGHT. Supervised return .checks/recovery-20261101T074138Z failed without observed injury. Audit .checks/audit-window-OFT5N7.jsonl shows endpoint_support rejection during planning, then water rose from 30m to 30.89m and bank recovery reached distance_limit (the original 3m aggregate bound). This is distinct from the earlier handoff timeout. Native camera .checks/camera-20261101T074237541679Z.jpg shows steep terrain below the raised deck, not an interior doorway or portal.

Tas is offline; the controller-only panel has restarted. No unchanged retry. Next investigate support-ray refusal normals/heights and preserve unfinished local-search diagnostics when bank recovery interrupts it; use those measurements to choose a checked approach/staging route, without extending limits or lowering clearances. Pantry and native portal-trigger criteria remain open. This result supersedes the pending native-test statement in the previous note.

**agent:codex/renki-milestones** at 2026-11-01T10:17:35Z

Diagnostics/benchmark 00d8161 passed full just check ci and committed review, was pushed and checksum-deployed. Instrumented return .checks/recovery-20261101T100938Z failed without observed injury; audit .checks/audit-window-hS4MYw.jsonl. Direct endpoint refusal is confirmed slope: terrain normal Y 0.76 near [-50.65,31.56,8.77], below unchanged 0.8 walking limit. Local search remained Searching at 3353 expansions / 26824 probes after 16.02 seconds and had explored supported ground beyond the bank. No repeated route benchmark has been earned.

Implementing a bounded checked staging handoff: after 2s near the shore, only expanded local-search nodes with checked incoming edges, >=3m displacement, >=1m progress toward the original goal, <=12m path cost and >=2m current-water clearance can offer a staging leg. Fresh destination-water and ordinary live body/support/portal/hostile checks remain. At most two such legs per goal across replans; no bank/task deadline or probe allowance is reset. Synthetic captured-start/contour tests distinguish staging from real arrival and missing support from geometric eligibility. The 2m destination preference accounts for observed 0.89m wave variation plus ordinary walking buffer/headroom; it is not a general water-policy relaxation. Next native run follows completed CI/review, not an unchanged retry.

Also fixing route-evidence relative paths at the just module boundary with an actual-recipe regression, and requiring 95% HP for benchmark keep-online after at most 90s safe-home healing. Tas is offline with controller-only panel running.

**agent:codex/renki-milestones** at 2026-11-01T10:34:00Z

Reviewed 9f993c2 passed full just check ci (.checks/20261101T102213Z-yEDJii.log), committed review and checksum installation. Changed-policy return .checks/recovery-20261101T102531Z / audit .checks/audit-window-BU9uSQ.jsonl demonstrated a native checked staging leg: 374 expansions, 2992 probes, 2.02s, supported endpoint [-45.73,32.37,5.26], about 7m physical travel without injury. Full return remained unsuccessful: the next local search was still Searching at 2861 expansions / 22888 probes after 16.02s. Tas logged out; controller-only panel remains online. Neither home arrival nor any repeated route/pantry receipt is earned.

Addressing remaining full-floor search ordering with an opt-in 3D distance-to-arrival-cylinder estimate for nearby dry searches. Supported edge/arrival rules, 4096-node/16s-stage limits, ordinary travel clearance and bank budgets remain unchanged. Distant frontier legs and supplied regional guidance keep the original estimate. HeightRouteTests prove fewer expansions through a synthetic stair connection, no invented support and identical distant frontier routing. Add bounded piece-type slope diagnostics and unrounded normals to diagnose native building support rather than guessing at stair colliders.

The automatic camera checkpoint did run but captured a loading screen. Delay the single optional attempt until >=6s after the first in-world state; no camera settings change. This is separate from movement success. Next native return is on the corrected search order after CI/review, not an unchanged retry.

**agent:codex/renki-milestones** at 2026-11-01T10:41:55Z

Reviewed43ee7d8 passed just check ci (.checks/20261101T103450Z-RzYVjq.log), committed review (.checks/20261101T103816Z-JfJM9a.log), push and checksum installation. Changed-policy return .checks/recovery-20261101T103852Z / audit .checks/audit-window-Cb7pe0.jsonl travelled10.6m without observed injury to[-36.92,34.11,7.52], still below home. Its final search remained Searching at2918 expansions/23344 probes after16.02s. Native logout and supervisor cleanup restored controller-only panel. No home/benchmark/pantry success earned. Optional delayed camera now captures actual world imagery.

Exact diagnostics identify a concrete admission bug: rotated wood_floor hits have normal Y1.0000001192092896, and WalkableNormal's exact upper bound1 rejects them as slope. This also prevents observed entrance floors from becoming hints. Correcting only the upper numerical bound to1+1e-6; the lower0.8 slope threshold, support heights, body/portal/water checks and all route/bank deadlines remain unchanged. Captured support and synthetic hint integration regression accept the measured flat floor while rejecting the captured0.446 stair face; boundary cases retain nonfinite/large-overshoot/sub0.8 rejection. CI/review and one corrected-policy native return follow; no unchanged retry.

**agent:codex/renki-milestones** at 2026-11-01T10:51:00Z

Reviewed4c8b3bf passed full CI (.checks/20261101T104202Z-3n4DmO.log), committed review (.checks/20261101T104426Z-V615oX.log), push and checksum installation. Corrected-policy return .checks/recovery-20261101T104459Z / audit .checks/audit-window-bQImaU.jsonl confirms both door floors now become entrance hints at Y36.09. The exact-normal correction is demonstrated; complete routing remains unproved. Tas moved only0.3m and stayed below the base near[-37.04,34.11,7.57],45.2/45.2HP, without observed injury. Both entrance searches and the final home search ended Searching at16s. Native logout and supervisor cleanup restored controller-only panel.

Current summaries conceal whether any upper-floor nodes were actually expanded, and their last-per-stage refusals come from unrelated distant terrain. Add bounded diagnostic-only highest/closest expanded support, target-floor expansion count and closest refusal per stage ordered by goal distance. Core regression distinguishes reachable upper-floor nodes from missing support and final short arrival connectors. The existing route-evidence projection retains these summaries. No physical policy, target, budget or autonomous goal is changed. One instrumented run after gate/review will distinguish an actual entrance obstruction from queue exhaustion; no production receipt is claimed.
