---
schema: 4
id: TKT-01M6CEA0PN12E5P815YWE2TQZW
title: Dig trenches to a specified corridor, width and depth
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - controller
  - movement
  - safety
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

Teach Tas to excavate a trench from a selected polyline/corridor with width, depth or invert elevation, side slopes and endpoint requirements. Survey current terrain and reachable native digging limits first; report bedrock, terrain deformation limits, separate rock obstacles or water conflicts when the requested cross-section cannot be achieved.

### Execution
Choose an excavation sequence and standing positions that preserve access and an exit; keep spoil drops, carrying capacity, tool repair and combat pressure in the project flow. Exclude foundations, structures, utilities and protected zones unless the project explicitly covers them. The default behavior must not assume a drained or dry trench is possible. Progress is the observed trench profile and usable access, not a count of swings.

## Acceptance criteria

- [ ] Preview trench footprint, target profile, constraints, standing positions and escape/access route.
- [ ] Excavate within the authorized footprint using normal tools and measured terrain changes.
- [ ] Pause or revise for deformation limits, water, bedrock, structures, danger or exhausted supplies/tools.
- [ ] Complete only when sampled width/depth/profile and access meet declared tolerances; retain a resumable remaining-work ledger.
