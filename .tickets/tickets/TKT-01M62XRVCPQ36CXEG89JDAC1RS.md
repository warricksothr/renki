---
schema: 4
id: TKT-01M62XRVCPQ36CXEG89JDAC1RS
title: Drop the least valuable, heaviest items when overloaded, and remember them
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - plugin
  - controller
  - protocol
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
  branch: t3/pickup
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: b747d3554e70b43a2d6b8a9684de164ed91d041b
  session: null
  claimed_at: 2026-10-29T04:12:58Z
  expires_at: null
archive: null
created_at: 2026-10-29T04:12:58Z
updated_at: 2026-10-29T04:12:58Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

In the game (2026-10-28) the bot reached 306/300 and could not move: over
its carry weight the game stops its stamina coming back and, with none
left, will not move it at all. Players cannot take items out of another
player's pack, so only the bot can fix it.

### What it does

When the bot is encumbered, it drops the least valuable, heaviest items
first, just enough to get back under its carry weight. It never drops what
it has equipped, or its tools, weapons and armour. Then it says what it had
to leave behind, and remembers where, per world, so the drops can be picked
up later (by an owner, or by the bot once hauling exists).

### Pieces

- **Plugin:** a `drop` task (instant): drop N of a prefab at the bot's feet,
  never an equipped item, behind a Safety switch. The game marks a player's
  own drop so its auto-pickup does not take it straight back.
- **Inventory (6.3):** each item's unit `weight`, its `type` (material,
  consumable, weapon...) and `value` where the game gives one, so the
  controller can rank them.
- **Controller:** the ranking and the reflex, a switch to turn it off, the
  announcement, and a left-behind record in the controller's memory that is
  forgotten once perception shows the drops gone. Status and the panel map
  show what was left where.

## Acceptance criteria

- [ ] An encumbered bot drops the least valuable, heaviest items first until it is under its carry weight
- [ ] It never drops equipped items, tools, weapons or armour
- [ ] It says what it left behind
- [ ] Where it left what is remembered per world, shown in status and on the panel map, and forgotten once the drops are gone
- [ ] A switch turns it off, and the plugin's drop task sits behind a Safety switch
