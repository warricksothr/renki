---
schema: 4
id: TKT-01M674C36AET3YJMR7AFQZB6AD
title: Design cancellable look-ahead between movement steps
type: spike
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - movement
  - controller
  - tooling
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
created_at: 2026-10-30T19:25:17Z
updated_at: 2026-10-30T19:25:17Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

The owner observed pauses between movement points and wants next steps prepared in advance, with the ability to displace them so transitions become smoother. Investigate the actual pauses in the autonomous scout/goal trace before choosing an optimization: distinguish arrival holds, controller/task handoff, asynchronous navmesh loading, local route construction and repeated live route checks. Design a bounded provisional next-leg/candidate slot that can be prepared while the current action runs, tagged with world UID/connection, task and goal/authority generations, expected start/arrival, geometry freshness and expiry. Revalidate before committing movement and discard on stop, owner takeover, retreat/threat, target changes, stale observations or changed support. Prepared actions must not own the body, emit controls or become success receipts. Reuse the existing procedure/native-policy research branch (PR #92) and avoid an unbounded command queue. Cover memory/frame budgets and diagnostic events that explain waits, replacement and cache invalidation.

## Acceptance criteria

- [ ] Retained live traces identify and measure the causes of pauses between legs and tasks
- [ ] A bounded provisional next-step design specifies authority, scope, freshness, displacement and preemption rules
- [ ] Implementation recommendations and regressions cover smooth handoff and immediate invalidation without unverified movement
