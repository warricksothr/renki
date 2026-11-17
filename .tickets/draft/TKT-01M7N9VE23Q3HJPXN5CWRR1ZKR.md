---
schema: 4
id: TKT-01M7N9VE23Q3HJPXN5CWRR1ZKR
title: "Inspect: no approach found when Tas starts flush against a chest row"
type: bug
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - plugin
  - movement
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
created_at: 2026-11-17T17:46:07Z
updated_at: 2026-11-17T17:46:07Z
created_by:
  id: agent:claude-code/chests
  name: ""
updated_by:
  id: agent:claude-code/chests
  name: ""
extensions: {}
---

## Description

Seen live on 2026-11-17 during the first chest census at the new home warehouse.

After opening a chest in the floor row, Tas stood at about [-166.36, 30.97, -464.47], flush against that row with a barrel shelf above it. From there eleven consecutive `inspect_container` requests failed `unreachable` with detail "no supported, clear chest interaction position after bounded alternate approaches" and travel of 0 to 0.8 m. The targets included chests 10 m away across open floor (1:15512, 1:15522).

After `move_to` the open west approach [-169.69, 31.03, -467.8], the same chest 1:15512 opened at once with 3.3 m of travel (task t-38, .checks/task-9ja65k54.json). So the failure depends on where the body starts, not on the target.

The census tool works around it by walking back to its start point and retrying once (scripts/vm/chest-census.py, `next_step`). Stow, search_chests and the planned sorting will meet the same spot without that workaround.

### To find out

- Whether the approach search starts from a position it treats as unsupported or blocked when the body touches chest colliders, so no candidate route leaves it.
- Whether the bounded alternate approaches are measured from the body rather than from the target.

Evidence: .checks/census/20261117T174422Z.json (results 13 to 27).

## Acceptance criteria

- [ ] From a position touching a chest row, inspect_container reaches a chest across open floor
- [ ] A regression test covers a start position in contact with container colliders
