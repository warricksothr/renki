---
schema: 4
id: TKT-01M62VXZ6ZWJFZ5X6JJX0WGDCX
title: "Controller: tend template: cooking, fermenters, beehives, ore"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
assignees: []
milestone: null
parent: TKT-01M62VXZ2FWDX3D4Z6XX1MB5A6
origin: null
dependencies:
  - TKT-01M62VXYSN4RQV75HTA3BSD162
  - TKT-01M62VXZ4D32W7WBX7WV84DAVY
  - TKT-01M62VXYWJHB5BC9VGCBM0PVXD
  - TKT-01M62VXYY6F094WXQS3FBQ57RF
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

A template on the behaviour engine for stations that hold their output or
need feeding:

- Take food off cooking stations before it burns. This is time-critical, so
  the instance gets a high priority.
- Collect from fermenters and beehives when they are ready.
- Bring ore or wood from the chests to the smelters and kilns that take it.

What it collects goes to `stow`.

## Acceptance criteria

- [ ] Cooked food comes off before it burns while the bot is in the base
- [ ] Ready fermenters and beehives are emptied and the goods stowed
- [ ] Ore in the chests is carried to a smelter that takes it
