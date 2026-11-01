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
updated_at: 2026-11-01T02:21:51Z
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
- [ ] A native supervised return reaches the actual home floor without injury before further production or repair trials.

## Implementation plan

Continue docs/WORK-TRIP-ENCOUNTERS.md. Parent PR #140 implements same-task escape/quiet/replan, and #141 adds eight-envelope incremental geometric guidance. The a5b1dc3 field return progressed from the previous coastal position but stopped51m from home at a new water/Boar pinch point, without injury. Evidence and limitations are retained in notes. Stationary receipts/reserve budgets, graded exposure and validated defense remain later stages.

Current slice: let opted-in move_to observe a refused route for changed conditions before giving up. Share the same three lifetime attempts,30s attempt timeout and original enclosing deadline with escape. Require at least5 quiet seconds and changed quantized local water/hostile/body/indexed-geometry observations before requesting another full physical search. No unchanged blind retries, fabricated arrival, injury exemption or new body owner. Unsafe state, cancellation, deadline and budget exhaustion remain terminal. Preserve the original route refusal and mode transitions in traces.

Correct the dry shore fallback discovered in the latest trace: a distant shore target remains only a geometric hint, and every half-metre step must pass the same live support, water, body and hostile probe as ordinary dry travel. On refusal hold controls and expose a precise route failure to the bounded observation policy. Preserve the existing non-dry swimming behavior.

Add focused pure transition and captured checked-shore regressions; run plugin build/focused checks, just check ci and committed review. Publish above #141 and install safely. Attempt one guarded changed-build recovery if admission permits, keeping physical/hard margins and no-injury guards. Keep exact native/audit evidence, do not repeat an unchanged failed order and do not require manual creature clearing. Original route feasibility, live home return, outdoor delivery and repair remain unearned until demonstrated.

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
