---
schema: 4
id: TKT-01M62ACH8W3RVZC77KB21CB383
title: "Spike: sparse occupancy grid to path around obstacles the navmesh misses"
type: spike
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - movement
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
created_at: 2026-10-28T22:34:11Z
updated_at: 2026-10-28T22:34:11Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

The bot gets stuck inside and around buildings. Valheim's navmesh path
(`Pathfinding.GetPath`, Humanoid agent) is the only planner today, with a
straight line when it finds nothing, and stuck recovery (jump, strafe,
re-path) as the only answer to obstacles it does not know about.

### The idea

We have exact world positions, so the plugin can keep a sparse occupancy
grid: cells of about 1 m keyed by (x, z), stored only once something is
known about them.

- What fills it: blocked where the bot collided or stuck recovery fired;
  walkable where it has walked; and on-demand probes from the plugin
  (capsule casts for walls, a ray down for ground height and slope, water
  level), run for unknown cells near the planned route.
- Dynamic areas: players build and tear down, and doors open and close, so
  learned cells decay unless seen again, and a probe overrides memory.
- How it is used: navmesh first; when that fails, or stuck recovery starts,
  A* over the grid in a window around the bot (say 64 by 64 m), with
  unknown cells costed higher than known walkable ones.
- Where it lives: the planner in the plugin, which has the physics. What it
  learns could go to the controller per world (as `trace` or a new message)
  so it survives restarts and the panel can draw it.

### Questions the spike answers

1. Which failure is it in practice? Use the audit log's `walker` traces from
   real stuck cases: `no navmesh path; straight line`, or a navmesh path
   that runs into something.
2. Doors: is the real blocker inside buildings a closed door? Opening doors
   (`Door.Interact`) may matter more than a grid.
3. Do Valheim's navmesh tiles cover player-built pieces, and how fast do
   they update after building?
4. What do probes cost per frame, and what cell size works on Valheim's
   terrain and stairs?

## Acceptance criteria

- [ ] Audit traces from at least three real stuck cases say whether the navmesh failed or its path ran into something
- [ ] Doors are ruled in or out as the main blocker inside buildings
- [ ] A design note: cell size, what fills and decays cells, probe cost, and where the grid lives
