---
schema: 4
id: TKT-01M62N12HJW32JJTD2M3PEAEA4
title: "Controller: loot and deliver-when-full behaviours for hauling"
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
  - TKT-01M62VXYSN4RQV75HTA3BSD162
  - TKT-01M62VXYY6F094WXQS3FBQ57RF
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T01:40:10Z
updated_at: 2026-10-29T03:40:55Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

Scene 1 of the M2 benchmark, built as two instances on the behaviour engine
rather than a hand-written routine (reshaped 2026-10-28, see the behaviours
epic TKT-01M62VXYQB6).

- **loot** (template `loot`, on by default): picks up drops near the bot,
  or near the owner it follows, while there is room. It leaves alone what an
  owner dropped on purpose: a drop that appears at an owner's feet with no
  felling, splitting or breaking nearby counts as deliberate for a few
  minutes.
- **deliver when full** (a `stow` instance, `items: *`, triggered by
  inventory full): carries the load to the chests at home, then goes back to
  the owner or the site. Choosing a ship or a waypoint instead comes later
  (TKT-01M62VXZ1JG9).

"help me" / "collect wood" (and the voice) switch both on for the owner who
asked. "stop" ends the current run. It says briefly what it is doing ("Full.
Taking this home.") and what stops it: no chest, chests full, nothing to pick
up. Owners only.

## Acceptance criteria

- [ ] loot runs as an instance, on by default, and picks up drops near the bot or its leader while there is room
- [ ] loot leaves an owner's deliberate drops alone
- [ ] Full, deliver-when-full stows the load in the chests at home and goes back
- [ ] It says when it cannot go on (no chest, chests full), and stop ends the run at once
