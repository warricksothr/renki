---
schema: 4
id: TKT-01M6CEA0SSA8HG7VJBVJKB5AAD
title: Grade terrain to walkable slopes and smooth transitions
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

Teach Tas to create controlled sloped terrain instead of requiring flat ground. Accept a target surface or corridor profile with maximum longitudinal slope, cross-slope, step height, width and tie-in elevations. Survey roughness and native terrain granularity before deciding what can be achieved.

### Execution and proof
Compute bounded cut/fill actions and protected shoulders or exclusions. Sequence them with reachable standing locations and access to supplies. Re-observe ground changes and invalidate local navigation/collision knowledge. Validate physical traversability in both directions and measure the resulting profile; a clear map line or navmesh path alone does not prove a usable grade. Expose remaining steep/rough sections and native-limit failures to the operator.

## Acceptance criteria

- [ ] Preview requested surface/profile, slope and step limits, cut/fill and tie-in constraints.
- [ ] Apply bounded normal terrain actions and measure convergence to the requested grade.
- [ ] Validate finished slope/roughness and physical traversal with preserved protected boundaries.
- [ ] Stop or revise where native terrain limits, hazards or required resources make the profile infeasible.
