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
milestone: M7
parent: TKT-01M66PFAYA1NTW2JHNRDKBK42E
origin: null
dependencies:
  - TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
  - TKT-01M62N2557R401S3E6JECSAPN4
  - TKT-01M62VXZ0J9Q2WH4XN4E6N7F27
  - TKT-01M66QF9G576JFCQ0JBP3RWR2E
  - TKT-01M6EFQNMW1J01MX78QAB91CZV
  - TKT-01M6EFQNR1F93A8ZYP95G4H2CP
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-30T15:25:07Z
updated_at: 2026-11-02T15:59:32Z
created_by:
  id: agent:codex/renki-exploration
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m7
  name: ""
extensions: {}
---

## Description

Implement the frontier selection and failed-approach memory proposed in docs/EXPLORATION.md after M2/M3. Consume coverage with provenance, estimate information gain and outbound/return cost, account for observed terrain/danger/exclusions, and keep unknown costs explicit. Retain bounded scoped failure evidence; retry only after meaningful change or bounded cooldown/revalidation. This ticket remains draft.

### Split (rollout step 2)
This is the umbrella for two one-PR children:
- TKT-01M6EFQNMW (Rank exploration frontiers from measured coverage): pure ranking in a new internal/explore package with recorded scenes
- TKT-01M6EFQNR1 (Remember failed frontier approaches with bounded revalidation): failure memory and the replay scenes this ticket's third criterion names

### Why it matters to M7
Goal 7.2 (frontier choice is honest in replay) is measured here.

### Files
- internal/explore/ (new), internal/atlas/store.go, internal/memory/danger.go, internal/goal/areas.go, internal/goal/store.go

### Verification
`just check controller test-packages ./internal/explore ./internal/goal` and `just check ci`. Done when both children are done.

## Acceptance criteria

- [ ] Rank bounded frontier candidates using observation gain, terrain/danger constraints and return cost without assuming unknown land is safe
- [ ] Persist scoped failed approaches with reason/age/revalidation criteria; repeated observations alone do not enable identical retries
- [ ] Replay tests cover no safe frontier, shared-map-only knowledge, stale coverage, alternative approaches and bounded memory

## Notes

**agent:codex/renki-atlas** at 2026-10-30T15:40:14Z

Durable terrain/current-state history is tracked by TKT-01M66QF9G576JFCQ0JBP3RWR2E — Persist a bounded world atlas with merge and expiry. Added that store as a dependency so frontier planning can reference accumulated evidence with explicit age/provenance instead of treating the 256-tile live cache as durable coverage. M2/M3 and coverage dependencies remain; this ticket is still draft. See docs/ATLAS.md.
