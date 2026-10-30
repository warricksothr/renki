---
schema: 4
id: TKT-01M66HZZWP4W0WVXN7TBMQ2JFM
title: Persistent goals with scripted recovery and combat testing
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - tooling
  - safety
assignees: []
milestone: M3
parent: TKT-01M62N2557R401S3E6JECSAPN4
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-goals
  branch: t3/persistent-goals
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 3d84b118ca809a31420c88027d417610290205a5
  session: null
  claimed_at: 2026-10-30T14:04:07Z
  expires_at: null
archive: null
created_at: 2026-10-30T14:04:06Z
updated_at: 2026-10-30T14:23:29Z
created_by:
  id: agent:codex/renki-goals
  name: ""
updated_by:
  id: agent:codex/renki-goals
  name: ""
extensions: {}
---

## Description

Implement the owner's explicitly requested durable achieve-this system, with documented architecture and a restricted Starlark policy above existing behavior runners/native C# routines. The initial goal is a bounded ordinary-Greyling combat test that retains its objective across routine failures and controller restarts, records recovery/progress, and requests a revised plan after exhausting recovery. Owner stop always cancels. Model-written policy revisions are designed but not automatically enabled in the first slice.

## Acceptance criteria

- [x] Goal objective, policy version, checkpoints, outcomes and budgets survive restart without replaying uncertain actions
- [x] Restricted Starlark policies compose existing runners with bounded recovery and host-verified success
- [x] Owner stop, task permissions, world changes and native safety remain authoritative
- [x] Authenticated panel shows goals, progress and needs-plan context with start/resume/stop controls
- [ ] Replay tests cover failure recovery, restart, cancellation, budgets and false success; bounded live combat goal is exercised

## Implementation plan

Implement internal/goal with an atomic durable record store, immutable policy/hash, intent-before-dispatch and receipt-before-policy checkpoints. Register its template in the existing behavior engine; add lifecycle and bounded-wait hooks so cancellation remains authoritative. Restrict Starlark to bounded decision trees (no loops, imports, allocation arithmetic or arbitrary calls), with step/time/input/output bounds. Combat policy may choose validated existing combat runner modes and waits; the host owns Greyling eligibility, scope, readiness, retry fingerprints, budgets and exact native death evidence. Add authenticated goal API/panel controls and update the supervised VM test to submit one goal, then monitor through child failures. Verify recovery, restart uncertainty, takeover, scope, storage failures and policy limits before full CI, committed review, deployment and a bounded live trial. Restart/disconnect conservatively suspend for explicit resume in this first slice; no uncertain pending action can resume.

## Notes

**agent:codex/renki-goals** at 2026-10-30T14:22:12Z

Implemented durable goals.json, immutable bundled Starlark decision policies, intent/receipt checkpoints (including Windows write-through replacement), bounded retry/wait/deadline budgets and native death proof. Existing behavior engine owns body lease; lifecycle hooks cover active and queued cancellation. Goal survives interruption without replaying uncertain actions; first slice requires explicit resume after disconnect/restart. Authenticated panel controls and a persistent-mode VM supervisor are implemented. Focused goal/app/behavior replays and supervisor checks pass, including failed encounter -> changed target/direct alternative -> exact kill receipt, owner takeover, permission denial, disk failures, false/mismatched success, stale/world/area scope and restart uncertainty. Browser fixture verified empty/offline needs-plan rendering and disabled resume while offline with no JS errors. Full CI running; live trial and actual melee evidence remain open. Automatic policy revision/model calls and broader M2 goals remain future work.
