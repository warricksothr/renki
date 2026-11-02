---
schema: 4
id: TKT-01M6EFHE4EMKX75VMHWWW7DY7P
title: "Behaviours: deliver at 80% full or when the pile is gone, then return"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - chat
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies:
  - TKT-01M6EFHE2NGKM21P7S45KBK8JQ
  - TKT-01M6EFHE67EXA1WDS82SYP6X9Z
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:55:08Z
updated_at: 2026-11-02T15:55:22Z
created_by:
  id: agent:claude-code/renki-plan-m2
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m2
  name: ""
extensions: {}
---

## Description

Part of TKT-01M62N12HJ (Controller: loot and deliver-when-full behaviours for hauling), split out so each piece fits one pull request. Goal 2.2 in docs/ROADMAP.md says the bot delivers when it is 80 % full or 60 s after the pile near the owner is gone, then comes back. Today the engine's `inventory_full` event (`internal/behaviour/engine.go`, `triggers`) fires only when no slot is free or the bot is over its carry weight, and nothing returns the bot to the owner after a stow.

### What to build

- A trigger, or a condition on the existing `stow` template, that fires when pack weight reaches a set fraction of `WeightMax` (default 0.8) or free slots fall to a set count, and one that fires when no eligible drop has been within the loot radius for 60 s while the bot carries loot items. Both thresholds are instance parameters.
- A deliver-when-full instance: `stow` with `items` set to the loot items and `to` the chests within 10 m of home, chosen as `planWood` in `internal/behaviour/wood.go` chooses them.
- After a successful stow, return: start the same follow again for the owner who was being helped, or `move_to` the last site if the owner has gone. Reuse the return step `collect` already has (`Return`).
- A short chat line when it leaves ("Full. Taking this home.") and when it cannot (no chest, chests full), using the engine's existing `Notify` and `StorageNeed` paths so the same line is not repeated.
- Chat: "help me" and "collect wood" switch loot and deliver on for the owner who asked; "stop" ends the current run. Wire it like `gatherCommand` in `internal/chat/gather.go`.

### Files

- `internal/behaviour/engine.go`, `internal/behaviour/stow.go`, `internal/behaviour/collect.go`, `internal/behaviour/conditions.go`, `internal/behaviour/wood.go`
- `internal/chat/gather.go`, `internal/chat/brain.go`
- `docs/BEHAVIOURS.md`

### Verification

- `just check controller test-packages ./internal/behaviour ./internal/chat`
- `just check ci`
- Live proof belongs to TKT-01M62N12HJ, through the haul report (TKT-01M6DDFTDD).

## Acceptance criteria

- [ ] A deliver run starts when pack weight reaches the set fraction (default 0.8) or free slots reach the set count, in a test
- [ ] A deliver run starts 60 s after the last eligible drop near the owner is gone while loot is carried, in a test
- [ ] After a successful stow the bot follows the same owner again, or returns to the last site if the owner is gone, in a test
- [ ] No chest and chests full are said in chat once, through Notify, in a test
- [ ] help me and collect wood switch loot and deliver on for the asking owner; stop ends the run; non-owners are refused
