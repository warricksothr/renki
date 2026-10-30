---
schema: 4
id: TKT-01M66PMAPGXNJHB49E7WMGEBYS
title: Select exploration frontiers and remember failed approaches
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

Implement the frontier selection and failed-approach memory proposed in docs/EXPLORATION.md after M2/M3. Consume coverage with provenance, estimate information gain and outbound/return cost, account for observed terrain/danger/exclusions, and keep unknown costs explicit. Retain bounded scoped failure evidence; retry only after meaningful change or bounded cooldown/revalidation. This ticket remains draft.

## Acceptance criteria

- [ ] Rank bounded frontier candidates using observation gain, terrain/danger constraints and return cost without assuming unknown land is safe
- [ ] Persist scoped failed approaches with reason/age/revalidation criteria; repeated observations alone do not enable identical retries
- [ ] Replay tests cover no safe frontier, shared-map-only knowledge, stale coverage, alternative approaches and bounded memory
