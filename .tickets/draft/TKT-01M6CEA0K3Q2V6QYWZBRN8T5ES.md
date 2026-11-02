---
schema: 4
id: TKT-01M6CEA0K3Q2V6QYWZBRN8T5ES
title: Terrain shaping, paths and construction projects
type: epic
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - plugin
  - movement
  - safety
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
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

Teach Tas to carry out owner-specified trenching, leveling, grading, physical path construction and building projects. The user requested these backlog tickets on 2026-11-01; this is future capability work, separate from finishing current M2/M3 food, hauling and defence acceptance.

### Project model
Represent work with a world-scoped footprint or corridor, target elevations or blueprint, materials/tool requirements, tolerances and a finite work budget. Offer the same plan and progress evidence to human operators in the panel and agent tools. Distinguish planned work, measured effects, remaining work and blocked prerequisites. Compose durable project goals from bounded C# actions rather than puppeting individual inputs from a model.

### Boundaries
Use ordinary game permissions, tools, stamina, durability and resource consumption. Terrain modification and construction need explicit project authorization scoped to the selected area. Existing structures, protected areas, access routes, water and nearby players are constraints to inspect, not assumptions to overwrite. Pause for threats, injury, supply/repair needs or changed plans; a fresh observation and native receipt must establish progress. Reconnects must reconcile completed work before resuming. No creative spawning, direct save editing or promise that terrain edits are reversible.

## Acceptance criteria

- [ ] Linked child tickets cover shared execution, trenching, leveling, grading, physical paths and building.
- [ ] Human and agent operators can inspect a bounded project, its requirements, progress and reasons for stopping.
- [ ] Each capability is verified in an isolated test area with measured world effects and interruption coverage.
