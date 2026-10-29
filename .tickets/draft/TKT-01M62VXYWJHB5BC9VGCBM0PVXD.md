---
schema: 4
id: TKT-01M62VXYWJHB5BC9VGCBM0PVXD
title: Remember what each chest holds, its size and when it was seen
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - plugin
  - controller
  - protocol
assignees: []
milestone: M2
parent: TKT-01M62VXYQB6ZDBQE4E6ERGN83C
origin: null
dependencies:
  - TKT-01M62N12C9EWXCD54WMBH08T4E
  - TKT-01M62EP0PDAB02DNJAJ6WS3MYD
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:48Z
updated_at: 2026-10-29T03:40:48Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

Stowing, stock targets and sorting all need to know what is in the base's
chests without walking to each one.

### Plugin

When the bot opens a container (deposit, withdraw, or just looking), it
reports the container's entity id, size and contents in the same shape as
the inventory message (6.3). A new message or an event in 6.x, with a
fixture.

### Controller

Memory (TKT-01M62EP0) keeps per world, for each container: where it is,
what it is (chest, reinforced chest, a ship's hold), its size, its contents
and when they were last seen. Room for an item is worked out as for the
bot's own pack (`Inventory.Room`). Anything read from memory carries its
age, so a caller can decide whether it is fresh enough.

Players change chests behind the bot's back. A deposit that finds less room
than remembered corrects the memory from what it sees.

## Acceptance criteria

- [ ] Opening a container reports its id, size and contents, with a protocol fixture
- [ ] The controller keeps each container's contents and when they were seen, per world, across restarts
- [ ] Room for an item in a remembered chest is worked out as for the bot's pack
- [ ] What the bot finds on opening a chest replaces what it remembered
