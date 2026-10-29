---
schema: 4
id: TKT-01M62VXYZMVEP74QVV9SRKDWGH
title: "Waypoints: named places an owner marks"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - chat
assignees: []
milestone: null
parent: TKT-01M62VXYQB6ZDBQE4E6ERGN83C
origin: null
dependencies:
  - TKT-01M62EP0PDAB02DNJAJ6WS3MYD
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

Owners name places from chat: `<name> mark this as the mine`, `forget the
mine`. A waypoint is a position and a name, per world, kept in memory and
drawn on the panel map. Behaviours and chat can use the name: "go to the
mine", or a `stow` whose chests are at the mine.

The home stays its own thing. A waypoint is any other place worth a name.

## Acceptance criteria

- [ ] Owners can mark, rename and forget named places from chat, per world
- [ ] Waypoints survive a restart and show on the panel map
- [ ] move_to and behaviour parameters accept a waypoint by name
