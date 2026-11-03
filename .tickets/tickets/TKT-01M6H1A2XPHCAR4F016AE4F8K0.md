---
schema: 4
id: TKT-01M6H1A2XPHCAR4F016AE4F8K0
title: End the body task when the final portal guard refuses movement
type: bug
status: in-progress
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
claim:
  actor: agent:codex/home-survey-delivery
  branch: t3/map-home-location
  worktree: /home/sothr/.t3/worktrees/renki/t3-b02dd361
  commit: 8f662f66df84c9794137fee2fc1fb5a0c4586de7
  session: null
  claimed_at: 2026-11-03T15:44:22Z
  expires_at: null
archive: null
created_at: 2026-11-03T15:44:10Z
updated_at: 2026-11-03T15:48:32Z
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

- [ ] A final native portal movement refusal cancels the owning body task and reports one unreachable receipt in the same step.
- [ ] No further ticks, jump or strafe recovery can follow that refusal; cancellation faults and an idle runner remain contained.
- [ ] Focused lifecycle regressions, the real plugin build, full just check ci and committed review pass.
- [ ] A native approach ends with one immediate unreachable receipt on its first final guard rejection and stays outside the portal during settling; retain actual displacement and final location.

## Implementation plan

Wire the final native portal refusal to TaskRunner.RefuseMovement after task ticking but before controls are submitted. Driver stands immediately; runner releases the active body task and reports one unreachable outcome, containing cancellation faults. Preserve death/session/ordinary replacement behavior. Add lifecycle regressions proving no further recovery ticks and no duplicate/idle receipts. Run focused core checks, real DLL and full gate, commit/review/deploy, then supervise a short already-blocked portal approach without re-enabling Loot. Record current remote location and inability to use controlled portal return as a separate limit.

## Notes

**agent:codex/home-survey-delivery** at 2026-11-03T15:47:18Z

Grooming before native verification: an approach may physically move up to the portal clearance before its first guard rejection. Criterion4 measures immediate task termination at that rejection and no subsequent stuck escalation/teleport, with actual displacement retained, rather than requiring every probe to start at exactly the already-blocked coordinate. The current remote body has settled slightly away from that earlier coordinate. No native pass is claimed yet.
