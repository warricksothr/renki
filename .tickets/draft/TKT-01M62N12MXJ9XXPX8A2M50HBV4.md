---
schema: 4
id: TKT-01M62N12MXJ9XXPX8A2M50HBV4
title: "Controller: corpse run after a death"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - movement
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies:
  - TKT-01M62EP0PDAB02DNJAJ6WS3MYD
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T01:40:10Z
updated_at: 2026-10-29T01:40:23Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

Agreed on 2026-10-28 for M2: after a death, the bot's things are in a tombstone where it died. Memory records each death's place (TKT-01M62EP0P); the corpse run walks back there, empties the tombstone (interact), and returns, if afterDeath allows and the place is not dangerous. A haul that dies otherwise loses the load and the axe.

## Acceptance criteria

- [ ] After a respawn the bot can go back to its tombstone, empty it and return
- [ ] It does not go back into a place that just killed it without being told
