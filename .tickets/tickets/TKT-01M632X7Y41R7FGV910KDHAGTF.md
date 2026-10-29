---
schema: 4
id: TKT-01M632X7Y41R7FGV910KDHAGTF
title: "Panel: make the map readable without zooming in"
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - controller
assignees: []
milestone: null
parent: TKT-01M628X106RD9FFR8YBXV7K5HT
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:claude-code/renki-groom
  branch: t3/panel
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: a239abe9b9791f12b72a6b9deaa078002e91b692
  session: null
  claimed_at: 2026-10-29T05:45:19Z
  expires_at: null
archive: null
created_at: 2026-10-29T05:42:44Z
updated_at: 2026-10-29T05:45:19Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

Nimah's screenshot (2026-10-29) shows the map has to be zoomed far in
before anything on it means something:

- Every structure is the same small grey square: a chest, a bed, a
  station and a fire look alike.
- Labels are drawn wherever they fall, so a row of chests at the base is
  an unreadable smear.
- Resources are anonymous green dots until 12 px per metre.
- It opens at 4 px per metre, so the 40 m perception radius fills a third
  of the map.

### What changes

- A shape and colour for each kind: chests, beds, fires, stations,
  portals, doors; trees, stumps, logs, rocks, pickables, drops; players,
  creatures, hostiles, tamed.
- Things of one kind close together on screen draw as one marker with a
  count ("6 chests"). Labels are placed in priority order (the bot,
  players, threats, home, left behind, structures, drops) and skipped
  where they would overlap.
- Hovering shows a tooltip for what is under the pointer, every item of a
  group: kind, prefab, distance, and the stack, open or shut, tag, stars
  or health as they apply.
- The first view fits the perception radius; the zoom buttons and the
  wheel work as now.

## Acceptance criteria

- [ ] Each kind of structure, resource and character has its own shape or colour, and the legend shows them
- [ ] Labels never overlap; nearby things of one kind group under one marker with a count
- [ ] Hovering a marker shows what it is: kind, prefab, distance from the bot, and its details
- [ ] The map opens zoomed to the perception radius, and the base is readable at that zoom
