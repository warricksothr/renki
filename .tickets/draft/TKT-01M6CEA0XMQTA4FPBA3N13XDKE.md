---
schema: 4
id: TKT-01M6CEA0XMQTA4FPBA3N13XDKE
title: Build from bounded blueprints with verified placement and support
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - controller
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

Teach Tas to build a small authorized structure from an inspectable blueprint or ordered piece list anchored to a surveyed location. Read buildable pieces, recipes and current game/mod constraints; do not hard-code creative availability. The first acceptance project should be a small floor/platform or shelter in an isolated test plot.

### Planning and execution
Preview piece transforms, snap relationships, footprint, supporting ground and material/workbench/hammer requirements. Sequence foundational/supporting pieces before dependent pieces, respecting wards, collision, other players and reserved access. Normal native placement must validate orientation, snapping, resources and structural support. Missing materials or repair needs should yield a visible project prerequisite, using existing supply/storage goals rather than spawning items.

### Durable proof
Track placed pieces with stable world identities and observed transforms/support. Reconcile player alterations, failed placements and uncertain interrupted attempts so resuming cannot duplicate pieces. Removing existing structures is a separate explicitly authorized operation; failed work must not auto-demolish the base. Share blueprint edits and live remaining-work diagnostics between panel and agent tools.

## Acceptance criteria

- [ ] Preview a finite anchored blueprint, piece order, support and recipe/tool requirements.
- [ ] Place supported pieces through normal game checks with material and world-object receipts.
- [ ] Verify transforms, snapping, structural support and required usable shelter/platform properties.
- [ ] Reconcile edited, missing or uncertain pieces across interruptions without duplicate placement or unrequested demolition.
- [ ] Expose actionable supply/repair/support blockers and pass an isolated repeatable construction check.
