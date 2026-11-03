---
schema: 4
id: TKT-01M6H1A2XPHCAR4F016AE4F8K0
title: End the body task when the final portal guard refuses movement
type: bug
status: done
status_reason: null
priority: high
due_on: null
labels:
  - plugin
  - movement
  - safety
  - live
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-03T15:44:10Z
updated_at: 2026-11-03T15:52:41Z
created_by:
  id: agent:codex/home-survey-delivery
  name: ""
updated_by:
  id: agent:codex/home-survey-delivery
  name: ""
extensions: {}
---

## Description

Automatic Loot pickup t-38 approached Resin near the new-home darkforest portal on2026-11-03. The final native guard refused portal_cast at15:36:24.061Z near[-187,30.97,-459.99], but the pickup task remained running. It later entered stuck Jump at15:36:27 and Strafe at15:36:30, shifted nearer the trigger and teleported at15:36:32 to[414.46,32.64,-1175.96]. Loot then failed target_lost. Exact retained audit .checks/audit-window-FfbfFj.jsonl. This happened after the owner supplied Raspberry and Tas recovered; it was not a survey leg or an intentional portal order.

Driver currently clears this frame's movement/jump/run/dodge when PortalClear fails, without ending the owning body task. Make this physical trigger refusal terminal for that body task in the same physics step. Cancel it, idle and clear all inputs, report one unreachable receipt, and never let subsequent task ticks enter stuck recovery. Preserve native collision/portal checks and ordinary owner/task replacement contracts; no scoped portal allowance, trigger bypass or controlled portal travel is introduced.

Add core lifecycle regressions covering exactly one final receipt, no further ticks after refusal, contained throwing cancellation, and no spurious idle-body receipt. Build real native DLL, run focused lifecycle and full just check ci, commit/review/deploy under the lease, then retain a short known-portal probe showing task failure on the first guard rejection, no later stuck escalation/teleport, and measured displacement. Do not re-enable automatic Loot or intentionally reproduce the earlier stuck escalation. Record actual final location; the existing guard blocks a deliberate portal return and controlled traversal remains future work.

## Acceptance criteria

- [x] A final native portal movement refusal cancels the owning body task and reports one unreachable receipt in the same step.
- [x] No further ticks, jump or strafe recovery can follow that refusal; cancellation faults and an idle runner remain contained.
- [x] Focused lifecycle regressions, the real plugin build, full just check ci and committed review pass.
- [x] A native approach ends with one immediate unreachable receipt on its first final guard rejection and stays outside the portal during settling; retain actual displacement and final location.

## Implementation plan

Wire the final native portal refusal to TaskRunner.RefuseMovement after task ticking but before controls are submitted. Driver stands immediately; runner releases the active body task and reports one unreachable outcome, containing cancellation faults. Preserve death/session/ordinary replacement behavior. Add lifecycle regressions proving no further recovery ticks and no duplicate/idle receipts. Run focused core checks, real DLL and full gate, commit/review/deploy, then supervise a short already-blocked portal approach without re-enabling Loot. Record current remote location and inability to use controlled portal return as a separate limit.

## Notes

**agent:codex/home-survey-delivery** at 2026-11-03T15:47:18Z

Grooming before native verification: an approach may physically move up to the portal clearance before its first guard rejection. Criterion4 measures immediate task termination at that rejection and no subsequent stuck escalation/teleport, with actual displacement retained, rather than requiring every probe to start at exactly the already-blocked coordinate. The current remote body has settled slightly away from that earlier coordinate. No native pass is claimed yet.

**agent:codex/home-survey-delivery** at 2026-11-03T15:52:41Z

Implemented and deployed cc5f323. Driver now stands and calls TaskRunner.RefuseMovement when the final physical portal guard rejects the proposed input. The runner cancels/releases the active body and emits one unreachable final receipt; later physics ticks cannot reach that task's jump/strafe recovery. No portal geometry or entry allowance was relaxed. Core lifecycle regressions cover later ticks, duplicate refusal, cancellation faults, no body and independent later orders. just check plugin test-core TaskRunnerTests passed6s in .checks/20261103T154448Z-NGaaHC.log; just check fmt passed7s in .checks/20261103T154528Z-UM9pVX.log; full just check ci passed167s in .checks/20261103T154547Z-hCXJML.log; committed just check review passed7s without findings in .checks/20261103T154846Z-pX0e2G.log; checksum-verified just check vm deploy passed91s in .checks/20261103T154925Z-Y54sUK.log.

Native changed check .checks/task-gcwg1_fw.json: move_to t-1 accepted/running at15:51:41.847Z; the first portal_cast guard refusal was15:51:42.0975404Z at[414.29,31.65,-1176.12]; its single failed/unreachable receipt was15:51:42.0981151Z, in the same physics step, with ordinary movement refused a portal trigger; task stopped before stuck recovery. This is the expected negative probe outcome, not a failed feature check. Baseline .checks/new-home-portal-fix-baseline.json was[413.58,31.65,-1175.97]. Post-settling .checks/new-home-portal-fix-after.json is[414.42,31.65,-1176.15], grounded on structure, idle,25/25HP, no food, injurySeq0 in this restarted client. Measured progress before refusal and small physical settling are retained; do not call it zero displacement. There was no subsequent jump/strafe recovery or teleport. Loot stayed persistently disabled after deployment, confirmed in .checks/new-home-portal-fix-behaviours.json.

The owner supplied Raspberry after the earlier storage injury; Tas recovered. Automatic timer Loot then chased Resin and accidentally entered darkforest during the old guard's still-running recovery. Final actual location is the remote portal platform, not the new-home house. Known source portal1:161078 links to home-side1:161081, but intentional portal traversal/return remains unsupported and was not bypassed. No bed was claimed and no survey ran. Keep Loot disabled until the incident/current owner intent is reviewed. Other tasks' incident notes and docs/HANDOFF.md retain the distinction between earned earlier home arrival and current remote location.

## Summary

Delivered/deployed cc5f323: final portal input refusal cancels the body task and reports one same-step unreachable receipt, preventing later stuck jump/strafe escalation. Core lifecycle, full gate, committed review, real deployment and native first-refusal evidence pass. Loot remains disabled; Tas is healthy at the remote portal. Controlled return, a free bed and full home survey are separate pending work.
