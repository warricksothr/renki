---
schema: 4
id: TKT-01M62VXZ2FWDX3D4Z6XX1MB5A6
title: "Base upkeep: fuel, machines and chest assignments"
type: epic
status: draft
status_reason: null
priority: normal
due_on: "2026-12-06"
labels:
  - controller
  - plugin
assignees: []
milestone: M5
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:48Z
updated_at: 2026-11-02T15:59:48Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-plan-m5
  name: ""
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

### Where the code stands (2026-11-02)

- The behaviour engine, `stow`, the idle-at-home triggers and `marker_cargo` (surplus cargo into chests that already hold the item) exist: internal/behaviour/engine.go, stow.go, idle.go, marker_cargo.go. See docs/BEHAVIOURS.md.
- Perception reports fire fuel and lit, and cooking-station slots with `cooking`, `done` and `burnt` (docs/BRIDGE-PROTOCOL.md, structures; plugin/Renki.Plugin/Perceiver.cs). Smelters are listed with no state; fermenters and beehives are not reported at all.
- Native cooking of one input, with its one-wood fuel step for a cold supporting fire, works live (docs/COOKING.md). There is no standalone task to fuel a fire or a smelter.
- Remembered chest contents persist with their ages (docs/MEMORY.md, internal/home/containers.go).
- Nothing yet keeps fires lit, takes other people's food off, or holds an item-to-chest table.

### M5 goals this epic owns

docs/ROADMAP.md, M5: goal 5.1 (fires stay lit), 5.2 (no food burns) and 5.3 (assigned items reach their chests), each measured over two consecutive 2 h home soaks by TKT-01M6EFM4G7 (VM: two 2 h base upkeep soaks with the upkeep report) using the report from TKT-01M6EFK453 (VM: base upkeep report from a home soak audit window).

### Children and order

1. TKT-01M62VXZ4D (Plugin: add fuel and ore to machines and take their output), then TKT-01M62VXZ5P (Controller: fuel template) for goal 5.1.
2. TKT-01M62VXZ6Z (Controller: tend template, in progress) for goal 5.2.
3. TKT-01M62VXZ84 (Controller: assign every item to chests), then TKT-01M6EFKDSV (sort misplaced items) and TKT-01M6EFKDQQ (panel editor) for goal 5.3.
4. TKT-01M62VXZ39 (Perception: machine state) is needed only for smelters, kilns, fermenters and beehives.
5. TKT-01M62VXZ97 (suggest assignments) is unscheduled, a stretch beyond M5.

## Acceptance criteria

- [ ] With fuel on, fires in the base stay lit when cold or at night (furnaces, smelters and kilns join this criterion only if the owner keeps machines in M5)
- [ ] With machines on, food comes off cooking stations before it burns
- [ ] Every item an owner has assigned ends up in its chests, and the owners hear when there is no room

## Definition of done

- [ ] M5 goals 5.1, 5.2 and 5.3 (docs/ROADMAP.md) are met: two consecutive upkeep soak summaries from TKT-01M6EFM4G7 are linked in a note here
