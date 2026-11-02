---
schema: 4
id: TKT-01M62VXZ97S93DBB7T2H88H3EC
title: "Controller: suggest chest assignments from contents and signs"
type: task
status: draft
status_reason: null
priority: low
due_on: null
labels:
  - controller
assignees: []
milestone: null
parent: TKT-01M62VXZ2FWDX3D4Z6XX1MB5A6
origin: null
dependencies:
  - TKT-01M62VXZ84SA9TSJC6NJNPV8KJ
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:48Z
updated_at: 2026-11-02T16:23:39Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-plan
  name: ""
extensions: {}
---

## Description

Later: the bot suggests what each chest is for from what it already holds
(a chest that is mostly wood is the wood chest), and from a sign placed next
to it. It offers the suggestions to an owner, who accepts them into the
assignment table. It never assigns on its own.

This is unscheduled, a stretch beyond M5, and is not on an exit goal (docs/ROADMAP.md). It needs the assignment table from TKT-01M62VXZ84 (Controller: assign every item to chests).

### What exists

- Remembered chest contents per world with their ages (internal/home/containers.go, `GET /api/memory/containers`).
- `marker_cargo` already prefers a chest that holds an item when unloading (internal/behaviour/marker_cargo.go), which is the same signal used informally.
- Signs are not perceived: `Classify` in plugin/Renki.Plugin/Perceiver.cs does not list them, and `interact` refuses signs (docs/BRIDGE-PROTOCOL.md). Reading a sign's text needs a perception change.

### What to build

- Controller: for each remembered chest with a snapshot under a configurable age, suggest the item or selector that makes up at least 60 % of its stacks, skipping chests already in the table or marked don't-touch.
- Plugin (optional second step, separate PR if needed): report a sign's text and position in perception, and suggest an assignment for a chest within 2 m of a sign whose text names an item or selector.
- Owners see suggestions with `<name> suggest chests` and in the panel's assignment editor, and accept one by name. Nothing changes until an owner accepts.

### Files

- `internal/home/containers.go`: snapshots
- the assignment table and chat commands from TKT-01M62VXZ84
- `plugin/Renki.Plugin/Perceiver.cs` and `docs/BRIDGE-PROTOCOL.md` for signs

### Verification

`just check controller test-packages ./internal/home ./internal/behaviour ./internal/chat` with tests for: a mostly-wood chest suggested as `wood`, a mixed chest not suggested, a stale snapshot skipped, and nothing applied without acceptance. `just check ci`.

## Acceptance criteria

- [ ] The bot suggests assignments from a chest's contents and a sign beside it
- [ ] Suggestions take effect only when an owner accepts them

## Notes

**agent:claude-code/renki-plan** at 2026-11-02T16:15:34Z

Applied on 2026-11-02 by the planning pass: milestone cleared, so this ticket is unscheduled. No exit goal in docs/ROADMAP.md needs it. This supersedes any earlier note that only proposed the move. The owner can schedule it at a weekly review.

**agent:claude-code/renki-plan** at 2026-11-02T16:23:39Z

Supersedes the 2026-11-02 note's 'milestone cleared': this ticket never had a milestone. It stays unscheduled, a stretch beyond M5, and its description and the base-upkeep epic now say so.
