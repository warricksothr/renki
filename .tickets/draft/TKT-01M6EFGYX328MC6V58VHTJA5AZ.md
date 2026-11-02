---
schema: 4
id: TKT-01M6EFGYX328MC6V58VHTJA5AZ
title: "Chat: bind owners to player IDs, confirmed from the panel"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - chat
  - safety
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
created_at: 2026-11-02T15:54:52Z
updated_at: 2026-11-02T15:58:03Z
created_by:
  id: agent:claude-code/renki-plan-m4
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m4
  name: ""
extensions: {}
---

## Description

Goal 4.4 says owners bind to player IDs. Authority today is `slices.Contains(owners, c.From.Name)` (`internal/chat/brain.go` around :321 and :340, `internal/chat/gather.go:16`, `internal/chat/waypoints.go:35`, `internal/access/access.go:40`), and Valheim character names are local and not unique, so anyone can copy an owner's name. Split out of TKT-01M6DCFSME (Chat: fence untrusted lines in owner turns, stop naming owners), item 2.

### What to build

- Bind each owner to the `PlayerID` that `protocol.ChatSender` already carries (`internal/protocol/messages.go`, `ChatSender.PlayerID`).
- Choose one of the two approaches the review offered, and record which here: learn the ID on first sight and let the panel confirm or clear it, or require it in config. Either way, a name match with a different player ID is refused.
- `owners` is an owners-only setting in `internal/settings/settings.go`; the bound IDs belong beside it, editable only by owners and the panel, never by the agent.
- The panel's Settings section shows each owner with their bound ID and can clear it.
- Update docs/PLAYERS.md.

### Files

- `internal/access/access.go`, `internal/chat/brain.go`, `internal/chat/gather.go`, `internal/chat/waypoints.go`
- `internal/settings/settings.go`
- `internal/app` (where settings reach the panel), `panel/src` Settings section
- `docs/PLAYERS.md`

### Verification

- `just check controller test-packages ./internal/access ./internal/chat ./internal/settings ./internal/app`.
- A test: a chat line from an owner's name with another player ID cannot issue a command.
- `just check panel test` and `just check ci`.

## Acceptance criteria

- [ ] Each owner is bound to a player ID, and a name match with another ID is refused, covered by a test
- [ ] The panel shows each owner's bound ID and can clear it; the agent cannot change it
- [ ] docs/PLAYERS.md describes how owners are bound
