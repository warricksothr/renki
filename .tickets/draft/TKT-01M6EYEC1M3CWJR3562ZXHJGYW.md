---
schema: 4
id: TKT-01M6EYEC1M3CWJR3562ZXHJGYW
title: "Loot: anchor on the followed or commanding player, not any owner"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - controller
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T20:15:36Z
updated_at: 2026-11-02T20:15:36Z
created_by:
  id: agent:claude-code/renki-wave1
  name: ""
updated_by:
  id: agent:claude-code/renki-wave1
  name: ""
extensions: {}
---

## Description

The `Loot` instance (TKT-01M6EFHE2N, shipped in #165) is on by default and picks up drops within 30 m of *any* live owner it can see (`liveOwners` in `internal/behaviour/loot.go`). The owner wants it tied to one person: the player Tas is with.

### What to build

- Loot anchors on one player: the one Tas is following, or failing that the last player who gave Tas an order. That player must be an owner, or hold the `command` permission (an `operator` profile, `internal/access/access.go`). Companions and guests never anchor loot.
- Drops near other owners, outside the anchor's area, are left alone.
- With no anchor in sight, loot does nothing.
- The trace and the panel name the anchor player.

### Files

- `internal/behaviour/loot.go` (`liveOwners`, the eligibility walk), `internal/behaviour/engine.go` (`View`: who is being followed, who commands), `internal/access/access.go` (`Has`)
- `internal/session` for the current follow target
- `docs/BEHAVIOURS.md` (the loot paragraph)

### Verification

- Tests: two owners in sight, Tas following one, so only drops near the followed one are taken. An operator anchors; a companion does not. No anchor, no run.
- `just check controller test-packages ./internal/behaviour`, then `just check ci`.

## Acceptance criteria

- [ ] Loot takes only drops in the anchor player's area, the anchor being the followed player or the last commander, and an owner or operator
- [ ] Companions and guests never anchor loot, and with no anchor in sight loot does not run, covered by tests
- [ ] docs/BEHAVIOURS.md describes the anchor; just check ci passes
