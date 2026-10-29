---
schema: 4
id: TKT-01M62VXZ2FWDX3D4Z6XX1MB5A6
title: "Base upkeep: fuel, machines and chest assignments"
type: epic
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - plugin
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

Keeping the base running is a set of templates on the behaviour engine:

- **Fuel:** keep fires lit when it is cold or night, or when someone is
  cooking over them, and keep smelters and kilns fuelled while they have
  something queued.
- **Machines:** take food off cooking stations before it burns, and collect
  from fermenters and beehives when ready. Feed ore and wood from the chests
  into smelters and kilns. Bars, coal and finished items fall to the ground
  in Valheim, so loot pickup and stow already deal with most output.
- **Organising:** a table giving every item its chests, which becomes `stow`
  instances, plus a chest for loose items. Learning what each chest is for
  comes last.

Everything here is off by default until an owner turns it on. It all runs
inside the base, around home.

## Acceptance criteria

- [ ] With fuel on, fires in the base stay lit when cold or at night and working furnaces stay fuelled
- [ ] With machines on, food comes off cooking stations before it burns
- [ ] Every item an owner has assigned ends up in its chests, and the owners hear when there is no room
