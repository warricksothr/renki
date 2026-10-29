---
schema: 4
id: TKT-01M62N12F04QEC0F53JCS1M21A
title: "Plugin: harvest task: chop, split and break with the best tool"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - plugin
  - movement
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

`harvest` (7.2): approach a tree, log, stump or rock by entity id, equip the best tool that meets its `minTool`, face it, and swing until it breaks; with `collect` (the default), sweep up the drops within a few metres afterwards.

A felled tree becomes a new `log` object and a `stump`; a log splits into halves, each a new object, and those break into wood (6.2). One harvest is one object; the controller chains them by watching perception for what appeared where the old one was. Swinging needs care with stamina, and with the falling trunk, which hurts.

Fails with `no_tool` when nothing in the inventory meets minTool, `inventory_full` when collecting cannot fit more, and `target_lost` when someone else breaks it first.

## Acceptance criteria

- [ ] harvest fells a tree, splits a log and breaks a stump or rock by id with the best tool it carries
- [ ] It collects the drops within a few metres unless collect is false
- [ ] It fails with no_tool, inventory_full or target_lost as 6.6 names them
- [ ] It does not stand under a falling trunk
