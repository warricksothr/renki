---
schema: 4
id: TKT-01M62N12C9EWXCD54WMBH08T4E
title: "Plugin: deposit into a chest (and withdraw)"
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

`deposit` (7.2): walk to a container by entity id, open it as a player would, move the named items (by prefab, a count or "all") into it, and close it. `withdraw` is the same the other way. A full chest stores what fits and reports the rest; a chest in use by someone else or behind a ward fails with a reason the controller can say.

## Acceptance criteria

- [ ] deposit moves the named items into a chest by id and reports what it moved and what did not fit
- [ ] withdraw takes named items out
- [ ] A chest that is full, in use or warded fails with a reason, not silently
