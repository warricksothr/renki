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
  - live
assignees: []
milestone: M5
parent: TKT-01M6864352N8E28YCGHA19AN90
origin: null
dependencies:
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
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

Implement the user-approved keep-the-pantry-stocked routine: reliable checked base departure/return, stock low/high thresholds, bounded wildlife hunting and source rotation, base cooking/fuel/deposit, panel policy and reasons, and repeatable supervised idle replenishment. Build on household_food rather than a separate planner. Preserve player orders, native permissions, actual inventory receipts and safe cancellation; no spawned supplies or manual clearing as the normal test loop.

## Acceptance criteria

- [ ] Checked base departure and return handle retained route failures without lowering general travel clearances
- [ ] Idle household policy replenishes low stock to a high target with bounded food-specific hunting and source cooldowns
- [ ] Native hunting, ingredient pickup, cooking and pantry deposits earn separate scoped receipts
- [ ] Panel and tools configure pantry policy and expose current decisions and blockers
- [ ] Full CI and committed review pass; supervised repeated replenishment outcomes are retained

## Implementation plan

User approved closing automatic personal-food bootstrap, then testing repeated household loops. Extend the existing durable food goal and behaviour lease rather than starting an independent task dispatcher. Add an opt-in idle home-resupply preset with higher priority than daytime chores/rest and explicit awake/sheltered/fresh/near-home admission. Detect low nourishment or an insufficient carried reserve; a satisfied predicate must create no new durable episode. Use native receipts and the existing goal limits, safety, home pinning, uncertain-action handling and cancellation. Sleep and newer human orders remain protected; never add automatic logout.

Expose the preset through the authenticated behaviour API and existing behaviour rows/tools. Add meaningful regressions for zero-food25HP bootstrap from the bedroom, proactive reserve refill, satisfied no-op, stale/foreign/home-changed/below-floor/sleeping/threat admission and cancellation. Run focused checks, full just check ci, commit/review, and controller-only deploy. Verify an actual automatic idle-started resupply from the current empty pack, then enable/observe bounded household production goals with real low/high storage targets, source rotation and separate acquisition/cook/deposit receipts. Record honest repeated outcomes and address blockers instead of manually clearing or fabricating supplies. Desired resource outcomes build on the existing stock targets; aggregate meal categories and unsupported soup recipes remain separately scoped follow-up work.

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

**agent:codex/renki-milestones** at 2026-11-01T11:03:32Z

Reviewedf243371 passed just check ci (.checks/20261101T105408Z-4jqUWU.log), committed review (.checks/20261101T105631Z-dxQePV.log), push and checksum deployment. Instrumented return .checks/recovery-20261101T105655Z / audit .checks/audit-window-HtySsG.jsonl provides the missing distinction: both entrance searches expanded zero target-floor nodes, highest support about34.98m, with nearest body casts hitting wood_floor underneath the doors nearY34.4. No return earned; offline at[-37.07,34.1,7.58],44.4/44.4HP, no observed injury, panel restored.

Implement checked under-floor escape staging: only when a raised dry/hostile-avoiding search starts beneath observed building geometry, select an expanded node outside overhead pieces with >=3m displacement, >=0.25m rise, <=12m checked cost and <=8m additional horizontal distance from the target. This permits deliberate outward repositioning instead of searching directly beneath an entrance. Fresh destination overhead/water checks, ordinary edge/body/portal/hostile admission, >=2s wait and the same aggregate two-leg allowance remain; no task/bank budget renewed or arrival invented. Shore staging for raised goals also avoids endpoints under a building. Captured-start/synthetic-overhang regression and native build pass; full gate/review and one changed-policy return remain pending.

Route-evidence now includes bounded entrance selections and native startup return/logout reasons so future diagnosis does not require raw audit scans. This supersedes any inference that the normal correction alone completed the entrance route.

**agent:codex/renki-milestones** at 2026-11-01T11:15:59Z

Build630dd11 passed just check ci (.checks/20261101T110342Z-1mBaqi.log), committed review (.checks/20261101T110631Z-OqSG2f.log), push and checksum deployment. Return .checks/recovery-20261101T110659Z PASSED:37.4m observed travel, sheltered floor[-37.01,36.09,9.12],43.7/43.7HP, no observed injury. Audit .checks/audit-window-Tj3Mql.jsonl records full checked entrance routes at676 expansions/4.18s and608 expansions/3.78s. No staging leg was used. This proves the full return on the build, not native under-floor escape selection or repeated route reliability; do not attribute the successful path specifically to an unused branch.

The strict benchmark has not been run or passed: pinned homeY37.66 versus observed floorY36.09 exceeds its0.75m vertical criterion, and current raw-marker distance3.23m exceeds its3m start bound. Keep home and strict threshold unchanged; future benchmark support for legacy eye-height markers needs an explicit observed floor reference pinned to the same home/world/connection. Do not count a benchmark refusal or this recovery as repeated proof.

Production food resupply .checks/food-20261101T111055Z PASSED with native stored deer-meat withdrawal/eating, reserve2, active food and sheltered health recovery; two tasks/receipts. Tas remained online. Then automatic pantry trial .checks/food-20261101T111255Z (one Mushroom, north mushrooms, acquisition+deposit required) stopped at admission with zero tasks: nearby threat prevents food resupply. Saved before-map.json shows six Greydwarfs and three Greylings, all targeting Tas, several at the doorway. No injury observed; game stopped and independent panel restored. Do not retry against the unchanged base attack or request manual enemy staging as the normal loop. No pantry deposit or milestone acceptance earned; field ACs remain open.

Next development: retain these distinct return/resupply successes and threat refusal; resolve strict benchmark floor provenance without moving home or loosening acceptance, then repeat routes and restart autonomous pantry production when native threat admission permits. Current state: game offline, controller/panel online, installed build630dd11.

**agent:codex/renki-milestones** at 2026-11-01T14:03:21Z

Implemented authoritative random-event identity/notice/centre/radius/timer/membership and separate local notices without invented bounds. State, fresh panel status and shared map/export expose the same knowledge; stale/offline observations become unknown. Perception distinguishes event-spawn provenance, owner-authoritative target knowledge and effective player-hunt mode. Pantry receipts settle first; household work defers at observed home shelter without actual nearby/pursuing threats, shares the 240s wait budget, may eat carried food, and needs fresh clear rather than timer expiry before resuming. Separate personal resupply remains available. Distant event bounds are not yet general route exclusions.

Focused native build/core geometry checks, controller protocol/session/app/goal checks and desktop/mobile world-events UI checks passed. Full just check ci is running. Native event observation and ownership-transfer checks remain unearned; Tas remains offline after the prior nine-attacker capture. No raid identity can be inferred from that older capture. No pantry repeated acquisition/cook/deposit or group-combat pass is claimed.

The concrete native group-peeling design is in docs/WORLD-EVENTS.md. Filed TKT-01M6BPPVDE5S8Q5A61N218JXDM — Peel isolated enemies from groups with native pursuit checks — as draft under the attack ticket for subsequent implementation. Current melee eligibility remains ordinary Greyling/Boar/Neck; six Greydwarfs require additional combat preparation/support. No crowd-limit weakening or blind live login.

**agent:codex/renki-milestones** at 2026-11-01T14:04:26Z

Full just check ci passed (.checks/20261101T140128Z-kKghIN.log), including 50 desktop/mobile panel scenarios. Live event/ownership verification, group peeling, and repeated pantry acquisition/cook/deposit remain open.

**agent:codex/renki-milestones** at 2026-11-01T14:07:13Z

Committed/reviewed/pushed event awareness as 7e8a2d8 on PR147; just check review fcbda3b..HEAD passed with no findings. just check vm install passed package checksum verification (.checks/20261101T140531Z-yCCJIh.log). Independent panel restarted; native game remains stopped, controller API reports version7e8a2d8 connected=false, and status/map omit unavailable world-event knowledge. Native event field validation, group peeling and pantry field acceptance remain open.

**agent:codex/renki-milestones** at 2026-11-01T20:55:08Z

Live observation on 2026-11-01 at 20:51:48Z, after restoring the pre-migration idle definitions: Tas is sheltered at home, idle, HP25/25 (unfed base health), no active food and no carried food. No food_resupply or household production goal is running. The home cooking, food storage and cargo idle definitions each show five completed attempts; their latest results require an active meal and resupply first. West-mushroom foraging has zero attempts because Tas is not there. AllowGather=true; AllowCombat=false. The background food keeper can eat carried food but does not acquire it and does not ask for food at full base-health ratio. This leaves idle maintenance unable to bootstrap itself from an empty pack. Preserve ordinary safety: evaluate an explicit personal ready-meal resupply prerequisite triggered before food expires, then hand back to idle work. This is an observed control-flow gap, not evidence that home chests are empty. User requested a walkthrough, not a behavior/configuration change, so none was made. Evidence: .checks/modpack-2323/current-observation.json, current-behaviours.json and current-goals.json.

**agent:codex/renki-milestones** at 2026-11-01T23:39:54Z

Current batch food_resupply passed from25/25HP with zero active/carried food: .checks/food-20261101T233548Z, goal e0b2068ae63f10c9361662c1,8 measured tasks. It inspected current storage, ate verified CookedDeerMeat and NeckTailGrilled, withdrew/ate Raspberry, stocked MushroomYellow, recovered health and verified safe home while remaining online. just check vm food-check --reserve2 passed in80s (.checks/20261101T233548Z-cnL1ho.log).

A fresh current-layout cooking test also passed in57s (.checks/20261101T233751Z-zf5WId.log; .checks/cooking-20261101T233751Z). One chest inspected, station1:73602, normal direct_spawn CookedDeerMeat output and matched inventory delivery count1. Tas reached89.2/89.2HP, no active body task and remained sheltered/online. This validates resupply/bootstrap and one batch; it does not establish the extended autonomous forage/hunt/cook/deposit pantry loop.

**agent:codex/renki-milestones** at 2026-11-01T23:41:48Z

Correction to the previous validation note: the exact recipe invocation was just check vm food-check --reserve 2 (with a space before 2).

**agent:codex/renki-milestones** at 2026-11-02T05:20:18Z

Current fresh observation .checks/20261102T051239Z-E43cry.log reproduces the bootstrap gap: Tas awake/sheltered in the bedroom on day145,25/25HP, no active or carried food, idle, injury0. Existing rest works; cooking/cargo refuse for lack of an active meal. This does not establish empty storage.

Implemented personal_food as another mode of the existing FoodWork durable-episode adapter. The authenticated idle-resupply preset pins saved home, admits daytime awake grounded sheltered bodies within12m horizontally/1.75m vertically, uses15s dwell/priority10/five-minute cooldown/100 attempts per controller session, and preserves ordinary lease/cancellation and native food-goal limits. No new dispatcher, model call, logout or weakened outdoor admission. Nourishment with300s remaining plus a two-meal pack reserve is the readiness predicate; full unfed HP does not satisfy it. Satisfied needs create no durable goal. Existing nighttime attachment/idle suppression remains in force.

Focused just check controller test-packages ./internal/goal ./internal/app ./internal/behaviour passed (.checks/20261102T051838Z-ZbT8YP.log), including captured bedroom empty-pack return/storage progression, proactive reserve/meal-expiry pressure, satisfied no-op, sleep/stale/home/floor/threat/busy admission and uncertain cancellation. Extended just vm food-check with --idle-resupply MARKER to verify natural idle dispatch and its episodeId; its temporary definition is disabled afterward. just check vm test-food-check passed (.checks/20261102T051846Z-aYAcoz.log). Full gate is running.

User also proposed editable desired resource outcomes. Existing stock.targets/minimum already express per-item pantry outcomes with real chest receipts and personal supply precedence. Documented this foundation alongside the prerequisite. Aggregate categories such as ten ready meals and unsupported recipes such as mushroom soup need additional planner/recipe support and are not claimed by this change. Live automatic bootstrap and consecutive full acquisition/cook/deposit/rest loops remain unearned.

**agent:codex/renki-milestones** at 2026-11-02T05:25:51Z

Controller0347b4f passed full just check ci177s (.checks/20261102T051903Z-sIS8ie.log), committed review (.checks/20261102T052204Z-YEQdKS.log), and controller-only deployment (.checks/20261102T052219Z-5zhXpw.log). The automatic idle trial .checks/food-20261102T052233Z created goal68f1539ea4bf9570e7d03738, returned from the bedroom, checked the real food chest, normally withdrew1Blueberries and ate it with matching receipts. Tas remained sheltered/online, no injury,28.5/33HP with a fresh active meal. This proves the automatic trigger and real storage/eat path, but not full reserve completion.

The episode failed at a recovery wait with "invalid or expired bounded wait". Engine.Tick used a clock sample from view construction for the runner, then a later sample for deadline validation; a persisted wake between those samples was future to the runner and expired to the engine. Align View.Now with the tick's validation timestamp. A deterministic advancing-clock regression reproduces a stored wake boundary and requires normal completion on the next tick; wait/deadline budgets are unchanged.

Also separate daytime admission from ongoing execution: a new personal episode requires awake daytime shelter, but sunset alone must not cancel an already dispatched native food/cooking action. Remove the preset's continuing night condition, retain admission guard and sleeping/owner-command precedence, and add a sunset-during-owned-action regression. Full revalidation and another automatic live episode follow. The first goal's failed result and exact receipts remain retained; no automatic replay of its actions or logout occurred.

**agent:codex/renki-milestones** at 2026-11-02T05:41:16Z

All 24 outstanding PRs129–152 are merged into main88f4943; no open PRs remained after landing. Controller8653c38 and plugin6e23e2d remain deployed. Permanent automatic personal-food preset passed without a manual body task: goal34db7b46b5898b16d97ac004 inspected actual storage, withdrew3MushroomYellow, ate1 and verified2reserve; safe sheltered completion, no injury (.checks/20261102T053423Z-6rUGlo.log).

Household CookedMeat2 trial .checks/food-20261102T053524Z cooked a verified batch but failed before deposit. Native home-return t-10 succeeded on frame2342; goal consumed state2341 (81cm from home) before arrival state2344 (36cm from home). This is receipt/state ordering rather than a failed route. Carry the immutable terminal native sequence through session.Task and behaviour.Result and require newer state before reconciling a pending food action. No relaxed arrival threshold or replay. Captured-frame regression and native late-receipt immutability checks pass with just check controller test-packages ./internal/session ./internal/behaviour ./internal/goal ./internal/app (.checks/20261102T054048Z-tFFyTW.log). Full gate, deployment and cooking/deposit retry follow. Broader pantry acceptance stays open; Tas remains online.

**agent:codex/renki-milestones** at 2026-11-02T05:52:10Z

Arrival fix75db0ae passed focused checks, full just check ci178s (.checks/20261102T054116Z-9E2coD.log) and committed just check review HEAD~1..HEAD (.checks/20261102T054418Z-76JbSn.log). Controller-only deployment passed (.checks/20261102T054435Z-WyVc12.log), preserving Valheim. PR153 merged as43de40a; all25 PRs129–153 are landed and the open PR list is empty. Controller75db0ae, plugin6e23e2d.

Retained live results:
- .checks/food-20261102T054448Z stopped at not-in-world while the controller reconnected; no goal or body work.
- .checks/food-20261102T054518Z goalc45a0cd5c6f97434d9b58b3d correctly succeeded with one stored CookedMeat satisfying minimum1; supervisor required a deposit and correctly rejected this as the requested proof.
- .checks/food-20261102T054548Z goal524b1e92efd505813eb75aaf correctly reached exact target2 with one surplus-pack deposit. Supervisor's additional cooking predicate correctly failed because no cooking was needed.
- .checks/food-20261102T054625Z goala4914aa72b61b3d96fe0c6b8 exercised target4. Native raw withdrawal, t-9 cooking/direct_spawn delivery, t-10 precise cook_return, and t-11 matched pantry deposit all succeeded. This confirms the receipt/state ordering fix on real cooking return. Household full outcome did not pass: stored raw meat was exhausted, five configured search visits failed, and final home return failed unreachable. Goal needs_plan after12 tasks/one batch; no hunting or collection earned. Keep all broader pantry ACs open.

Next blocker is native departure/return near[-30.94,35.54,-0.14], rather than cooked-food receipt ordering. Audit .checks/audit-window-3KPFD5.jsonl (05:47:00–05:50:15Z) records endpoint/midpoint support rejections on wood_stack(Clone)/log (normalY around0.60), no-hit candidates, and support_step transitions from35.53 to36.09. These are investigation leads, not proof that relaxing slope/step clearance is correct. Inspect cached building geometry and supported transitions at the exit before another identical wildlife trial. No manual clearing, teleportation, fabricated resources, automatic action replay or logout.

Final fresh observation .checks/20261102T055127Z-WXwZOu.log: controller75db0ae connected/in-world,62.1/62.1HP, injurySeq0, grounded at the captured exit, no active task. Tas remained online. Automatic resupply and the successful cook-return/deposit evidence remain distinct from the failed outdoor hunt/pantry completion. Repeated full acquisition/cook/deposit/rest loops remain unearned.

**agent:claude-code/renki-plan-m3** at 2026-11-02T15:58:15Z

### Remaining for M3: serves no M3 goal

Milestone review (M3 grooming, 2026-11-02): pantry stocking serves none of M3's goals 3.1-3.5 in docs/ROADMAP.md. It belongs with its parent TKT-01M6864352 (Pressure-driven work, storage and maintenance goals) in M5, where the roadmap measures "no food burns" and assigned items reaching chests. Proposed for M5; the owner decides.

What is left: all five ACs are open; the last note says the cooking/deposit retry and full gate follow. Files to start in: `internal/goal/food_stock.go`, `internal/goal/food_work.go`, `internal/behaviour/idle.go`, `internal/session`. Verify: `just check controller test-packages ./internal/session ./internal/behaviour ./internal/goal ./internal/app`, `just check ci`; live, `just vm food-check` / `just vm resupply-food` under the VM lease (TKT-01M6EF5DTG).

**agent:claude-code/renki-plan** at 2026-11-02T16:15:35Z

Applied on 2026-11-02 by the planning pass: milestone set to M5, to match the exit goal in docs/ROADMAP.md this ticket serves. This supersedes any earlier note that only proposed the move. The owner can move it back.
