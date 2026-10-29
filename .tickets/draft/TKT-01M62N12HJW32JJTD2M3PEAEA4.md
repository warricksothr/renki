---
schema: 4
id: TKT-01M62N12HJW32JJTD2M3PEAEA4
title: "Controller: haul what the owner drops home to a chest"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - chat
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies:
  - TKT-01M62N129K1RGB8VSJZ3SDAC7N
  - TKT-01M62N12AYHHGR5W6D9RDEQKVM
  - TKT-01M62N12C9EWXCD54WMBH08T4E
  - TKT-01M62EP0PDAB02DNJAJ6WS3MYD
  - TKT-01M62N12G98VW2CYW94FG5RCTB
  - TKT-01M62G71QCVNG3DVW1HZ8HXA23
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T01:40:10Z
updated_at: 2026-10-29T01:40:29Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

Scene 1 of the M2 benchmark. On "help me" / "collect wood" (and its voice equivalents), the bot stays near the owner, picks up drops near them (pickup) while there is room, and when it is full, or the owner has moved on and the pile is gone, carries the load home: the chest nearest home it knows of that has room (deposit). Then it comes back to the owner or the site.

It says what it is doing, briefly ("Full. Taking this home."), and what stops it: no chest known, chests full, nothing to pick up. "stop" ends it. Owners only.

## Acceptance criteria

- [ ] On the command, the bot picks up drops near the owner while it has room
- [ ] Full, it carries the load to a chest at home, stores it, and goes back
- [ ] It says when it cannot go on: no chest, chests full
- [ ] stop ends the routine at once
