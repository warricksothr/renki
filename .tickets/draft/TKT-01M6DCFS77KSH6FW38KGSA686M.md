---
schema: 4
id: TKT-01M6DCFS77KSH6FW38KGSA686M
title: "Plugin: stop per-step allocations in reach and sight checks"
type: chore
status: draft
status_reason: null
priority: low
due_on: null
labels:
  - plugin
  - live
assignees: []
milestone: M4
parent: TKT-01M6DCF2ABDDNG3QF369NDPBT5
origin: null
dependencies:
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T05:42:33Z
updated_at: 2026-11-02T15:57:05Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m4
  name: ""
extensions: {}
---

## Description

Several reach and line-of-sight checks allocate on every 50 Hz physics step. Each one calls `GetComponentsInChildren<Collider>()`, then `Physics.RaycastAll` and LINQ over the colliders.

### Where

- `plugin/Renki.Plugin/UseReach.cs:191-197`
- `InteractTask.cs:217` and `:241`
- `TransferTask.cs:48` (runs for the whole walk to a chest)
- `CookingTask.cs:174`

### Why it matters

Valheim's Mono collector stops the world, so steady garbage becomes periodic hitches on the bot's client. That client is also the one on the stream.

### Fix

- Cache the target's colliders when the task starts.
- Use `Physics.RaycastNonAlloc` into a reused buffer.
- Run the check only within reach distance, or at about 4 Hz.

Measure before and after with the Unity profiler or GC counts in a trace.

### Goal

Goal 4.6 (review closed). Low priority: hitches matter on the streamed client, but nothing in goals 4.1-4.5 needs this.

### Files

Line numbers refer to commit f1a5483.

- `plugin/Renki.Plugin/UseReach.cs`, `plugin/Renki.Plugin/InteractTask.cs`, `plugin/Renki.Plugin/TransferTask.cs`, `plugin/Renki.Plugin/CookingTask.cs`

### Verification

- `just check plugin build` with `VALHEIM_DIR` set, and `just check ci`.
- Live, under the VM lease (TKT-01M6EF5DTG): deploy with `just vm deploy`, walk Tas to a home chest (a deposit through the panel or `just vm resupply-food`), and record GC allocation per physics step before and after from a plugin trace. The measurement needs the game, so this ticket is in the live lane.

## Acceptance criteria

- [ ] Reach and sight checks reuse buffers and cached colliders
- [ ] Measured GC allocation per physics step during a chest walk is recorded before and after
