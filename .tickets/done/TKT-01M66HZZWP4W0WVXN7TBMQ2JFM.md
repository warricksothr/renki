---
schema: 4
id: TKT-01M66HZZWP4W0WVXN7TBMQ2JFM
title: Persistent goals with scripted recovery and combat testing
type: task
status: done
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
claim: null
archive: null
created_at: 2026-10-30T14:04:06Z
updated_at: 2026-10-30T14:33:31Z
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
- [x] Replay tests cover failure recovery, restart, cancellation, budgets and false success; bounded live combat goal is exercised

## Implementation plan

Implement internal/goal with an atomic durable record store, immutable policy/hash, intent-before-dispatch and receipt-before-policy checkpoints. Register its template in the existing behavior engine; add lifecycle and bounded-wait hooks so cancellation remains authoritative. Restrict Starlark to bounded decision trees (no loops, imports, allocation arithmetic or arbitrary calls), with step/time/input/output bounds. Combat policy may choose validated existing combat runner modes and waits; the host owns Greyling eligibility, scope, readiness, retry fingerprints, budgets and exact native death evidence. Add authenticated goal API/panel controls and update the supervised VM test to submit one goal, then monitor through child failures. Verify recovery, restart uncertainty, takeover, scope, storage failures and policy limits before full CI, committed review, deployment and a bounded live trial. Restart/disconnect conservatively suspend for explicit resume in this first slice; no uncertain pending action can resume.

## Notes

**agent:codex/renki-goals** at 2026-10-30T14:22:12Z

Implemented durable goals.json, immutable bundled Starlark decision policies, intent/receipt checkpoints (including Windows write-through replacement), bounded retry/wait/deadline budgets and native death proof. Existing behavior engine owns body lease; lifecycle hooks cover active and queued cancellation. Goal survives interruption without replaying uncertain actions; first slice requires explicit resume after disconnect/restart. Authenticated panel controls and a persistent-mode VM supervisor are implemented. Focused goal/app/behavior replays and supervisor checks pass, including failed encounter -> changed target/direct alternative -> exact kill receipt, owner takeover, permission denial, disk failures, false/mismatched success, stale/world/area scope and restart uncertainty. Browser fixture verified empty/offline needs-plan rendering and disabled resume while offline with no JS errors. Full CI running; live trial and actual melee evidence remain open. Automatic policy revision/model calls and broader M2 goals remain future work.

**agent:codex/renki-goals** at 2026-10-30T14:33:31Z

Live bounded goal 13e472d7a93327c0e03aad39 exercised on reviewed/deployed 4f09e16 (PR #72 merged as 6b79dce). Evidence: .checks/combat-trial-20261030T142652052738Z/summary.json. Goal persisted and waited 120 seconds, then needs_plan: observation/recovery wait budget exhausted. Zero eligible Greyling encounters/attempts, no death proof; minimum HP 66.6 at the declining food maximum, no cleanup errors. Supervisor preserved the terminal diagnostic, logged out/stopped Valheim, restored combat false and panel-only service. This earns orchestration/limits coverage only; positive melee, block and dodge remain open on the attack ticket. Native search without an already observed target is a useful next expansion, requiring explicit target binding/evidence before success. New user-requested marker/player-profile work follows; M2/M3 remain incomplete.

## Summary

Landed PR #72; durable combat goals, restricted Starlark decision policies, host budgets/proof, cancellation/restart handling, authenticated panel and persistent live supervisor deployed. Full CI and committed review pass. Bounded live goal safely exhausted its observation budget without an eligible Greyling, with evidence and cleanup retained. Actual combat success remains an independent open M3 check.
