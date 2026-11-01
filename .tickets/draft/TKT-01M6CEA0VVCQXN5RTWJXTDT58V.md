---
schema: 4
id: TKT-01M6CEA0VVCQXN5RTWJXTDT58V
title: Create physical paths between markers with a chosen surface
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
  - TKT-01M6CEA0SSA8HG7VJBVJKB5AAD
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

Teach Tas to construct and maintain an actual path or road between selected markers or along an operator-edited corridor. This modifies the world; it is separate from choosing a navigation route. Survey candidate corridors, protect buildings/resources/exclusions, and preview width, grade, surface and tie-ins before work.

### Scope
Start with game-supported hoe paths and paved surfaces whose recipes and availability are read from the running game. Compose grading and surface placement with normal tool/material requirements. Report missing bridges, retaining structures or impassable water as explicit construction needs, rather than silently inventing crossings. Reuse the durable project ledger for material resupply and repair. Preserve route information at markers for later traversal and hauling.

## Acceptance criteria

- [ ] Preview a bounded corridor connecting the selected endpoints, width, surface and cost/prerequisites.
- [ ] Compose grading and native path/surface actions without editing outside authorized bounds.
- [ ] Observe constructed surface continuity and traverse both directions before declaring the path usable.
- [ ] Report infeasible crossings, unmet construction needs and remaining segments; resume safely after interruption.
