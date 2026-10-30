---
schema: 4
id: TKT-01M66QF9HTVA5EEF2ASPEQYA65
title: Show saved and stale atlas layers in the map
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
assignees: []
milestone: null
parent: TKT-01M66Q5SFWV99JCGJXVARVRJH7
origin: null
dependencies:
  - TKT-01M66QF9G576JFCQ0JBP3RWR2E
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

Expose bounded saved-atlas viewport/overview queries and panel rendering as proposed in docs/ATLAS.md. Accumulate terrain beyond the hot cache, allow explicit saved-world selection while offline and show observation age, source and resolution. Distinguish live connection, remembered/stale, imported and unknown evidence. Keep current canvas initially; persisted backdrop does not establish observed coverage, walkability or route safety.

## Acceptance criteria

- [ ] Panel restores accumulated terrain across reconnect/restart and can select a saved world while offline
- [ ] Hover/styling communicates age, provenance, resolution and stale/unknown status without presenting saved data as live
- [ ] Viewport/result budgets and world/connection changes during requests are checked; live task scope remains separate from saved-world selection
