---
schema: 4
id: TKT-01M62VXZ4D32W7WBX7WV84DAVY
title: "Plugin: add fuel and ore to machines and take their output"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - protocol
  - safety
assignees: []
milestone: null
parent: TKT-01M62VXZ2FWDX3D4Z6XX1MB5A6
origin: null
dependencies:
  - TKT-01M62PRDRKEHTQ1WC12FQ3S11J
  - TKT-01M62VXZ39Z020DAFQARB68K5M
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

Tasks to work a station as a player does, through the game's own calls so
wards still decide:

- `add_to`: put an item from the pack into a fire, smelter or kiln (fuel or
  ore), as many times as asked or until it is full
- `take_from`: take what is ready from a cooking station, fermenter or
  beehive

Both walk to the station first. They fail with `invalid_target`,
`inventory_full`, `missing_item` or `unreachable`. Safety gains a switch for
using stations.

## Acceptance criteria

- [ ] add_to puts fuel or ore from the pack into a fire, smelter or kiln, and reports how many
- [ ] take_from collects from cooking stations, fermenters and beehives
- [ ] Both respect wards and a Safety switch, and fail with the reasons 6.6 names
