---
schema: 4
id: TKT-01M62EP0PDAB02DNJAJ6WS3MYD
title: "Controller: remember where things are and should be, per world"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies:
  - TKT-01M62BMD18K3SE1WXJ7FDN8EPT
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-28T23:49:16Z
updated_at: 2026-10-29T01:40:22Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

The bot forgets everything it sees. Perception (TKT-01M62BMD1) tells the
controller what is around the bot now; memory keeps it, per world, so the
bot knows where things are when they are out of sight, and notices when
they are not where they should be.

### What it remembers

- **Places and things:** structures from perception (beds, chests,
  stations, portals, doors, fires) with position, prefab and when last
  seen; resources worth coming back to (ore, berry bushes); the home.
- **Expectations:** a remembered thing that perception should see from
  where the bot stands but does not is marked missing, with when; one that
  moved is updated. That is what "should be" means: the last known state,
  checked against the present.
- **Deaths and danger:** each death with place, cause (attacker id or
  prefab, drowning, falling) and time; places that hurt the bot become
  danger zones that decay over days, for pathing to avoid and the panel to
  show.

### Where it lives

The controller, beside the homes in dataDir (or its own file there),
written as perception arrives and pruned as entries age. The panel map
draws remembered things faintly and current ones solidly. The LLM summary
can say what is near ("there's a chest by the workbench").

## Acceptance criteria

- [ ] Structures and notable resources from perception are kept per world across restarts, with when each was last seen
- [ ] A remembered thing in view that perception no longer reports is marked missing
- [ ] Each death is recorded with place, cause and time, and makes a danger zone that decays
- [ ] The panel map draws remembered things and danger zones

## Notes

**agent:claude-code/renki-groom** at 2026-10-29T01:16:12Z

### Fidelity and attention (from Nimah, 2026-10-28)

The bot should not remember everything at the same fidelity. It should save its attention for the values that change and can drive behaviour. The design, given that:

- **Flyweights.** Each prefab is described once, in a per-world catalogue: kind, minTool, what it yields, and whether it is a material worth collecting. A remembered thing holds only its id, an index into that catalogue, a position rounded to 0.5 m, when it was last seen, and the few fields that change. A forest of thousands of trees then costs a few bytes a tree.
- **Fidelity tiers.**
  - **Scenery** (trees, rocks, logs): it never moves, so it is kept as counts per map cell ("cell (-40, 0): 14 Beech1, 2 rock4_coast_frac"), not one record each. The exceptions are things worth coming back for (ore, berry bushes), which are kept individually.
  - **Structures:** one record each. They rarely change; a door's open state is transient and is not stored.
  - **High signal:** drops, creatures and players, tracked individually with a short history. Drops are grouped per cell and prefab, with the total stack and its trend over time.
- **Attention is change.** The controller derives signals from differences, not from snapshots. Examples:
  - "Wood on the ground near (x, z): 12, up from 3 a minute ago" (chopping leaves a pile).
  - "Stone and copper ore accumulating by the mine."
  - "A creature has the bot as its target."
  - "A chest that should be here is not."

  Signals are what the brain, the LLM summary and future behaviours act on. For example, picking up a growing pile when the inventory has room needs the inventory stream (6.3) and a pickup task (M2, behind AllowGather). That behaviour is for M2 grooming.
- **The wire can follow suit later.** Characters and drops matter on every scan; scenery and structures could go out only when they change. The v1 snapshot is about 25 items a second, so this can wait until size matters (6.2 already says deltas come later).
