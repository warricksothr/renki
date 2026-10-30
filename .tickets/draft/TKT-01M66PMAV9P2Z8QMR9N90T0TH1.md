---
schema: 4
id: TKT-01M66PMAV9P2Z8QMR9N90T0TH1
title: Show and control exploration progress in the panel
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - chat
assignees: []
milestone: null
parent: TKT-01M66PFAYA1NTW2JHNRDKBK42E
origin: null
dependencies:
  - TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
  - TKT-01M62N2557R401S3E6JECSAPN4
  - TKT-01M66PMASYEM3Z9KJMB2979CEM
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

After M2/M3 and a working exploration host, expose mission start/stop/resume and progress using existing role/global/native permissions, as proposed in docs/EXPLORATION.md. Show authorized area/return anchor, map/observed/visited coverage, planned versus validated routes, frontiers/failures, discoveries and budgets. Add addressed exploration requests and read-only discovery summaries only for implemented capabilities. This ticket remains draft.

## Acceptance criteria

- [ ] Panel distinguishes planned travel, observed/visited coverage, route uncertainty and last-known discoveries with age/provenance
- [ ] Authorized start/change/stop/resume respects scope, waypoint versions and host authority; locate-only players cannot start travel
- [ ] Compact progress/diagnostic summaries and clear partial results are validated without exposing raw streams to a planner
