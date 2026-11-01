---
schema: 4
id: TKT-01M6CEA0MRCXAD97ZXSK43H3EG
title: Add bounded earthwork and building actions with plan previews
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - controller
  - protocol
  - safety
assignees: []
milestone: null
parent: TKT-01M6CEA0K3Q2V6QYWZBRN8T5ES
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

Provide the shared execution and observation foundation for authorized terrain and construction projects. Inspect the installed game API and actual hoe, pickaxe and hammer actions, including normal placement validation, ownership/wards, tool quality, build-piece requirements and terrain edit limits. Document which operations use existing terrain versus separate mineable objects; do not treat every rock mesh as editable terrain.

### Proposed flow
Survey a live work footprint and its surroundings, pin the requested target and constraints, and produce an inspectable plan before mutation. A Go project goal owns checkpoints and prerequisites; bounded C# tasks perform normal tool/placement actions and return receipts tied to the actual changed terrain or placed structure. Expose target/actual height, affected bounds, remaining deviation, tool wear and material deltas through a shared panel/agent contract.

### Validation and interruption
Invalidate affected local collision/navigation/atlas samples after edits, while preserving authoritative stable-world identity and remote observations. Prevent operations outside the authorized area and protected exclusions, reserve an escape route, and stop on threats, injury, water/footing hazards, missing tools/materials or a changed plan. An interrupted action with uncertain effects must be re-observed, not blindly replayed. Run isolated-world fixtures through named just recipes; the first probes must not alter the shared server's base.

## Acceptance criteria

- [ ] Document supported native edit/placement actions, limits and prerequisites on the selected game/mod runtime.
- [ ] Provide a read-only plan preview and explicit work bounds shared by panel and agent tools.
- [ ] Execute a bounded authorized native action and report measured world/material/tool effects.
- [ ] Refuse invalid, protected, hazardous or out-of-bounds actions and reconcile uncertain work after interruption.
- [ ] Invalidate affected terrain/collision/navigation observations and provide repeatable isolated-area checks.
