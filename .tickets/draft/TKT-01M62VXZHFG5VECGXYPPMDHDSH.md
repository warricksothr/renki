---
schema: 4
id: TKT-01M62VXZHFG5VECGXYPPMDHDSH
title: "Controller: hunt down the creature that killed the owner"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
assignees: []
milestone: M3
parent: TKT-01M62N2557R401S3E6JECSAPN4
origin: null
dependencies:
  - TKT-01M62N25ADM0Z968D974G0CHP2
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

Part of defending the owner (TKT-01M62N25A). When a creature kills an
owner, the bot marks it and hunts it within a leash of where it happened,
if guard judges the fight winnable. Otherwise it remembers it, warns the
owner, and marks the place as a danger zone. Off by default, a switch on
the guard behaviour.

## Acceptance criteria

- [ ] A creature that killed an owner is marked and hunted when the bot can win the fight
- [ ] When it cannot, the bot warns the owner and marks the place as dangerous
- [ ] It is off by default
