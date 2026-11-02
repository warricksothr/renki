---
schema: 4
id: TKT-01M6DCFSNTBFQYW1M4ZYEERSWC
title: "Audit: report write failures and bound the log by size"
type: bug
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - tooling
assignees: []
milestone: null
parent: TKT-01M6DCF2ABDDNG3QF369NDPBT5
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T05:42:34Z
updated_at: 2026-11-02T05:42:34Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-review
  name: ""
extensions: {}
---

## Description

The audit log is the main tool for working out what the bot did, yet it can fail silently and grow without limit.

### 1. Write failures are counted and never shown

`internal/audit/audit.go:255,271`: `Write` increments `l.errs`, and `Dropped()`/`Missed()` have no callers. When the disk is full or the file cannot be written, the audit stops while the live panel still looks healthy.

Fix: log the first failure and every Nth one to the console, not to the audit. Show the dropped and missed counts in `PanelStatus`.

### 2. Retention is by age only

`audit.go:335-364` keeps `keepDays` (14) and has no size cap. Every 5 Hz state frame, perception frame, terrain frame and trace is appended. `GET /api/timeline` and `GET /api/task` (`internal/panel/panel.go:507-593`, `audit/timeline.go:57`) decode whole day files on every request, and `task` reads up to three days.

Fix:
- Measure a real day file on the VM first, and record the size in a note.
- Then add `audit.maxBytes` with oldest-first pruning.
- Thin state frames in the file, for example to 1 Hz, as the panel's trail already does.
- Consider indexing or seeking for the timeline.

## Acceptance criteria

- [ ] Audit write failures appear in the console and panel status
- [ ] A measured day-file size on the VM is recorded
- [ ] The audit has a byte cap or thinned state frames, and timeline requests do not decode whole days
