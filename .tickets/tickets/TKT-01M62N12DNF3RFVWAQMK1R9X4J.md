---
schema: 4
id: TKT-01M62N12DNF3RFVWAQMK1R9X4J
title: "Plugin: equip and unequip items"
type: task
status: in-progress
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
claim:
  actor: agent:claude-code/renki-groom
  branch: t3/m2
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: aa2263bca5ab58f2a9742d8127c38a9002a8274f
  session: null
  claimed_at: 2026-10-29T15:34:54Z
  expires_at: null
archive: null
created_at: 2026-10-29T01:40:10Z
updated_at: 2026-10-29T15:34:54Z
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

## Implementation plan

equip {item: prefab}: of the stacks of that prefab, the unequipped one in best repair (then highest quality) through Humanoid.EquipItem. One already equipped succeeds as it is. A broken item, or one that is not equipment, is invalid_target, and the game saying no is refused, with what it said. unequip {slot}: right, left, hands (both), head, chest, legs, shoulder, utility, trinket, ammo, through UnequipItem; an empty slot is invalid_target. Both are instant and need AllowMovement, since they are the bot's own gear. Go: TaskEquip and TaskUnequip, args with the slot enum checked, KnownTasks, and the voice's menu. Docs: 7.2.
