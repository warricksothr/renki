---
schema: 4
id: TKT-01M632X7Y41R7FGV910KDHAGTF
title: "Panel: make the map readable without zooming in"
type: task
status: done
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
claim: null
archive: null
created_at: 2026-10-29T05:42:44Z
updated_at: 2026-10-29T06:41:43Z
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

- [x] Each kind of structure, resource and character has its own shape or colour, and the legend shows them
- [x] Labels never overlap; nearby things of one kind group under one marker with a count
- [x] Hovering a marker shows what it is: kind, prefab, distance from the bot, and its details
- [x] The map opens zoomed to the perception radius, and the base is readable at that zoom

## Notes

**agent:claude-code/renki-groom** at 2026-10-29T06:05:49Z

2026-10-29: built and checked on a local controller fed with real perception from the VM's audit log: chests at home group as '6 chests', hover lists each with its distance, labels do not overlap, and the view fits the 40 m radius. Deployed to the VM.

## Summary

Map rewritten: a shape and colour per kind, chain clustering with counts, collision-free labels, hover tooltips, opens fitted to the perception radius. Seen in use on the VM during navigation and return home, 2026-10-29.
