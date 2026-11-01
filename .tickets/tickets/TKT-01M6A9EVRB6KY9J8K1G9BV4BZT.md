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
  branch: t3/work-trip-encounters
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 91c8899ca0a6f064e21e1931c2751e6ef119ca98
  session: null
  claimed_at: 2026-11-01T01:17:02Z
  expires_at: null
archive: null
created_at: 2026-11-01T00:51:54Z
updated_at: 2026-11-01T01:25:05Z
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

The user's request to plan and start seamless threat handling authorizes this follow-up. Implement in stages: (1) one native travel task retains its goal across bounded escape/quiet/replan cycles; (2) stationary work can checkpoint and yield to the same encounter policy, with pressure-based recovery and tolerable injury budgets; (3) route risk becomes graded according to capability, exposure and escape options while physics remains hard; (4) extend validated defensive combat to ordinary Necks/Boars, then integrate it under the same body owner. Preserve human stop, world/connection ownership and receipt-based production proof throughout.

First implementation: explicit recoverThreats on dry hostile-avoiding move_to. Reuse native FleeTask with live hostile-clearance checks, no combat permission, three escape attempts,30s per attempt and5s quiet observation before reconstructing the original route. No deadline reset; exhausted/unsafe recovery returns a precise failure. The controller's household guard permits this handling only for a current-run matching travel task; distant targeting alone no longer terminates household goals. Close threats during stationary work, actual injury and existing critical footing/health checks remain terminal in this first stage. No engine stop/reacquire or replay of pickup/deposit/cooking. Retain phases, reasons, threat identity and attempt counts in traces.

Add deterministic Core transition tests, protocol validation on both sides, goal ownership regressions and native build checks. Document the complete staged design and first-slice limits. Run focused checks, full just check ci and committed review, publish above PR #139 and install safely with Tas offline. The multi-threat coast's physical feasibility and actual home return remain unearned acceptance criteria; do not retry the unchanged trapped position merely to claim live encounter coverage. Continue route-feasibility work using retained evidence separately from changing hard collision or hostile clearances.

## Notes

**agent:codex/renki-milestones** at 2026-11-01T01:25:05Z

Implemented stage 1 of docs/WORK-TRIP-ENCOUNTERS.md on t3/work-trip-encounters. Explicit recoverThreats requires dry + avoidHostiles on both protocol implementations. One native move_to retains task identity, destination and deadline across checked FleeTask escape, continuous quiet observation and route reconstruction. Limits are three lifetime attempts,30s each including5s quiet; escape seeks35m beyond the30m pursuit trigger. Child cancellation clears movement. Traces retain transition reason, threat and attempts. Household guard defers close threats only to a matching current-run native travel task with this opt-in; unfed bootstrap retains ordinary travel. No production receipts, combat authority or injury exemptions are manufactured.

Validation: just check ci passed (163s), retained in .checks/20261101T012158Z-lITWhC.log, including718 C# tests, controller checks and desktop/mobile panel checks. Earlier focused plugin build/lint, TravelRecoveryTests and controller goal/protocol checks also passed. Deterministic coverage includes lifetime budgets across successful resumptions, renewed pursuit, timeout, unsafe body/cancel/clock failures, protocol dependencies, current-run/task matching, stationary/stale/injury rejection and unfed bootstrap.

No live escape/resume, physical coastal feasibility, home return or outdoor food delivery is claimed. All original acceptance criteria remain unticked. Tas remains offline at the retained coastal bottleneck; do not retry the unchanged failed route solely to exercise this slice. Next validation requires physically checked route progress and then a same-task encounter followed by actual resumed progress. Stages2–4 retain stationary receipts, explicit damage/reserve budgets, graded danger planning and validated defense as follow-ups. Publication review and offline installation follow this note.
