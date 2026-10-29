---
schema: 4
id: TKT-01M62VXZ0J9Q2WH4XN4E6N7F27
title: Know which parts of the map the bot has explored
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - controller
  - protocol
assignees: []
milestone: null
parent: TKT-01M62VXYQB6ZDBQE4E6ERGN83C
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

The minimap knows which areas are explored. The plugin sends that, coarse
and only when it grows, so the controller can prefer routes over known
ground and avoid sending the bot across unexplored land to reach a goal.

This needs a protocol message (or a field on an existing one) for the
explored cells and a fixture. The controller keeps it per world.

## Acceptance criteria

- [ ] The plugin reports explored map cells when they change, with a protocol fixture
- [ ] The controller keeps the explored area per world and can say whether a straight route crosses unexplored land
