---
schema: 4
id: TKT-01M6DCFSHHG0DZJW21HTZGZGH4
title: "Memory: durable memory.json and settings writes, with rollback"
type: bug
status: draft
status_reason: null
priority: high
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
updated_at: 2026-11-02T15:57:56Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m4
  name: ""
extensions: {}
---

## Description

The home store (memory.json) backs nearly every planning decision. It is less durable than the newer goal and atlas stores.

### 1. No fsync, and a damaged file blocks startup (medium)

`internal/home/store.go` `save` (around :244-258) does `os.WriteFile(tmp)` then `os.Rename`, with no file sync and no directory sync. The landmark batch rewrites the whole file every 30 s, and `SetInterrupted` rewrites it on every task start and end. After a VM hard reset or power cut, memory.json can be empty or garbage. `json.Unmarshal` then fails and `cmd/controller/main.go:86-90` exits with code 2.

`internal/goal` already does this properly (`CreateTemp` + `Sync` + `replaceFile`, write-through `MoveFileEx` on Windows). `internal/settings` has the same weak pattern as the home store.

Fix: move goal's save helper into a small shared package and use it from home and settings. Keep the last good file as `.bak` and fall back to it, loudly, when the main file fails to parse.

### 2. Older setters change memory before the save succeeds (low)

`SetHome`, `SetInterrupted`, `UpdateLeftBehind`, `Shun`/`Unshun`, `AddGrave` and `ForgetGrave` mutate `s.mem` before `save()` and do not roll back on error. Newer setters do roll back. Fix: snapshot and restore, as the newer methods do.

### 3. World identity (low; plausible)

Moved to TKT-01M6EFGYQR (Memory: key by world UID, per-world interrupted flag, schema version).

### Goal

Goal 4.3 (stores survive a hard stop): 3 forced controller kills during saves leave memory.json, goals.json and settings readable, with no lost home. Item 3 (world identity) moved to TKT-01M6EFGYQR (Memory: key by world UID, per-world interrupted flag, schema version); goal 4.3 does not need it. The live kills are in TKT-01M6EFGYV6 (VM: run the 24 h soak and link its summary from the M4 epic).

### Files

Line numbers refer to commit f1a5483; at 0b932ef the weak save is `internal/home/store.go:255-258` and `internal/settings/settings.go:424-427`.

- `internal/goal/store.go`, `internal/goal/replace_unix.go`, `internal/goal/replace_windows.go` (the helper to move into a small shared package)
- `internal/home/store.go` (`save`, and the setters `SetHome`, `SetInterrupted`, `UpdateLeftBehind`, `Shun`, `Unshun`, `AddGrave`, `ForgetGrave`)
- `internal/settings/settings.go`
- `cmd/controller/main.go` (:86-90, the exit on a bad memory.json)

### Verification

- `just check controller test-packages ./internal/home ./internal/settings ./internal/goal` plus the new shared package.
- A test that runs a helper process saving in a loop, kills it 3 times mid-save, and asserts memory.json, goals.json and settings each parse afterwards with the home present (or fall back to `.bak` with a logged warning).
- A test per setter that a failed save leaves memory unchanged.
- `just check ci`.

## Acceptance criteria

- [ ] memory.json and settings writes fsync the file and directory and replace it write-through, using one shared helper
- [ ] A corrupt memory.json falls back to the last good copy with a loud log instead of exiting
- [ ] Every home store setter rolls back on a failed save
- [ ] A test kills a saving controller process 3 times, and memory.json, goals.json and settings each load afterwards with the home present
