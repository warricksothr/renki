---
schema: 4
id: TKT-01M6EFGYQRRP7W4XEBZW8M3ED3
title: "Memory: key by world UID, per-world interrupted flag, schema version"
type: task
status: draft
status_reason: null
priority: low
due_on: null
labels:
  - controller
assignees: []
milestone: M5
parent: TKT-01M6DCF2ABDDNG3QF369NDPBT5
origin: null
dependencies:
  - TKT-01M6DCFSHHG0DZJW21HTZGZGH4
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:54:52Z
updated_at: 2026-11-02T16:15:35Z
created_by:
  id: agent:claude-code/renki-plan-m4
  name: ""
updated_by:
  id: agent:claude-code/renki-plan
  name: ""
extensions: {}
---

## Description

The world-identity part of TKT-01M6DCFSHH (Memory: durable memory.json and settings writes, with rollback), item 3, split out because it is a migration on its own and goal 4.3 does not need it. The reviewer rated it low and plausible.

### Problem

- Homes, containers, landmarks, waypoints and danger in `internal/home` are keyed by world *name*. docs/ATLAS.md uses world UID plus epoch, and pickups already record `WorldUID`.
- Two servers named "Dedicated", or a restored save, share homes and chests.
- memory.json has no schema `version`.
- The `interrupted` flag is one global boolean (`internal/home/store.go` around :52, `internal/home/home.go` around :351-357), so an outage in world A sends the bot "home" in world B.

### Fix

Add `version` to memory.json, store the world with the interrupted flag, and key by `worldUid` when the state provides it, migrating entries keyed by name. Related: TKT-01M62EP0PD (Controller: remember where things are and should be, per world), which is in progress and now in M3. Coordinate with it before starting, because both edit the per-world maps in `internal/home`.

### Files

- `internal/home/store.go`, `internal/home/home.go`, and the per-kind files beside them (`containers.go`, `landmarks.go`, `waypoints.go`, `danger.go`)
- their `_test.go` files
- docs/ATLAS.md or the home store's doc, wherever memory.json's shape is described

### Verification

- `just check controller test-packages ./internal/home ./internal/app`.
- A test loads a memory.json without `version` keyed by world name and migrates it.
- `just check ci`.

## Acceptance criteria

- [ ] memory.json carries a schema version
- [ ] The interrupted flag is stored per world
- [ ] Homes, containers, landmarks, waypoints and danger key by world UID when known, and name-keyed entries migrate, covered by a test

## Notes

**agent:claude-code/renki-plan** at 2026-11-02T16:15:35Z

Applied on 2026-11-02 by the planning pass: milestone set to M5, to match the exit goal in docs/ROADMAP.md this ticket serves. This supersedes any earlier note that only proposed the move. The owner can move it back.
