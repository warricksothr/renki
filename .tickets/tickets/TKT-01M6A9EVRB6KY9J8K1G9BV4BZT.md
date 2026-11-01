---
schema: 4
id: TKT-01M6A9EVRB6KY9J8K1G9BV4BZT
title: Resolve a multi-threat coastal return bottleneck
type: bug
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - movement
  - plugin
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
  branch: t3/blocked-travel-observation
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: a5b1dc30c996e767f7a9906f3825a63774790073
  session: null
  claimed_at: 2026-11-01T01:17:02Z
  expires_at: null
archive: null
created_at: 2026-11-01T00:51:54Z
updated_at: 2026-11-01T05:21:21Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

The explicit work-trip return profile in PR #139 admitted a healthy, fed, dry body at[-61.67,33.37,-85.61], about96m from home[-35.9,37.66,6.53]. The native search found no eligible progress leg after3228 expansions/25626 probes, with899 hostile-corridor refusals plus water, body casts, support and obstacle refusals. It logged out without movement or injury; final67.6/67.6HP. Evidence `.checks/recovery-20261101T004838Z`, audit `.checks/audit-window-BDzhju.jsonl`. This is a bounded-search failure, not proof that no physical route exists.

Seven loaded threat envelopes were sampled: Greyling[-69.67,32.82,-47.75], radius40; Boar[-28.98,38.4,-97.8], radius24; Necks[-70.72,29.82,-107.38],[-65.85,30.69,-111.72],[-50.15,30.63,-133.66],[-107.96,30.37,-118.38], radius16; Boar[-111.52,30.22,-121.19], radius24. Guidance did not produce an active estimate; the earlier route wrapper discarded inactive graph diagnostics. PR #139 retains inactive hint/waypoint diagnostics without granting another search or movement permission.

Use the retained terrain and collider evidence to determine whether a safe route exists under current clearances, especially initially-away travel around overlapping circles near the coast, selection of four envelopes from seven threats, and the48m local range. A route may require additional loaded observations or a different bounded regional planner, but neither unknown terrain nor a geometric graph can grant movement. If a route exists, add a captured regression and bounded native progress; otherwise expose which observation or environmental change is needed. Do not retry unchanged orders, relax hostile/physical checks, move the pinned home, use cheats, or make manual creature clearing the default test procedure. Outdoor food delivery and home repair remain blocked until a verified home return.

## Acceptance criteria

- [ ] Retained threats and terrain establish route feasibility without treating geometric hints as physical clearance.
- [ ] Feasible initially-away detours make bounded checked progress, or an actionable infeasible outcome identifies the missing observation or change.
- [x] A native supervised return reaches the actual home floor without injury before further production or repair trials.

## Implementation plan

Continue above #145 /7df0cb2 on t3/checked-escape-bearings. Replace the single straight-away native escape target with a finite seven-bearing search only for hostile-avoiding flee children. A sliced physical/hostile preflight checks the first3m in half-metre steps before a bearing becomes a normal Walker goal. The preflight is guidance, never movement authority; the walker still checks the full route and live execution. A refused walker goal consumes its bearing, then tries the next without resetting the original body task or30s encounter budget. Cancel/exhaustion are terminal. Keep explicit toward-safety semantics and ordinary non-hostile flee behavior.

Align food-source admission with native horizontal exclusion radii for currently perceived hostile creatures (ordinary Neck16m, Boar24m, other/starred40m), preserving the existing smaller conservative exclusion for other non-player creatures. Native loaded observations remain authoritative beyond controller perception. Add captured shore-position/synthetic-support regressions, all-bearing refusals, per-tick/lifetime budgets, chosen-goal refusal, moving-threat reaim and cancellation; controller regressions cover captured berry/bush clearance, horizontal/wrong-height threats, safe alternative resources and unmodified isolated prey selection.

Run focused plugin/core and controller checks, full just check ci, commit and review against7df0cb2, publish/link a draft above #145. Deploy reviewed controller/plugin together with checksum verification. Use a distinct safe source or an addressed escape route under existing supervised tools; no unchanged failed trapped coastal retry, extra deadlines, lowered clearances, cheats or manual clearing. Earn actual pack gain, home-floor return and exact storage receipts separately; retain incomplete field outcomes. Preserve home and Jarl's Copper. Leave Tas online only after fresh idle safe-home verification. Original coastal AC1/AC2 remain open until evidence earns them.

## Notes

**agent:codex/renki-milestones** at 2026-11-01T01:25:05Z

Implemented stage 1 of docs/WORK-TRIP-ENCOUNTERS.md on t3/work-trip-encounters. Explicit recoverThreats requires dry + avoidHostiles on both protocol implementations. One native move_to retains task identity, destination and deadline across checked FleeTask escape, continuous quiet observation and route reconstruction. Limits are three lifetime attempts,30s each including5s quiet; escape seeks35m beyond the30m pursuit trigger. Child cancellation clears movement. Traces retain transition reason, threat and attempts. Household guard defers close threats only to a matching current-run native travel task with this opt-in; unfed bootstrap retains ordinary travel. No production receipts, combat authority or injury exemptions are manufactured.

Validation: just check ci passed (163s), retained in .checks/20261101T012158Z-lITWhC.log, including718 C# tests, controller checks and desktop/mobile panel checks. Earlier focused plugin build/lint, TravelRecoveryTests and controller goal/protocol checks also passed. Deterministic coverage includes lifetime budgets across successful resumptions, renewed pursuit, timeout, unsafe body/cancel/clock failures, protocol dependencies, current-run/task matching, stationary/stale/injury rejection and unfed bootstrap.

No live escape/resume, physical coastal feasibility, home return or outdoor food delivery is claimed. All original acceptance criteria remain unticked. Tas remains offline at the retained coastal bottleneck; do not retry the unchanged failed route solely to exercise this slice. Next validation requires physically checked route progress and then a same-task encounter followed by actual resumed progress. Stages2–4 retain stationary receipts, explicit damage/reserve budgets, graded danger planning and validated defense as follow-ups. Publication review and offline installation follow this note.

**agent:codex/renki-milestones** at 2026-11-01T01:26:53Z

Published draft PR #140, Retain travel goals through bounded native threat recovery: https://github.com/warricksothr/renki/pull/140, stacked above #139. Required committed review passed with no findings (.checks/20261101T012505Z-AwCzZ4.log). Full just check ci evidence is .checks/20261101T012158Z-lITWhC.log. All stack layers #129–#140 are linked to this thread and unwatched.

Installed implementation commit350a295 on Blight using just check vm install; manifest and installed plugin DLL checksums verified (.checks/20261101T012538Z-BuJHqF.log). The package transfer needed its built-in second attempt to release an executable lock and then completed. just check vm panel-start passed (.checks/20261101T012610Z-PFttsa.log). Subsequent just check vm status confirmed Valheim not running, controller running, bot disconnected (.checks/20261101T012631Z-BkNvAg.log). Authenticated just check vm panel GET /api/status returned version350a295 and connected:false (.checks/20261101T012632Z-mV9S2N.log).

No live join or unchanged coastal recovery was attempted. First travel-recovery slice is installed but not live-proven. This ticket remains in-progress with all physical-route/home-return criteria open; outdoor food-delivery and repair trials still wait on a verified return. Next work should establish physical coastal progress and then retain same-task escape/quiet/resume evidence, followed by stationary checkpoint and reserve-budget integration from the staged design.

**agent:codex/renki-milestones** at 2026-11-01T02:09:35Z

Reproduced the seven-threat coastal failure offline using fixture coastal-return.json, extracted from audit-window-BDzhju.jsonl (exact start/home, seven envelopes and841 measured terrain samples). Baseline four-envelope guidance is inactive (17 ring points,15 rejected hints); representing eight permits an active graph and an initially-away first leg over measured terrain. This establishes a geometric/search deficiency, not live body clearance or full route feasibility.

Implemented up to eight represented envelopes while all sampled threats constrain every edge. Native graph generation is deferred and sliced into waypoint/edge units; incomplete graphs cannot score a route. At most64 ring points,2209 build units,65 terrain callbacks per unit and4096 distinct hint cells. RouteSearch waits without physical probes during building, preserves one fallback and original physical node bounds, and has one additional finite local time stage without resetting the enclosing task deadline. Build completion and unit counts join cumulative build-time diagnostics. Physical support/water/body/wall/hostile checks remain unchanged.

Focused ThreatGuidanceTests and plugin build passed (.checks/20261101T020632Z-1BpPAJ.log, .checks/20261101T020636Z-UQUhme.log). just check ci passed in144s (.checks/20261101T020656Z-VJACiU.log). Regressions cover captured initially-away terrain leg, full eight-envelope build bounds, deferred preferred/fallback and inactive builds, no physical movement authorization while building, bridges, changed threats and bounded retry. VM remains game-offline before installation (.checks/20261101T020841Z-NBlROy.log). Next: required committed review, stacked PR above #140, install and one supervised work-trip return with retained evidence. All physical/live acceptance criteria remain unticked.

**agent:codex/renki-milestones** at 2026-11-01T02:16:34Z

PR #141 (https://github.com/warricksothr/renki/pull/141) passed committed review without findings and was installed as a5b1dc3 with checksums verified. Live work-trip return .checks/recovery-20261101T021015Z / audit-window-wmGgj6.jsonl made real checked progress from[-61.71,33.36,-85.64] to[-38.97,31.22,-44.23], about51m from home, before native no-route logout. Final66.6/66.6HP, injurySeq0. No home arrival, delivery or repair claimed. The original seven threats had changed; early physical progress used ordinary routing, so this does not live-prove the captured eight-envelope detour.

The final active incremental graph built177 units in2.87ms total; final worst measured planning slice0.55ms. An early ordinary planning slice still reached26.63ms, outside the2ms target and unrelated to deferred graph generation. The final physical frontier had only2 expansions/16 probes:8 hostile-corridor and7 water refusals, amid a Boar behind the bot and low coastal support. Earlier live movement had rejected an approach due to hostile clearance, then the dry-margin shore fallback steered toward[-37.66,31.43,-44.91]. That fallback currently commands a straight shore direction without the ordinary short-step body/hostile checks. Correct this bypass before another trial.

Continue on t3/blocked-travel-observation: preserve opted-in move_to through a bounded blocked-route observation. Replan only after observed local water/threat/indexed geometry changes and a minimum quiet wait, sharing the lifetime recovery-attempt limit and original deadline. Unchanged/unsafe/exhausted conditions remain terminal; close threats still use existing escape. Shore direction remains a hint: each dry shore half-step must pass live support, water, body and hostile checks before movement. Record exact original refusal and observed transitions. Add pure policy and checked-shore regressions, full gate and committed review, publish above #141, then one changed-build supervised recovery if readiness permits. Do not replay the unchanged failed order or relax physical/hostile margins.

**agent:codex/renki-milestones** at 2026-11-01T02:21:51Z

Implemented changed-condition route observation in the native travel policy and adapter. A refused opted-in move_to holds controls, retains original destination/body owner/deadline, and waits at least5 quiet seconds plus a changed coarse water/hostile/body/indexed-geometry observation before rebuilding the route. Observation is sampled at4Hz over64m of loaded characters. It shares three lifetime attempts and30s per attempt with escape; unchanged, unsafe or exhausted conditions remain terminal. Native transitions retain the original route refusal. No receipt/arrival/injury exemption or combat authority is invented.

Corrected the dry shore fallback: every half-metre step now passes the same live dry support/water/body/hostile probe before steering. Failure holds controls and exposes the precise shore-step refusal to the bounded observation policy. Ordinary non-dry swimming behavior is unchanged.

Focused TravelRecoveryTests, DryTravelTests and plugin build passed (.checks/20261101T021807Z-mQOZxA.log, .checks/20261101T021809Z-DMMl4a.log, .checks/20261101T021811Z-5xe6Pu.log). Full just check ci passed in141s (.checks/20261101T021914Z-F67P9o.log). Subsequent required committed review/publication/install and one supervised changed-build trial remain. Current code has no live changed-condition resume or home-arrival proof; all original acceptance criteria remain unticked.

**agent:codex/renki-milestones** at 2026-11-01T02:30:14Z

Published #142 (https://github.com/warricksothr/renki/pull/142) after clean committed review and installed6a487d0 with checksum verification. Trial .checks/recovery-20261101T022241Z / audit-window-jkeFFS.jsonl exercised three native blocked->changed->walk cycles, each with at least5s quiet and observed route changes. The same startup-return coordinator retained its objective and deadline; this is not a normal wire-task-ID proof. After initial failed searches and a checked shore-step water refusal, Tas resumed approximately88m sampled travel and reached the actual36.09m home floor. Final62.3/62.3HP, sampled injurySeq0. No sheltered release/production success claimed.

The final native logout reason was home is not safe shelter. The existing return guard treated entering its outer3m circle on an uncovered porch as immediate failure, although the actual1.5m move endpoint and planned sheltered approach had not been reached. Corrected that guard: ground+shelter+correct floor and >=12m threat clearance still establish success; a nearby threat still refuses immediately, and an unsafe actual1.5m endpoint still refuses. An uncovered outer approach/final stair can continue normally. Added six cases covering approach and endpoint safeguards; focused RecoveryReturnTests passed (.checks/20261101T022724Z-GKlW0r.log).

Full just check ci passed in the latest gate (.checks/20261101T022732Z-2bW6FT.log). Required committed review precedes updating #142 and installing the guard fix. Tas is offline on the home floor near[-37.27,36.09,9.56]. The shortest active food has164s left, so do not reuse work-trip admission requiring270s. A changed-build local return is within its original80m/120s/160m limits and existing two-food reserves; attempt that local guarded finish, then refresh personal food if safe shelter is verified. Original physical snapshot and home-release acceptance remain unearned until actual evidence; no unchanged failed-order replay.

**agent:codex/renki-milestones** at 2026-11-01T02:42:32Z

The c9c084e local return passed: .checks/recovery-20261101T023045Z / audit-window-kx0qK3.jsonl reached the actual home floor, grounded and sheltered, at[-37.11,36.09,9.25],62.2/62.2HP, injurySeq0. Native recovery released normally and Tas remained logged in. This earns AC3; the original seven-threat physical detour remains unproven.

The following personal food trial (.checks/food-20261101T023203Z / audit-window-X7kPa8.jsonl) refreshed chest1:70040 and found no requested mushrooms. Forage interaction succeeded but had no verified inventory gain. A targeting Neck at14.3m caused the old food-goal terminal threat guard; a Greyling at22.9m and Boar at21.2m were also loaded. No injury or production was claimed. The supervisor logged out. Subsequent local return .checks/recovery-20261101T023454Z / audit-window-fqLH6S.jsonl failed all8 initial hostile-corridor headings without movement/injury. Tas remains offline near[1.62,39.78,-53.16],57.4/57.4HP, without carried meals. Do not repeat unchanged recovery or request manual clearing.

User now explicitly prioritizes combat-aware gathering runs. Continue on t3/work-trip-defense above #142. Preserve completed action receipts before encounter decisions. Add an explicit permission-gated bounded native defense choice for a single confirmed ordinary Greyling, falling back to checked escape for groups, unsupported targets or insufficient readiness. Keep original travel destination, task deadline, lifetime attempts, stop/session/world semantics, and physical clearance. Personal food travel needs the same opt-in as household travel. Injury tolerance must be scoped to the exact owned encounter and a finite goal-wide damage/event budget; stationary/unowned/uncertain actions retain terminal guards. Neck/Boar/group capability is not established by eligibility alone. Live trials require readiness and an addressed route/capability issue; current trapped mixed-threat state is not a repeatable single-Greyling test.

**agent:codex/renki-milestones** at 2026-11-01T03:00:39Z

Implemented the first combat-aware food-travel slice on t3/work-trip-defense: explicit defendThreats plus both attack permission gates; a single confirmed unstarred Greyling within16m with fed/equipped/armoured80% health reserves; at most12s defense within the original30s recovery attempt. Refused/interrupted defense uses checked escape without a new body owner or reset deadline. Exact observed death, quiet observation and original-target replan remain required. Other targets/groups retain escape behavior. Added reported journey ownership so the controller emergency retreat loop does not compete.

Personal food trips now share household fed travel recovery. Completed scoped action receipts are reconciled before threat/injury guards. Exact owned travel gets one goal-wide10HP cumulative observed damage allowance,30HP/65% health and initial-health floors. Injury markers are wire sequence values, so added connection-scoped actual damage totals independent of healed HP; unknown amounts/death invalidate allowance. Baselines persist across tasks/restart, uncertain actions remain unreplayable, and stationary/unowned injury remains terminal. The supervisor mirrors ownership/budget and verified travel completion between polls.

Focused controller/session/retreat checks passed (.checks/20261101T025709Z-dVGZFI.log), native TravelRecoveryTests passed (.checks/20261101T025605Z-EMoyW0.log), and supervisor checks passed (.checks/20261101T025536Z-CQluA7.log). Full just check ci passed in160s, including the Unity plugin build and762 C# tests (.checks/20261101T025716Z-CKsUnA.log). Required committed review precedes push/publication.

No new live trial or deployment is claimed. Read-only just check vm status0 confirmed Valheim not running, controller/panel running, bot disconnected (.checks/20261101T025823Z-4NPLhl.log). Installed version remains c9c084e; last physical state/failed mixed-threat return remain as recorded above. This implementation does not establish automatic Neck/Boar/group defense, interrupted stationary-work resumption, normal wire-task encounter continuity, outdoor collection+stock delivery, or repair. Do not repeat the unchanged trapped return. AC1/AC2 remain open; earned sheltered return AC3 remains checked.

**agent:codex/renki-milestones** at 2026-11-01T03:27:55Z

Implemented single-use forage and loose-item collection continuation on t3/forage-encounter-continuation above #143 (25ac946). Native wrappers retain the original child/task/deadline and shared encounter policy, snapshot exact source identity/item/quantity and carried count, and revalidate before rebuilding only the approach. Changed source/pack, unresolved pickup use, or failed encounter returns stopped; merge substitution is disabled. Pending-use refusal precedes combat/escape inputs. Only ordinary non-aggravating pickables are admitted; alt/hold/sleep/repair/cook, chest access and tree swings remain excluded.

Fed food goals request the same bounded escape/optional Greyling defense for collection and retain their single cumulative damage baseline. Only successful owned recovery results acknowledge bounded damage. Fresh positive delayed RPC products are credited before subsequent threat/injury decisions without dispatching new work; exact run/task/source/yield checks remain. Journey ownership now includes task type, preventing same-ID stale movement metadata from suppressing retreat during another body. Controller dispatch now independently checks enabled flee/attack children against capabilities, welcome and live runtime permits. Upgrade controller and plugin together.

Focused plugin build, RecoverableWorkTests, controller goal/protocol/retreat/session checks and supervisor checks passed. Full just check ci passed in157s: .checks/20261101T032500Z-0UOR1M.log. Coverage includes held-child lifecycle, exact source/pack changes, failed encounter/pending use, stop/death/reconnect/deadman/original timeout, protocol and child permissions, stale journey task metadata, collection damage baseline, failed-result injury rejection, uncertain result refusal and delayed receipt accounting. Required committed review and publication follow.

Read-only just check vm status0 confirmed Valheim not running, controller/panel running, bot disconnected (.checks/20261101T032508Z-KCpk4r.log). Installed version still c9c084e; no field trial or deployment claimed yet. Preserve the retained mixed-threat location/readiness limits and earned AC3. Do not replay the unchanged trapped return or ask for manual clearing. AC1/AC2 and live same-task encounter-to-collection/storage proof remain open.

**agent:codex/renki-milestones** at 2026-11-01T03:29:20Z

Published draft PR #144, Retain forage and pickup work through bounded encounters: https://github.com/warricksothr/renki/pull/144, above #143. Implementation1502720 passed required committed review without findings (.checks/20261101T032757Z-i4ls3O.log); full gate passed with787 C# tests in157s (.checks/20261101T032500Z-0UOR1M.log).

Installed1502720 on Blight while Valheim was offline with just check vm install; package manifest and installed Renki.Core/Renki.Plugin DLL checksums verified (.checks/20261101T032826Z-haRPEx.log). Restored controller/panel with just check vm panel-start (.checks/20261101T032846Z-uYUuwr.log). Read-only just check vm status0 confirmed game not running, controller running, bot disconnected (.checks/20261101T032900Z-dGnQU6.log). Authenticated just check vm panel GET /api/status confirmed version1502720 and connected:false (.checks/20261101T032900Z-qy0ILT.log). No join, combat or collection/storage field success is claimed.

Verified linked stack #129 through #144 and all layers unwatched. Ticket remains in-progress: earned sheltered-home AC3 stays checked; original coastal route AC1/AC2 and live normal-task recovery-to-collection/storage proof remain open. Tas stays at the retained mixed-threat location without carried meals. Next field work needs a physically checked exit and food readiness, or validated mixed-threat capability; do not replay the unchanged return, relax physical margins or ask for routine manual clearing. Cooking, pending chest access and tree-swing encounter recovery still require separate transactional design.

**agent:codex/renki-milestones** at 2026-11-01T03:55:31Z

User explicitly requested deploy/start and live testing. Verified the already-installed reviewed implementation1502720 (only later ticket-record commits differ), started Tas with just check vm start2, and used existing supervised recipes. No production code or safety setting changed. Native combat was already enabled on Blight; no combat switch was changed.

Live evidence: .checks/audit-window-QhlFlM.jsonl (summary .checks/20261101T035341Z-PEm4yF.log) covers03:39:40–03:53:30UTC. The first food-check (.checks/food-20261101T034151Z) retained normal wire taskt-1 through blocked→changed→walk, reached the actual36.09m home floor, withdrew3 Mushroom from1:70031 and ate1, all without injury. The production goal succeeded; the requested outdoor-acquisition predicate was absent because chest resupply was enough, so the test supervisor correctly reported a failed acquisition proof and logged out. This was a test setup mismatch, not failed resupply.

A staged shortage trial (.checks/food-20261101T034506Z) normally stored2 carried mushrooms in excluded chest1:70031 and searched using1:70040. Search taskt-3 retained its ID through two blocked/changed-condition cycles, then hit its original90s deadline before collection. Last sampled position[-32.36,34.17,23.18],45.2/45.2HP, injurySeq0. Native route candidates repeatedly failed live validation near[-41.16,35.29,13.28] and[-25.84,34.54,20.72]; local searches reported support_step/body_cast and some hostile-corridor refusals. The timeout does not prove no route exists, and raising it alone would not resolve those physical rejections. No collection, storage delivery or combat proof earned.

Restoration trial just check vm food-check --join --reserve2 --keep-online passed (.checks/food-20261101T034826Z / .checks/20261101T034826Z-i9CFx0.log), returning from the south approach and resupplying while leaving Tas online. A later ordinary flint interaction with recovery enabled (t-4, source1:69835) safely refused with unreachable after live navmesh/local body-clearance rejection; Tas remained home. A controlled normal drop of1 Wood (t-5) produced exact drop2008100581:219 and pack35→34. Recovery-enabled pickup t-6 passed with took1 Wood (.checks/task-in0jsm6k.json / .checks/20261101T035310Z-OM4Jy0.log); fresh pack returned to35. This is positive opted-in pickup without a creature encounter, not encounter-resumption proof. All staged Wood recovered; original mushrooms are accounted for by resupply/storage.

Fresh final status/map/pack (.checks/20261101T035339Z-ThmQzj.log, -wxFyMi.log, -fkSNvP.log) confirmed connected in DizzysHellheim,1502720,[-34.62,36.09,7.86],37.7/37.7HP, injurySeq0, grounded+sheltered+near fire, no body task, no observed creatures, Wood35 and Mushroom2. Health maximum fell with food expiry; there were no damaged events in the retained audit. Tas remains logged in, not stopped by test cleanup. Prior gate/review still applies to unchanged production code; validate only new ticket/docs evidence before committing it.

Next focus is reliable checked departure/stair traversal around the raised house and actionable per-candidate physical refusal diagnostics, followed by outdoor forage/return/storage and a real same-task encounter. Preserve clearance, original task deadlines and source/receipt rules; do not claim live Greyling/Neck/Boar/group defense or original coastal AC1/AC2. Earned AC3 stays checked.

**agent:codex/renki-milestones** at 2026-11-01T04:14:27Z

Implemented navmesh-only support projection on t3/checked-stair-projection above #144 /3bf4b14. Raw navmesh height no longer vetoes a successful continuously checked physical step; recorded trails keep their original65cm floor agreement. Support/body/wall/water/hostile probes, measured goal arrival, original512-sample budget and task deadline remain unchanged. Rejection traces now include checkingFrom, supported, validationRejection and maxHintHeightOffset; accepted candidates retain maxHintHeightOffset.

Focused just check plugin test-core Navmesh, just check plugin build and just check plugin fmt passed (.checks/20261101T041120Z-XXTgK3.log, -CiRMCu.log and .checks/20261101T041147Z-58j4SM.log). Full gate is running in .checks/20261101T041210Z-hKXErF.log; no field success is claimed. Synthetic regressions cover smoothed stairs, recorded-floor refusal, wrong-storey arrival, physical refusal and original sample bounds. Live diagnosis must establish whether the previously silent rejection was height mismatch; a further real physical refusal needs its own diagnosis. Original coastal AC1/AC2 remain open.

**agent:codex/renki-milestones** at 2026-11-01T04:16:07Z

Published/link draft PR #145, Project navmesh stair hints onto checked physical support: https://github.com/warricksothr/renki/pull/145, above #144. Implementation59a9ba2 passed full just check ci in143s (.checks/20261101T041210Z-hKXErF.log) and required committed just check review 3bf4b14..HEAD with no findings (.checks/20261101T041447Z-Vj0w8R.log).

Installed59a9ba2 using just check vm install (.checks/20261101T041519Z-UXNBIl.log), which stopped the client only for native DLL installation and verified package plus installed DLL checksums. Restored panel with just check vm panel-start (.checks/20261101T041543Z-nB4Dlm.log). Supervised normal food resupply/join is running in .checks/food-20261101T041549Z before new outdoor work; no field route result yet. Prior read-only state remained sheltered, idle and injurySeq0 as its Mushroom expired. Preserve honest separation of route, acquisition, storage and encounter proofs.

**agent:codex/renki-milestones** at 2026-11-01T04:27:16Z

Field deployment59a9ba2: normal resupply/join passed and left Tas online (.checks/food-20261101T041549Z). Outdoor household Raspberry=1 trial scoped to1:70040 and marker berries required both new acquisition and stock deposit (.checks/food-20261101T041655Z). It departed the actual home floor and travelled roughly115m without injury, but the patch was occupied by two Greydwarfs plus a Greyling. Search taskt-4 retained its ID through three blocked/changed cycles and exhausted that budget. A subsequently observed exact bush admitted collectiont-5; an engaged Greydwarf triggered escape. Straight-away escape made checked progress west until water blocked the remaining route; no collection or deposit receipt. Supervisor logged out uninjured near[-27.46,30.83,-20.59].

Audit .checks/audit-window-4VoNvn.jsonl / summary .checks/20261101T042000Z-cSbCCq.log explicitly proves a checked navmesh prefix with maxHintHeightOffset0.76894m was accepted and followed before a later hostile-corridor refusal. All observed subsequent candidate rejections report physical_probe with actual hostile/water reasons, rather than a silent speculative-floor rejection. This supports the projection change, not universal house-route success.

The dedicated recover-home helper was an unsuitable setup: its native admission requires HP50 and two active meals; Tas had31.4/31.4 and one active meal. It correctly refused the join. The helper eventually entered cleanup after join failure; an attempted interrupt landed in cleanup, so just check vm recovery-return clear was explicitly run and passed (.checks/20261101T042314Z-96bppT.log). No recovery proof claimed for .checks/recovery-20261101T042052Z.

Ordinary supervised food-return from the shore (.checks/food-20261101T042319Z) safely progressed around the west side to the southern approach[-44.02,34.18,16.41], roughly13m from home, but timed out on its original120s deadline; it remained uninjured and logged out. Audit .checks/audit-window-cAoA66.jsonl / summary .checks/20261101T042639Z-4rCIvh.log shows multiple planned routes that physically stalled on sloped support and were refused/replanned. No time limit or clearance changed. Next recovery starts from this materially closer, higher observed approach, rather than replaying the old trapped coastal path. Once sheltered, prefer a different source marker over the occupied berry patch. Outdoor collection/storage and same-task encounter continuation remain unproven; original coastal AC1/AC2 stay open.

**agent:codex/renki-milestones** at 2026-11-01T04:35:51Z

Restoration from the closer southern approach passed with just check vm food-check --join --reserve2 --keep-online (.checks/food-20261101T042715Z): native return reached[-36.38,36.09,7.88], sheltered and uninjured, then left Tas online. A normal chest withdrawal moved1 MushroomYellow but found no Mushroom; the partial invalid_target receipt is retained (.checks/task-6mn1uk8i.json). Yellow Mushroom was actually eaten; subsequent fresh status showed41.6/41.6HP with Raspberry+MushroomYellow active, injurySeq0 and idle shelter.

A distinct southern household Mushroom=1 acquisition+deposit trial (.checks/food-20261101T042930Z) travelled toward south mushrooms but refused an occupied shore corridor. Retained audit .checks/audit-window-szCF9s.jsonl / summary .checks/20261101T043255Z-auj6th.log shows Boar[17.21,31.02,50.22], plus Greyling/Neck envelopes and water/support/body refusals. Source searcht-8 exhausted its bounded changed-condition attempts; automatic returnt-9 succeeded on the actual home floor. The goal correctly ended needs_plan for exhausted/unsafe source, no collection or deposit receipt, and the supervisor kept Tas online only after fresh safe-home proof. Roughly159m observed travel with injurySeq0. This is positive failed-source-to-safe-return evidence, not household production success.

A fresh nearby Branch1:86293 was explicitly interacted with using recovery+defense flags (t-11, .checks/task-vtkmjff0.json), succeeding used the Branch in30s. Fresh pack stayed Wood35, so no collection was credited. Food return (.checks/food-20261101T043412Z) safely restored home and kept Tas online. Fresh perception then identified exact branch product991629321:314, Wood1 at[-12.25,32.53,26.56]; a separate exact normal pickup is running. Do not count the interaction as pickup or deposit older carried Wood as acquisition proof. Jarl's dropped Copper remains untouched.

**agent:codex/renki-milestones** at 2026-11-01T04:37:48Z

Exact branch-product pickup991629321:314 (t-14) refused before acquisition after43s (.checks/task-n3btyjl8.json / .checks/20261101T043549Z-NXrSs1.log). Its retained diagnostics show checked departure through the changed wood gate, followed by a Greyling647877508:283 at[8.3,33.27,-6.07] whose40m exclusion blocks the last approach from[-22.65,33.26,24.7] to[-12.25,32.53,26.56]. Boar/Neck envelopes and water/body/support restrictions also constrained search. Pack Wood was35 before and after branch use; no pickup gain and no storage receipt are claimed. No older carried Wood was deposited to manufacture proof.

Ordinary supervised return/resupply .checks/food-20261101T043701Z passed in20s and kept Tas online after fresh safe-home verification. The nearby-branch outdoor interaction and multiple actual-floor returns are positive results; both household production trials and the exact outside pickup are honestly incomplete. No injury was observed in any field trial. Do not request routine manual clearing, lower hostile/physical margins, increase deadlines as a substitute for a route fix, change home or pick up Jarl's Copper.

Next implementation priorities: checked alternate escape bearings when the straight-away goal reaches shore; source admission/selection that accounts for loaded hostile envelopes, rather than starting a new collection merely because a remembered/live source is outside the smaller immediate-threat radius; continued measured execution agreement on house slopes and gates. Preserve original encounter and task budgets, exact source/pack revalidation, cancellation and transaction rules. Retain original coastal AC1/AC2 as open and earned home-floor AC3 as checked. Collection-to-storage and real encounter-to-resumed-work need fresh positive receipts before being marked complete.

**agent:codex/renki-milestones** at 2026-11-01T04:59:05Z

Implemented on t3/checked-escape-bearings above #145 /7df0cb2. Hostile-avoiding native flee now uses seven lifetime bearings (0,+30,-30,+60,-60,+90,-90), each admitted only after a sliced3m physical+loaded-hostile prefix (half-metre steps, eight probes/tick, at most42 in one flee child). Prefixes only choose a normal Walker goal; all full-route and live execution checks remain. Refused goals consume their bearing, reset only the walker's local stuck detector, and never renew the outer task or30s encounter deadline. Cancellation/exhaustion are terminal. Tangent goals stay anchored while the bot moves; only a materially moved threat re-aims them, avoiding circular pursuit. Existing toward-safety and non-hostile flee behavior remain.

Controller source admission now matches native horizontal exclusion radii for visible hostiles: ordinary Neck16m, Boar24m, other/starred40m. Players, exact isolated prey exclusion and the existing nearby-creature/pursuit rules remain. This prevents the captured berry search from immediately selecting a bush only16m from a Greydwarf. It does not expose threats outside controller perception or establish a world route.

Focused just check controller test-packages ./internal/goal passed (.checks/20261101T045647Z-1RqnZW.log); just check plugin build passed (.checks/20261101T045720Z-XFhw7x.log). FleeBearings regressions cover captured positions with explicitly synthetic shore support, multiple-threat monotonic separation, physical/exhausted goals, per-tick/lifetime bounds, actual support projection, cancellation, moving-threat/fixed-tangent semantics and unchanged enclosing encounter time. Latest focused check .checks/20261101T045836Z-D4Qdy4.log passed. Native field proof remains pending; Tas was last read sheltered and idle at home with injurySeq0. No original coastal AC or collection/storage proof is newly earned.

**agent:codex/renki-milestones** at 2026-11-01T05:03:45Z

Published/link draft PR #146, Try checked escape bearings and avoid occupied food sources: https://github.com/warricksothr/renki/pull/146, stacked above #145. Implementation9263f4b passed just check ci in153s, including804 C# tests plus controller/panel/tooling (.checks/20261101T045917Z-mhH65W.log), and required committed just check review 7df0cb2..HEAD without findings (.checks/20261101T050216Z-rvEpX3.log). Selected-bearing traces include physics-cast counts; exhausted preflights retain physical failure examples and a bounded loaded-hostile snapshot.

Normal home resupply on the prior implementation passed (.checks/food-20261101T045918Z), leaving Tas online sheltered with Blueberries+MushroomYellow and personal reserve. Installed reviewed9263f4b using just check vm install (.checks/20261101T050254Z-0tWoiJ.log); controller/package/installed DLL checksums verified. Native restart is required for this plugin change. Restoring panel and preparing a distinct remembered northern mushroom site for a supervised production trial. No native alternative-bearing, resumed-collection or storage result is claimed yet; original coastal field criteria remain open.

**agent:codex/renki-milestones** at 2026-11-01T05:09:35Z

Live trial .checks/food-20261101T050359Z on9263f4b collected1 fresh Mushroom (interact t-3, exact pickup t-4, observed_yield_fresh_inventory) and ate it as a third active food. Increased food maximum correctly committed a home return. No deposit was earned. Return t-6 failed unreachable after3 native blocked-observation attempts: no checked dry shore step: water. No injury, minimum39.9HP; final grounded/out-of-water[-52.29,30.75,10.95],53.5/55.7HP after keeper refresh. Supervisor safely logged out/stopped game, restored panel. Audit .checks/audit-window-tDRhBD.jsonl; outer .checks/20261101T050359Z-mO6sIm.log.

Diagnosis: Swim promotes a dry grounded support below the conservative0.8m planning buffer into swimming recovery, before route planning/replay. It discards a physically checked uphill route for a distant geometric shore hint, then rejects the short shore step under the same0.8m rule. Current feet and actual inWater/swimming state must be distinguished from the safety buffer required for future movement. Next change removes that synthetic swim transition for an out-of-water body; actual in-water/swimming recovery and every existing0.8m midpoint/endpoint, support, body and hostile execution check remain. Add a captured-position/synthetic-bank regression proving only a fully checked uphill departure, changed-water refusal and unchanged physical/hostile refusals. Enable replay diagnostics and include water/body state in shore refusal evidence. Run focused maths, full gate, committed review, then deploy and test a normal supervised return from the saved position. No unchanged production retry or credit for collected food consumed personally.

**agent:codex/renki-milestones** at 2026-11-01T05:21:21Z

Grounded-route correction8b03f50 passed focused just check plugin test-core DryTravelTests (.checks/20261101T051009Z-BXmU2K.log), complete just check ci in143s with805 C# tests (.checks/20261101T051048Z-esGGwy.log), and committed just check review 7df0cb2..HEAD without findings (.checks/20261101T051325Z-rjlqXi.log). Pushed to PR #146; deployed with installed/controller/package checksum verification (.checks/20261101T051339Z-Vr5ZlW.log). Panel restored.

Positive native check: ordinary just vm food-check --join --reserve2 --keep-online (.checks/food-20261101T051404Z) returned from saved shore[-52.29,30.75,10.95] onto actual sheltered home floor[-36.4,36.09,7.89]. Observed travel29.8m; final57/57HP, no injury, idle; supervisor kept online. Actual in-water/swimming recovery remains intact, and no forward water margin was lowered. This is a single checked return, not general routing proof.

Follow-up household trial .checks/food-20261101T051602Z failed without injury. Search t-6 repeatedly encountered the west gate and descending support-step refusals; a later checked local route reached marginal shore[-52.47,30.71,10.94]. Native search and automatic return t-7 exhausted bounded recovery with all8 first-neighbor probes refused for water (1 expanded node, not a large-search-budget problem). This is distinct from the corrected synthetic-swim diversion. No fresh collection or stock deposit occurred. Supervisor logged out/stopped game safely and restored the panel. Combined audit .checks/audit-window-uSjpS9.jsonl; .checks/20261101T051925Z-pmpDRA.log. Inter-trial MushroomYellow storage was older personal cargo and is not outdoor-production proof.

Remaining: avoid choosing marginal shoreline approaches and repeated gate/slope failures; retain failed live approach evidence across replans, and expose numeric support/water clearance to distinguish a real boundary from a wrong-floor sample. A bounded physically dry uphill egress may need separate design for a body already below the conservative planning buffer; do not lower global water/hostile margins or enlarge timeouts to force this test. Do not repeat the unchanged northern-source/shore return. Tas is safely offline at the captured shore, not currently at home. Original coastal AC1/AC2, native alternative-bearing/resumed-collection and exact outdoor-to-chest delivery remain open. PR #146 stays draft.
