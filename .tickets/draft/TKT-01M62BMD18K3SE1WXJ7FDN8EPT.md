---
schema: 4
id: TKT-01M62BMD18K3SE1WXJ7FDN8EPT
title: "Plugin: send perception of characters, structures and resources"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - protocol
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-28T22:55:57Z
updated_at: 2026-10-28T22:55:57Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

The protocol defines `perception` (docs/BRIDGE-PROTOCOL.md 6.2), but the
plugin has never sent it: TKT-01M61T36M4 left perception for M2. Without it
the controller and the panel know nothing around the bot but chat, and the
panel's map shows only the bot itself.

### What to send

At `perceptionHz` (default 1), within `radius`, nearest first, capped at
`maxPerKind` per kind, as 6.2 describes:

- `characters`: `Character.GetCharactersInRange`; players by name, creatures
  by prefab, with hpFrac, stars, hostile, targetingMe, and tamed.
- `structures`: `Piece.GetAllPiecesInRadius`, reduced to the kinds 6.2 names
  (container, station, bed, portal, door, fire, smelter). Doors carry
  whether they are open, which 6.2 does not have yet: add it there.
- `resources`: trees, logs, rocks, pickables and drops, by component
  (TreeBase, TreeLog, MineRock, Pickable, ItemDrop).

### Uses

- The panel's map draws what is really around the bot.
- The controller and the LLM's summary know about doors, beds, portals and
  threats.
- TKT-01M62ACH8 (sparse occupancy grid spike) can rule doors in or out from
  real data.

### Watch

Frame size and frame time: a base can hold hundreds of pieces, so the caps
matter, and the scan should not run every physics frame.

## Acceptance criteria

- [ ] The plugin sends perception at perceptionHz with characters, structures and resources as 6.2 describes, capped and nearest first
- [ ] Doors report whether they are open; 6.2 and the fixtures on both sides say so
- [ ] The panel's map shows players, creatures and structures from the real game
- [ ] A perception scan in a built-up base takes under 2 ms, measured in the game
