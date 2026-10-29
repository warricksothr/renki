---
schema: 4
id: TKT-01M638V1VYY93J6343W7CJAD2X
title: "Stuck recovery: don't strafe off a ledge"
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

Seen on the VM, 2026-10-29 01:40: sent home, the bot stood stuck on the deck west of the house at (-42.13, 36.09, 8.93). The second recovery, Strafe, walked it sideways off the deck edge, and it fell 2.5 m to the ground at (-45.22, 33.62, 9.68), under the house's level. From there the way home ran under the raised floor.

Strafe and the push after a climb move the bot sideways or forwards without looking where it will land. Before such a step it should check for ground within a step's drop (a ray down a little ahead of the bot), and choose the other side, or skip the strafe, when there is none.

## Acceptance criteria

- [ ] Strafe and the climb's push do not step where the ground drops more than about a metre
- [ ] A trace says when a recovery skipped a side because of a drop
