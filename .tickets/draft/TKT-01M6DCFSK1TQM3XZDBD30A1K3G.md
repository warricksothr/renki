---
schema: 4
id: TKT-01M6DCFSK1TQM3XZDBD30A1K3G
title: "Atlas: drop permanent batch failures instead of halting recording"
type: bug
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
assignees: []
milestone: M4
parent: TKT-01M6DCF2ABDDNG3QF369NDPBT5
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T05:42:34Z
updated_at: 2026-11-02T06:10:52Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-review
  name: ""
extensions: {}
---

## Description

One terrain batch that can never succeed stops atlas recording for every world.

### Where

`internal/atlas/recorder.go:27-60` together with `store.go:166-168`.

### Failure

- A failed batch stays at `pending[0]`, and `DrainTerrain` is skipped while anything is pending, so permanent errors are retried every second. Two examples are the validation failure and "world generator changed; select a new atlas epoch".
- After a Valheim update that changes the generator version, recording stops for that world and for every world queued behind it.
- The 2048-tile session queue overflows (logged every second), and `Maintain` runs every second.

### Fix

- Have `Ingest` return a typed permanent error.
- The recorder drops and logs those batches, or sets them aside counted, and keeps draining.
- Surface the condition in panel status.
- The recorder has no tests. Add some covering transient retry, permanent drop, and one world's failure not blocking another's.

## Acceptance criteria

- [ ] A permanently failing terrain batch is dropped or set aside and other batches keep recording
- [ ] The recorder has tests for transient retry, permanent drop and per-world isolation
