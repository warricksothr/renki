---
schema: 4
id: TKT-01M62VXZ39Z020DAFQARB68K5M
title: "Perception: fuel, lit, queued and finished for fires and machines"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - protocol
assignees: []
milestone: null
parent: TKT-01M62VXZ2FWDX3D4Z6XX1MB5A6
origin: null
dependencies:
  - TKT-01M62BMD18K3SE1WXJ7FDN8EPT
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

Perception (6.2) lists fires and smelters but not their state. Add, for:

- **Fires** (campfire, hearth, torches, braziers): the fuel and its maximum,
  and whether they are lit
- **Smelters, kilns, blast furnaces, charcoal kilns, the spinning wheel and
  the like:** the fuel and its maximum, what is queued and how much, and
  whether they are working
- **Cooking stations:** what is in each slot and whether it is cooked or
  burning
- **Fermenters and beehives:** whether they are ready

Read from the ZDO, as doors are. The protocol doc and fixtures are updated.
Scan time stays within budget.

## Acceptance criteria

- [ ] Fires report their fuel and whether they are lit
- [ ] Smelters and kilns report their fuel, their queue and whether they are working
- [ ] Cooking stations, fermenters and beehives report what is ready or burning
- [ ] 6.2 and its fixtures describe the new fields, and scans stay within 2 ms
