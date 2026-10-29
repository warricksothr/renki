---
schema: 4
id: TKT-01M6348M7PNSNXWCW877A7FTDH
title: "Perception and panel: draw the walls of buildings near the bot"
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - plugin
  - protocol
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
  commit: 64b975c9484738143462ee88a7bcc23c9630f96d
  session: null
  claimed_at: 2026-10-29T06:06:26Z
  expires_at: null
archive: null
created_at: 2026-10-29T06:06:26Z
updated_at: 2026-10-29T06:06:26Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

Nimah, 2026-10-29: the map knows where beds, doors and fires are but not
the walls, so the base is a scatter of markers with no rooms. Perception
leaves walls and floors out on purpose (6.2): there are hundreds of them
and nothing the bot does needs each one.

A wall does not move, so the plugin works out once per piece whether it
is a wall (tall, thin and long, from the piece's own colliders in its own
frame) and where its foot runs, and sends the walls within the
perception radius as segments in a new `layout` message (6.9), only when
that set changes. The occupancy grid spike (TKT-01M62ACH) can use the
same data for paths later.

## Acceptance criteria

- [ ] The plugin sends the walls near the bot as segments in a layout message when they change, with a protocol fixture
- [ ] The controller keeps the newest layout, the audit log records it, and a new panel page gets it
- [ ] The panel map draws the walls under the markers, so rooms, doors and chests read in place
- [ ] A scan with layout stays within the perception budget
