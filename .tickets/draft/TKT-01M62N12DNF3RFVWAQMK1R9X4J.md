---
schema: 4
id: TKT-01M62N12DNF3RFVWAQMK1R9X4J
title: "Plugin: equip and unequip items"
type: task
status: draft
status_reason: null
priority: normal
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
created_at: 2026-10-29T01:40:10Z
updated_at: 2026-10-29T01:40:10Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

`equip` / `unequip` (7.2), instant tasks: equip an item from the inventory by prefab (an axe for chopping), or empty a slot. harvest equips the best tool itself, but the controller and the owner ("hold your axe") need it too.

## Acceptance criteria

- [ ] equip and unequip work by prefab and slot and report the result
