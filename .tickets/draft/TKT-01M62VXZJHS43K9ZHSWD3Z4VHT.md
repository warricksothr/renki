---
schema: 4
id: TKT-01M62VXZJHS43K9ZHSWD3Z4VHT
title: Decide whether the bot may take revenge on a player
type: spike
status: draft
status_reason: null
priority: low
due_on: null
labels:
  - controller
  - safety
assignees: []
milestone: null
parent: TKT-01M62N2557R401S3E6JECSAPN4
origin: null
dependencies:
  - TKT-01M62VXZHFG5VECGXYPPMDHDSH
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

Raised on 2026-10-28: should the bot mark for death a player who kills its
owner? Not decided. If it is wanted, it is its own switch, off by default,
and the bot asks an owner before going after anyone, every time. Settle
whether it is wanted at all before building anything.

## Acceptance criteria

- [ ] An owner has decided whether this is wanted, and the decision is recorded here

## Notes

**agent:claude-code/renki-plan-m3** at 2026-11-02T15:58:15Z

### Milestone review: owner decision, not an M3 goal

M3 grooming, 2026-11-02: this spike serves none of goals 3.1-3.5 and is waiting only on an owner's decision. Proposed: take it out of M3 (unscheduled). If the owner says no, archive it with that reason; if yes, it waits on TKT-01M62VXZHF (Controller: hunt down the creature that killed the owner) and on owner-bound player identity, TKT-01M6DCFSME (M4, only owners command it).

**agent:claude-code/renki-plan-m3** at 2026-11-02T15:58:34Z

Correction to the previous note: TKT-01M6DCFSME's title is "Chat: fence untrusted lines in owner turns, stop naming owners". It owns roadmap goal 4.4 (only owners command it), which is what the note meant.

**agent:claude-code/renki-plan** at 2026-11-02T16:15:34Z

Applied on 2026-11-02 by the planning pass: milestone cleared, so this ticket is unscheduled. No exit goal in docs/ROADMAP.md needs it. This supersedes any earlier note that only proposed the move. The owner can schedule it at a weekly review.
