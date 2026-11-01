---
schema: 4
id: TKT-01M6CEA0RJVVWTWBAB69AQS0B0
title: Level a selected footprint to an explicit elevation
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - controller
  - movement
assignees: []
milestone: null
parent: TKT-01M6CEA0K3Q2V6QYWZBRN8T5ES
origin: null
dependencies:
  - TKT-01M6CEA0MRCXAD97ZXSK43H3EG
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-01T20:55:07Z
updated_at: 2026-11-01T20:55:07Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

Teach Tas to cut and fill a bounded pad or footprint to an explicit surveyed elevation. The datum must be part of the plan rather than implicitly following the bot's feet on every hoe action. Inspect the native leveling, digging and raising rules and use actual terrain samples to choose successive actions.

### Constraints
Estimate cut/fill and required stone/tool/workbench conditions, then account for actual consumed resources. Keep edges and protected exclusions intact; reject unreachable elevations or deformation limits rather than claiming convergence. Preserve safe standing positions and access while working around slopes and existing foundations. Re-survey after interruptions and after nearby players change terrain.

## Acceptance criteria

- [ ] Preview footprint, datum, cut/fill estimates, material needs and height tolerance.
- [ ] Use normal digging/hoe/raising actions with resource and durability receipts.
- [ ] Verify a sampled finished pad against the fixed datum and check outside-area/exclusion preservation.
- [ ] Handle native height limits, changed ground, missing materials, danger and interrupted partial work without blind repetition.
