---
schema: 4
id: TKT-01M66QF9EH8W2EG78AKKKF05PP
title: Report stable world identity and truthful terrain freshness
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
parent: TKT-01M66Q5SFWV99JCGJXVARVRJH7
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-30T15:39:51Z
updated_at: 2026-10-30T15:39:51Z
created_by:
  id: agent:codex/renki-atlas
  name: ""
updated_by:
  id: agent:codex/renki-atlas
  name: ""
extensions: {}
---

## Description

Extend the terrain bridge as proposed in docs/ATLAS.md. Report the game world UID as a decimal string, configured namespace/atlas epoch and compatibility metadata. Distinguish fresh per-sample measurements from cached carried values, and send bounded confirmations for genuinely remeasured unchanged values. Preserve the current sampling/frame budgets and older-producer session-only behavior. Do not generate unexplored world data from a seed.

## Acceptance criteria

- [ ] Protocol fixtures preserve 64-bit world identity and distinguish worlds with the same display name and explicit epochs
- [ ] Per-sample freshness/confirmation cannot be renewed by unloaded carried samples, unchanged resends or an adjacent changed corner
- [ ] Focused checks cover partial tiles, identical fresh resampling, reconnect/reordering and bounded confirmation/sampling work
