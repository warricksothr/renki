---
schema: 4
id: TKT-01M6AZ7DJYGTWDR9G9ZP0MA20M
title: Traverse known portals with verified arrivals
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - movement
  - controller
assignees: []
milestone: null
parent: null
origin: null
dependencies:
  - TKT-01M6AYP8TQRJCX0ZAQEY0GBDCV
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-01T07:12:19Z
updated_at: 2026-11-01T07:12:59Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

Build on observed per-world portal links as explicit route transitions. Revalidate the exact link and native inventory/server restrictions, use normal approach/entry controls, verify teleport completion and actual arrival, clear local geometry after transport and expose changed-link, forbidden-cargo, unknown-destination, cancellation and arrival-timeout outcomes. Track observed exits and return-route knowledge without treating remote portal coordinates as safe landing proof. Preserve cargo such as owner-reserved ore. Deliberate entry needs a narrowly scoped exception to the final native trigger guard, never a global bypass.

## Acceptance criteria

- [ ] Portal travel is an explicit scoped transition distinct from walking and tag editing
- [ ] Link changes, native item/server restrictions and cancellation cannot cause unintended travel
- [ ] Arrival location and return knowledge are observed and persisted separately from source link metadata
- [ ] Repeatable native tests verify traversal and restoration without spawned supplies
