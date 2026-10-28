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
milestone: null
parent: null
origin: null
dependencies:
  - TKT-01M62BMD18K3SE1WXJ7FDN8EPT
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-28T23:49:16Z
updated_at: 2026-10-28T23:49:16Z
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
