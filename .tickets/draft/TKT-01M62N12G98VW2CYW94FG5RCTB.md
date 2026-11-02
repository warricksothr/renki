---
schema: 4
id: TKT-01M62N12G98VW2CYW94FG5RCTB
title: "Controller: notice where owners work, and the drops piling up"
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
dependencies:
  - TKT-01M62EP0PDAB02DNJAJ6WS3MYD
  - TKT-01M62BMD18K3SE1WXJ7FDN8EPT
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T01:40:10Z
updated_at: 2026-11-02T15:57:27Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-plan-m2
  name: ""
extensions: {}
---

## Description

Goal 2.3 in docs/ROADMAP.md: told "go chop wood", the bot goes to the last place it saw an owner working, if there is one near enough, else to the nearest known trees. Today `planWood` (`internal/behaviour/wood.go`) only knows the bot's own last requested site, saved in the "Owner wood gathering" definition. Nothing notices where an owner works.

From perception over time (the attention design on TKT-01M62EP0P): when trees vanish and logs, stumps and wood drops appear near an owner, that spot is a work site. Remember it per world with what was gathered there and when.

The other half of the original ticket, drops summed per cell and prefab with their trend ("wood piling up near Nimah"), moved to TKT-01M6EFHE67 (Controller: sum drops per cell and prefab with their trend), so that each fits one pull request.

### What to build

- A recorder beside the others in `internal/memory` that watches perception frames for a tree disappearing, or a log, stump or wood drop appearing, within 15 m of an owner (players named in settings `Owners`). Several such events within a few minutes in one 20 m area make one work site, stored per world with its centre, the owner, the prefabs gathered and the last time.
- Store it in `internal/home` like landmarks and danger zones, with the same atomic save, world isolation and bounded retention.
- `planWood` prefers the most recent owner work site in the same world within its distance limits over the bot's own last site, and says which it chose ("where Nimah was chopping").
- The panel map draws work sites, and the voice summary can mention the nearest one.

### Files

- `internal/memory/landmarks.go` and `internal/memory/danger.go` (recorder patterns), `internal/home/landmarks.go`, `internal/home/danger.go`, `internal/home/store.go`
- `internal/behaviour/wood.go` (`planWood`), `internal/behaviour/wood_test.go`
- `internal/app/panel.go` and `panel/src/workspace.jsx` for the map
- `docs/MEMORY.md`

### Verification

- `just check controller test-packages ./internal/memory ./internal/home ./internal/behaviour`
- `just check panel test`
- `just check ci`
- Live evidence for goal 2.3 comes from TKT-01M6EFHE80 (VM: wood-request trial for go chop wood and its refusals), after an owner has chopped somewhere.

## Acceptance criteria

- [ ] Felling, splitting and drops near an owner mark a work site, kept per world with what and when
- [ ] The panel shows work sites, and the voice summary can mention them
- [ ] planWood prefers the most recent owner work site in the same world within its distance limits, and its reply names it, in a test
