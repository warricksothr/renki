---
schema: 4
id: TKT-01M66PMASYEM3Z9KJMB2979CEM
title: Compose persistent exploration goals with discovery and return proof
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - movement
assignees: []
milestone: null
parent: TKT-01M66PFAYA1NTW2JHNRDKBK42E
origin: null
dependencies:
  - TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
  - TKT-01M62N2557R401S3E6JECSAPN4
  - TKT-01M66PMAPGXNJHB49E7WMGEBYS
  - TKT-01M66PMAQSBK2TH076X7M5FGAM
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

Extend the existing persistent goal host with a typed exploration specification and implemented child runners per docs/EXPLORATION.md after M2/M3. Persist scope, baseline coverage, selected/failed frontiers, original budgets and return state. The host, not a policy/model, validates useful new observations and supported arrival at the return anchor. Partial/needs-plan outcomes remain distinct from success. This ticket remains draft.

## Acceptance criteria

- [ ] Host enforces durable total time/travel/attempt/wait/area/biome limits and reserved return allowance across all runners
- [ ] Completion requires novel observed coverage plus fresh supported arrival evidence; repeated timestamps, movement success and shared map reveal alone cannot complete it
- [ ] Replay covers uncertain restart, missing telemetry, checkpoint failure, blocked return, owner stop/takeover and preserved budgets; a bounded land benchmark records actual discovery/return or an honest partial outcome
