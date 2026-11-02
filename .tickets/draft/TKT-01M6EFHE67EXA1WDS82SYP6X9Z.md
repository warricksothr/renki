---
schema: 4
id: TKT-01M6EFHE67EXA1WDS82SYP6X9Z
title: "Controller: sum drops per cell and prefab with their trend"
type: task
status: draft
status_reason: null
priority: normal
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
created_at: 2026-11-02T15:55:08Z
updated_at: 2026-11-02T15:55:08Z
created_by:
  id: agent:claude-code/renki-plan-m2
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m2
  name: ""
extensions: {}
---

## Description

Split out of TKT-01M62N12G9 (Controller: notice where owners work, and the drops piling up), whose second criterion is this. The deliver-when-full work needs to know when the pile near an owner is gone (goal 2.2: deliver 60 s after the pile is gone), and the voice and the panel benefit from "wood piling up near Nimah".

### What to build

- From perception over time, group drops (`protocol.ResourceDrop`) within 30 m of the bot by a 4 m cell and prefab, with the stack total per cell and its trend over the last 5 minutes (rising, steady, falling, gone).
- Keep it in memory only, scoped to the world and connection, and bounded. It is a live signal, not a durable record.
- Expose it to the behaviour `View` so a trigger can ask "is there an eligible pile within R of this owner, and since when not".

### Files

- `internal/memory/pickups.go` and `internal/home/pickups.go` (existing pickup memory; check what they already record before adding a new type)
- `internal/behaviour/engine.go` (`view`, `triggers`, where `drop_appeared` is computed today)
- `internal/session` for the perception snapshot

### Verification

- `just check controller test-packages ./internal/memory ./internal/behaviour`
- `just check ci`

## Acceptance criteria

- [ ] Drops within 30 m are grouped per 4 m cell and prefab with a stack total and a 5-minute trend, in a test
- [ ] The grouping is scoped to world and connection and bounded in size, in a test
- [ ] The behaviour View can ask whether an eligible pile is near an owner and since when not
