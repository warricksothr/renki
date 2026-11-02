---
schema: 4
id: TKT-01M6DCFSHHG0DZJW21HTZGZGH4
title: "Memory: durable memory.json writes, rollback and world identity"
type: bug
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - controller
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

The home store (memory.json) backs nearly every planning decision. It is less durable than the newer goal and atlas stores.

### 1. No fsync, and a damaged file blocks startup (medium)

`internal/home/store.go` `save` (around :244-258) does `os.WriteFile(tmp)` then `os.Rename`, with no file sync and no directory sync. The landmark batch rewrites the whole file every 30 s, and `SetInterrupted` rewrites it on every task start and end. After a VM hard reset or power cut, memory.json can be empty or garbage. `json.Unmarshal` then fails and `cmd/controller/main.go:86-90` exits with code 2.

`internal/goal` already does this properly (`CreateTemp` + `Sync` + `replaceFile`, write-through `MoveFileEx` on Windows). `internal/settings` has the same weak pattern as the home store.

Fix: move goal's save helper into a small shared package and use it from home and settings. Keep the last good file as `.bak` and fall back to it, loudly, when the main file fails to parse.

### 2. Older setters change memory before the save succeeds (low)

`SetHome`, `SetInterrupted`, `UpdateLeftBehind`, `Shun`/`Unshun`, `AddGrave` and `ForgetGrave` mutate `s.mem` before `save()` and do not roll back on error. Newer setters do roll back. Fix: snapshot and restore, as the newer methods do.

### 3. World identity (low; plausible)

- Homes, containers, landmarks, waypoints and danger are keyed by world *name*. docs/ATLAS.md uses UID + epoch, and pickups already record `WorldUID`.
- Two servers named "Dedicated", or a restored save, share homes and chests.
- memory.json has no schema `version`.
- The `interrupted` flag is one global boolean (`store.go:52`, `home.go:351-357`), so an outage in world A sends the bot "home" in world B.

Fix: add `version`, store the world with the interrupted flag, and key by `worldUid` when the state provides it, migrating entries keyed by name. Related: TKT-01M62EP0 (Controller: remember where things are and should be, per world).

## Acceptance criteria

- [ ] memory.json and settings writes fsync the file and directory and replace it write-through, using one shared helper
- [ ] A corrupt memory.json falls back to the last good copy with a loud log instead of exiting
- [ ] Every home store setter rolls back on a failed save
- [ ] The interrupted flag is per world, and memory.json carries a schema version
