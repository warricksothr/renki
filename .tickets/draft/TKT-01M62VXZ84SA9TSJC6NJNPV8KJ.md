---
schema: 4
id: TKT-01M62VXZ84SA9TSJC6NJNPV8KJ
title: "Controller: assign every item to chests, with a loose-items chest"
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
parent: TKT-01M62VXZ2FWDX3D4Z6XX1MB5A6
origin: null
dependencies:
  - TKT-01M62VXYY6F094WXQS3FBQ57RF
  - TKT-01M62VXYV33R1WXRHMRZ6GZMRH
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

A table per bot, item or group → an ordered list of chests, which generates
the `stow` instances. A chest marked loose takes anything without an
assignment. With none marked, unassigned items are reported instead.
Owners edit it from chat and the panel: `<name> wood goes in chest:wood-1`.

Sorting is moving items already in chests to where they are assigned. It
runs only while the bot is idle at home and nobody is using the chests, and
any order interrupts it. A chest marked "don't touch" is never moved from.

## Acceptance criteria

- [ ] Owners assign items or groups to chests from chat and the panel
- [ ] Unassigned items go to the loose chest, or are reported when there is none
- [ ] Sorting moves misplaced items to their chests while idle at home, never from a don't-touch chest
