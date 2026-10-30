---
schema: 4
id: TKT-01M66PMAQSBK2TH076X7M5FGAM
title: Add bounded expedition preparation, survey and return routines
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - plugin
  - movement
assignees: []
milestone: null
parent: TKT-01M66PFAYA1NTW2JHNRDKBK42E
origin: null
dependencies:
  - TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
  - TKT-01M62N2557R401S3E6JECSAPN4
  - TKT-01M62VXZ0J9Q2WH4XN4E6N7F27
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-30T15:25:07Z
updated_at: 2026-10-30T15:25:07Z
created_by:
  id: agent:codex/renki-exploration
  name: ""
updated_by:
  id: agent:codex/renki-exploration
  name: ""
extensions: {}
---

## Description

Implement the supplied land-expedition behaviors proposed in docs/EXPLORATION.md after M2/M3: validate supplies/gear/readiness and anchor, travel one bounded leg, report actual new survey evidence, reassess and return with a reserve. Native C# owns movement and immediate safety. Resolve waypoint versions explicitly and distinguish successful outbound traversal from a proven reversible return. Missing crafting, boats, portals and camp-building capabilities are separate work. This ticket remains draft.

## Acceptance criteria

- [ ] Prepare against explicit world/area/anchor/supply and native permission constraints, reporting missing prerequisites honestly
- [ ] Travel/survey reports actual scoped coverage and measured travel, handles supported elevation and rejects unsafe unsupported travel modes
- [ ] Return reserve and alternative/blocked return behavior are bounded, cancellable and tested without requiring player or enemy staging
