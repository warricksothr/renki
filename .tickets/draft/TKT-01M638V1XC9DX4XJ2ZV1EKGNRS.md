---
schema: 4
id: TKT-01M638V1XC9DX4XJ2ZV1EKGNRS
title: "Follow: stuck outside a wall, the leader inside, path straight through"
type: bug
status: draft
status_reason: null
priority: normal
due_on: null
labels:
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
created_at: 2026-10-29T07:26:24Z
updated_at: 2026-10-29T07:26:24Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

Seen on the VM, 2026-10-29 01:39-01:40, follow task t-1: Nimah stood inside the house at (-31.54, 36.09, 12.45). The bot stood outside the east wall at about (-28.1, 36.09, 11.55), 3.5 m away. The wall runs from (-28.6, 12.0) to (-29.4, 10.1), per the layout. Every re-path returned 2 waypoints, a straight line through the wall. The walker went through 9 recoveries (Jump, Strafe, Repath, Climb, three rounds) and gave up `stuck`, which sent it home.

Open question: why did the navmesh give a straight path through a wall? Either the navmesh does not include that wall, there is a gap the agent fits through and the bot does not, or GetPath returned a partial path that we treat as whole. Read the path trace's `end` (new: it is set when a path stops more than 1 m short) the next time this happens.

Follow should also not count it as stuck when it is already within its distance plus a little of the leader and cannot get closer.

## Acceptance criteria

- [ ] The cause of the straight path through the wall is known and written down
- [ ] A follower that is near the leader but cannot get closer waits, not recovers
