---
schema: 4
id: TKT-01M66QF9G576JFCQ0JBP3RWR2E
title: Persist a bounded world atlas with merge and expiry
type: task
status: done
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
updated_at: 2026-10-30T16:04:16Z
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

- [x] Restart restores accumulated terrain; newer compatible known observations replace only measured cells and unknown samples preserve old ages
- [x] Overlapping borders share canonical keys; equal fresh confirmations update age without duplicate revisions
- [x] Configurable stale/expiry/history/quota policies omit expired current data without resurrecting history and bound logical and physical storage
- [x] Replay and recovery checks cover negative coordinates, duplicates/reordering, cross-world/epoch isolation, clock anomalies, crash/save/migration failure and maintenance

## Implementation plan

Use pure-Go SQLite for Windows-compatible controller builds. Add a dedicated atlas store with versioned schema, WAL, one connection/writer, indexed canonical sample keys, provenance and stream timestamps, bounded changed revisions and configurable TTL/quotas. Ingest immutable accepted terrain batches from the session using a bounded recorder that retains failed batches and flushes at shutdown. Read bounded saved-world/viewports with per-sample ages and unknown holes; verify crash/reopen, border reconciliation, legacy exclusion, expiry/non-resurrection and idempotence.

## Notes

**agent:codex/renki-atlas** at 2026-10-30T16:04:16Z

Implemented SQLite schema 1 with canonical global samples, hashes, bounded history, producer-stream manifests, idempotent resends, partial merges, TTL/read filtering, quotas, WAL checkpoints and incremental compaction. The bounded recorder retains failed batches and attempts final flush. Tests cover reopen, negative/border coordinates, same-name worlds/epochs, invalid/incompatible batches, rollback/future schema and storage limits. just check ci passed. just check controller benchmark-atlas measured about 403 bytes/sample and 7.6 ms per 25-sample durable batch before the pruning optimization. Live deployment/reconnect proof follows after merge.

## Summary

Implemented SQLite schema 1 with canonical global samples, hashes, bounded history, producer-stream manifests, idempotent resends, partial merges, TTL/read filtering, quotas, WAL checkpoints and incremental compaction. The bounded recorder retains failed batches and attempts final flush. Tests cover reopen, negative/border coordinates, same-name worlds/epochs, invalid/incompatible batches, rollback/future schema and storage limits. just check ci passed. just check controller benchmark-atlas measured about 403 bytes/sample and 7.6 ms per 25-sample durable batch before the pruning optimization.
