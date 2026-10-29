---
schema: 4
id: TKT-01M62N129K1RGB8VSJZ3SDAC7N
title: "Plugin: send the inventory on change (6.3)"
type: task
status: done
status_reason: null
priority: high
due_on: null
labels:
  - plugin
  - protocol
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
updated_at: 2026-10-29T02:57:48Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

The controller cannot decide to haul without knowing what the bot carries and how much room is left. 6.3 defines `inventory`, sent on change and once after `welcome`; the plugin has never sent it.

Send the grid size, each occupied slot (prefab, stack, stackMax, quality, durability, equipped), and the carried weight against the maximum, which state already has. Watch Inventory.m_onChanged instead of polling.

## Acceptance criteria

- [x] The plugin sends inventory once after welcome and on every change, as 6.3 describes
- [x] The controller keeps the newest inventory and can say how many of a prefab it holds and how many more fit

## Notes

**agent:claude-code/renki-groom** at 2026-10-29T02:40:17Z

Verified in the game on 2026-10-28. After the deploy at 21:36:59 the first inventory arrived ('32 free of 32; empty'). Nimah dropped items at the bot's feet, and each change came through as it happened: 21:39:37 38 Wood; 21:39:40 16 Stone and 13 Flint; 21:39:43 42 more Wood, which the game stacked as 50 + 30.
