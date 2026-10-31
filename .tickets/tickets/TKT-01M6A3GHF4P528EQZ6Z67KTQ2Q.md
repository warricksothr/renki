---
schema: 4
id: TKT-01M6A3GHF4P528EQZ6Z67KTQ2Q
title: Stock household food through bounded pressure-driven jobs
type: task
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - safety
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
  branch: t3/raised-home-refusals
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: e14d7b1bf6a5da3e4a4acd98186e3425c263cea5
  session: null
  claimed_at: 2026-10-31T23:08:02Z
  expires_at: null
archive: null
created_at: 2026-10-31T23:07:58Z
updated_at: 2026-10-31T23:22:54Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

Implement the authorized household food slice independently of tool crafting/repair. Extend durable food goals with optional per-prefab household storage targets, an explicit set of destination chests, and named forage markers. Normal personal resupply remains the first priority. Fresh task-matched chest observations determine known stock; ordinary one-meal withdrawal probes can inspect unknown chests without a new native task. Stock is not inferred from memory. Surplus is deposited through matched native transfer plus fresh pack evidence. Multiple ready-food collections may continue before returning according to remaining shortage, carry capacity, remaining meals, distance and danger; a committed delivery phase prevents oscillation. Cooking reuses existing native task/receipts, bound to home. Idle episodes use the existing behaviour lease/dwell/cooldown/run cap and a runner lifecycle checkpoint hook; interrupted or uncertain actions are never replayed. Diagnostics expose the pressure decision and shortages. Equipment repair and cultivation remain on their existing separate tickets.

## Acceptance criteria

- [ ] Configure per-food household storage targets while retaining Tas personal reserve
- [ ] Fresh unique chest receipts and matched deposits prove stock; memory and missing access cannot claim success
- [ ] Bounded idle episodes select collection, cooking and committed delivery from shortages, capacity, distance and danger
- [ ] Cancellation, restart uncertainty and edited scopes cannot repeat uncertain native actions
- [ ] Focused regression checks, full gate and a bounded live household stock episode retain honest evidence

## Implementation plan

Implement the authorized household food slice independently of tool crafting/repair. Extend durable food goals with optional per-prefab household storage targets, an explicit set of destination chests, and named forage markers. Normal personal resupply remains the first priority. Fresh task-matched chest observations determine known stock; ordinary one-meal withdrawal probes can inspect unknown chests without a new native task. Stock is not inferred from memory. Surplus is deposited through matched native transfer plus fresh pack evidence. Multiple ready-food collections may continue before returning according to remaining shortage, carry capacity, remaining meals, distance and danger; a committed delivery phase prevents oscillation. Cooking reuses existing native task/receipts, bound to home. Idle episodes use the existing behaviour lease/dwell/cooldown/run cap and a runner lifecycle checkpoint hook; interrupted or uncertain actions are never replayed. Diagnostics expose the pressure decision and shortages. Equipment repair and cultivation remain on their existing separate tickets.

## Notes

**agent:codex/renki-milestones** at 2026-10-31T23:22:54Z

Implementation is on t3/household-food-stock above navigation PR #135. Adds optional stock targets to durable food goals, explicit unique destinations, current-run three-minute chest observations earned by normal one-meal probes, verified surplus deposits, source marker pinning, pressure-based multi-collection and committed delivery. Guarded idle episodes have distinct durable IDs, runner lifecycle checkpoints and real goal-stop routing to their owning behaviour. Household cooking returns precisely home before eating/healing. Panel controls and desktop/mobile request automation passed; focused goal/behaviour/app checks passed. Full gate is running; live household cooking/deposit trial remains pending. Broader repair-dependent pressure ticket remains draft with its dependency intact.
