---
schema: 4
id: TKT-01M62VXZKJFMNNZS4PC0T4RCDV
title: "Doors: a door whose state returns after 3 s counts as refused"
type: bug
status: draft
status_reason: null
priority: low
due_on: null
labels:
  - plugin
  - movement
assignees: []
milestone: null
parent: null
origin: null
dependencies:
  - TKT-01M62MB5YRGF9FT485055911JP
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:49Z
updated_at: 2026-10-29T03:40:49Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

Doorman waits SettleFor (3 s) for an opened door's state to come back. On a
slow link it can arrive later, so Tick judges the door refused, forgets it,
and does not shut it behind the bot. The door is then left open. Accepted in
PR 3 as rare. Fix if it shows up: wait longer while the door's owner has not
answered, or watch for the state for a while after judging it refused.

## Acceptance criteria

- [ ] A door that opens after the settle time is still shut behind the bot
