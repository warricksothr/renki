---
schema: 4
id: TKT-01M66QF9G576JFCQ0JBP3RWR2E
title: Persist a bounded world atlas with merge and expiry
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
  - TKT-01M66QF9EH8W2EG78AKKKF05PP
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

Implement the local atlas.sqlite store proposed in docs/ATLAS.md after capture identity/freshness is available. Use canonical world/epoch/layer/resolution/global-grid sample keys, a current materialized view and bounded changed revisions. Keep memory.json and the live session cache separate. Use transactional bounded batches, migrations, honest observation/confirmation/ingestion times, local WAL, recovery and explicit age/record/byte budgets. Benchmark storage overhead before selecting defaults.

## Acceptance criteria

- [ ] Restart restores accumulated terrain; newer compatible known observations replace only measured cells and unknown samples preserve old ages
- [ ] Overlapping borders share canonical keys; equal fresh confirmations update age without duplicate revisions
- [ ] Configurable stale/expiry/history/quota policies omit expired current data without resurrecting history and bound logical and physical storage
- [ ] Replay and recovery checks cover negative coordinates, duplicates/reordering, cross-world/epoch isolation, clock anomalies, crash/save/migration failure and maintenance
