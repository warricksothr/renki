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
updated_at: 2026-11-02T15:56:49Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m4
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

### Goal

Goal 4.2 (24 h soak): the soak fails on a store write error and on audit growth over 2 GB a day, so the audit must report its own failures and stay bounded. TKT-01M6EFGYV6 (the soak run) depends on this.

### Files

Line numbers refer to commit f1a5483.

- `internal/audit/audit.go` (`Write` errors around :255, `Dropped` and `Missed`, retention around :335-364), `internal/audit/timeline.go`, `internal/audit/audit_test.go`, `internal/audit/timeline_test.go`
- `internal/panel/panel.go` (`days`, `timeline`, `task` handlers), `internal/app/panel.go` (`PanelStatus`)
- `internal/config/config.go` (a new `audit.maxBytes`)
- `panel/src` where the status header shows health, if the counts appear there

### Verification

- The day-file size is read on the VM without the game: `just vm shell`, then the size of the newest file in the controller's audit directory. Record it in a note. This is read-only and needs no lease.
- `just check controller test-packages ./internal/audit ./internal/panel ./internal/app ./internal/config`, with a test that a failing writer shows up in `PanelStatus` and a test for oldest-first pruning by size.
- `just check panel test` if the panel changes, and `just check ci`.

## Acceptance criteria

- [ ] Audit write failures appear in the console and panel status
- [ ] A measured day-file size on the VM is recorded
- [ ] The audit has a byte cap or thinned state frames, and timeline requests do not decode whole days
