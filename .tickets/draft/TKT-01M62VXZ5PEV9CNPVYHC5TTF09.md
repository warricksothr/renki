---
schema: 4
id: TKT-01M62VXZ5PEV9CNPVYHC5TTF09
title: "Controller: fuel template: keep fires and furnaces going"
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

A template on the behaviour engine. Parameters: which fires and machines
(those in the base by default), what to burn (wood for fires, resin for
torches, coal for furnaces), when (cold, night, someone near, a cooking
station in use over it, a furnace with something queued), and a reserve of
fuel that stoking never uses up.

Fuel comes from the bot's pack first, then the base chests (withdraw). A
furnace with nothing queued gets no coal.

## Acceptance criteria

- [ ] Fires in the base are kept lit when cold or at night, and when food is cooking over them
- [ ] Furnaces are fuelled only while they have something queued
- [ ] Fuel comes from the pack, then the chests, and never below the reserve
- [ ] When there is no fuel, the owners are told once
