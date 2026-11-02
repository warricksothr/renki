---
schema: 4
id: TKT-01M62VXZKJFMNNZS4PC0T4RCDV
title: "Doors: a door whose state returns after 3 s counts as refused"
type: bug
status: draft
status_reason: null
priority: low
due_on: null
labels:
  - plugin
  - movement
assignees: []
milestone: null
parent: null
origin: null
dependencies:
  - TKT-01M62MB5YRGF9FT485055911JP
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:49Z
updated_at: 2026-11-02T16:15:34Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-plan
  name: ""
extensions: {}
---

## Description

Doorman waits SettleFor (3 s) for an opened door's state to come back. On a
slow link it can arrive later, so Tick judges the door refused, forgets it,
and does not shut it behind the bot. The door is then left open. Accepted in
PR 3 as rare. Fix if it shows up: wait longer while the door's owner has not
answered, or watch for the state for a while after judging it refused.

### Goal

None of goals 4.1-4.6 needs this. It was filed as "fix if it shows up" and has not been reported since. It waits on TKT-01M62MB5YR (Open closed doors in the bot's way, and close them behind it), which is in review in M2.

### Files

- `plugin/Renki.Plugin/Doorman.cs` (`SettleFor` at :28, the judging in `Tick`)
- `plugin/Renki.Core/Movement/Doors.cs` (`DoorRules`) and `plugin/Renki.Tests/DoorNavigationTests.cs`, if the timing decision moves into Core

### Verification

- `just check plugin test-core DoorNavigation` for any logic moved into Core.
- `just check plugin build` with `VALHEIM_DIR` set, and `just check ci`.
- Reproducing the slow link needs the game; a live check is optional.

## Acceptance criteria

- [ ] A door that opens after the settle time is still shut behind the bot

## Notes

**agent:claude-code/renki-plan** at 2026-11-02T16:15:34Z

Applied on 2026-11-02 by the planning pass: milestone cleared, so this ticket is unscheduled. No exit goal in docs/ROADMAP.md needs it. This supersedes any earlier note that only proposed the move. The owner can schedule it at a weekly review.
