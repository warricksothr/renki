---
schema: 4
id: TKT-01M62N12M69CCYJ2STYPT31KJ5
title: "Controller and plugin: eat to recover health"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - controller
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies:
  - TKT-01M62N129K1RGB8VSJZ3SDAC7N
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T01:40:10Z
updated_at: 2026-10-29T01:40:22Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

The game heals only from food, and the bot never eats: after the fights on 2026-10-28 it sat at 11/25 health for good. use_item (7.2) lets it eat; the controller keeps food slots filled from what it carries, or asks the owner for food when it has none.

## Acceptance criteria

- [ ] use_item eats a food by prefab
- [ ] The bot keeps its food slots filled from its inventory
- [ ] It asks an owner for food when it has none and its health is low
