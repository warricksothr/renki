---
schema: 4
id: TKT-01M62PRDRKEHTQ1WC12FQ3S11J
title: "Plugin: interact task for beds, tombstones and the like"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - plugin
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T02:10:24Z
updated_at: 2026-10-29T02:10:24Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

`interact` (7.2): walk to an entity by id and use it as a player would (its Interactable, with `alt` and `hold`), behind the safety switch that fits what it is. The building block for claiming a bed, emptying a tombstone on a corpse run, and later levers and the like. Reports what the game answered, including refusals (a ward, a bed someone else owns, "you need a roof").

## Acceptance criteria

- [ ] interact walks to an entity and uses it, with alt and hold
- [ ] A refusal comes back as a failure with the game's reason, not a success
