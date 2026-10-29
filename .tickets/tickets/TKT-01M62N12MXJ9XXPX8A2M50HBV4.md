---
schema: 4
id: TKT-01M62N12MXJ9XXPX8A2M50HBV4
title: "Controller: corpse run after a death"
type: task
status: in-progress
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
  - TKT-01M62PRDRKEHTQ1WC12FQ3S11J
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:claude-code/renki-groom
  branch: t3/m2
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 48cb711abc94a3ce603bd1fef00cb17097057d3f
  session: null
  claimed_at: 2026-10-29T15:54:34Z
  expires_at: null
archive: null
created_at: 2026-10-29T01:40:10Z
updated_at: 2026-10-29T15:54:34Z
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

## Implementation plan

Lean version, ahead of full memory (TKT-01M62EP0P), because the bot lost its gear to three graves at home on 2026-10-29. Plugin: perception reports graves with something in them as structure kind 'tombstone', with mine on the bot's own (TombStone.Awake postfix, since they are not pieces). Controller: the home keeper remembers each death per world (the last 5) in the store. A death within 30 m of home keeps the bot from going back after the respawn, and it says so. Chat: 'get your things' empties its own grave in sight, else walks to the last one and empties it there. A grave gone is forgotten. The bot never goes back on its own, which is criterion 2.
