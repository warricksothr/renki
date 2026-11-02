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
  branch: t3/coastal-route-guidance
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: aacdd5abbefc6168dcfcd5c59b481f859bfbcee6
  session: null
  claimed_at: 2026-11-01T01:17:02Z
  expires_at: null
archive: null
created_at: 2026-11-01T00:51:54Z
updated_at: 2026-11-01T02:09:35Z
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

Continue the staged encounter design in docs/WORK-TRIP-ENCOUNTERS.md. Stage1 (native same-task escape/quiet/replan) is implemented in parent PR #140 and installed with Tas offline; stationary receipts/reserve budgets, graded exposure and validated defense remain subsequent slices.

This branch targets the physical return prerequisite. Extract a compact regression fixture from retained audit-window-BDzhju.jsonl: the exact seven threat envelopes, start/home and measured terrain, with provenance. Reproduce why four-envelope geometric guidance is inactive. Check whether bounded representation of the surrounding threats can expose an initially-away dry detour; treat missing samples and terrain interpolation as hints, never as movement authority. Keep all physical support, water, body, floor and hostile-clearance checks unchanged. Preserve a bounded search and frame work; do not simply expand an expensive synchronous graph.

Implement the smallest demonstrated routing correction with regressions for the captured bottleneck, changing threats, unsupported terrain/bridges, node/time budgets and multi-leg progress. Record explicitly what offline terrain can and cannot establish. Run focused plugin checks, full just check ci, commit/review and publish above #140. Install only after verification. A supervised live work-trip recovery is warranted only after a new tested planner addresses the retained failure; capture exact checked movement and actual floor/shelter arrival or a precise new refusal. Never repeat an unchanged failed order, enable combat/cheats, relax collision/hostile clearances or require manual creature clearing. Outdoor delivery and repair follow only verified home recovery. Keep acceptance criteria unticked until earned.

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
