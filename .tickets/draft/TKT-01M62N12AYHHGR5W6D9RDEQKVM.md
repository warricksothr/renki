---
schema: 4
id: TKT-01M62N12AYHHGR5W6D9RDEQKVM
title: "Plugin: pickup task for drops on the ground"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - plugin
  - movement
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies:
  - TKT-01M62N129K1RGB8VSJZ3SDAC7N
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T01:40:10Z
updated_at: 2026-10-29T01:40:22Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

`pickup` (7.2): walk to a drop by entity id and take it, as a player would (Humanoid.Pickup), behind Safety.AllowGather. Fails with `inventory_full` when nothing more fits, `target_lost` when the drop is gone (someone else took it, or it despawned), and `unreachable` as move_to does.

Perception already reports drops with their stack, and the walker already reaches points.

## Acceptance criteria

- [ ] pickup walks to a drop and takes it, reporting how many it took
- [ ] It fails with inventory_full, target_lost or unreachable as 6.6 names them
- [ ] It is refused unless Safety.AllowGather is on
