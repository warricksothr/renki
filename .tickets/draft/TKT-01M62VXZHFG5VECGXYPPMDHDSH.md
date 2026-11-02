---
schema: 4
id: TKT-01M62VXZHFG5VECGXYPPMDHDSH
title: "Controller: hunt down the creature that killed the owner"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
assignees: []
milestone: null
parent: TKT-01M62N2557R401S3E6JECSAPN4
origin: null
dependencies:
  - TKT-01M62N25ADM0Z968D974G0CHP2
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

Part of defending the owner (TKT-01M62N25A). When a creature kills an
owner, the bot marks it and hunts it within a leash of where it happened,
if guard judges the fight winnable. Otherwise it remembers it, warns the
owner, and marks the place as a danger zone. Off by default, a switch on
the guard behaviour.

## Acceptance criteria

- [ ] A creature that killed an owner is marked and hunted when the bot can win the fight
- [ ] When it cannot, the bot warns the owner and marks the place as dangerous
- [ ] It is off by default

## Notes

**agent:claude-code/renki-plan-m3** at 2026-11-02T15:58:15Z

### Milestone review: not an M3 goal; not ready

M3 grooming, 2026-11-02: hunting an owner's killer serves none of goals 3.1-3.5 in docs/ROADMAP.md. Proposed: unscheduled, or M7. Not ready for an agent, for three reasons:

- "If guard judges the fight winnable" conflicts with docs/SELF_DEFENSE.md ("Do not estimate victory from health fraction and stars alone") and the escort policy in docs/ESCORT.md, which has no winnability test. The owner needs to say what decides a hunt.
- The plugin reports the bot's own `died` event with an attacker (docs/BRIDGE-PROTOCOL.md), not an owner's death and killer. `plugin/Renki.Plugin/CharacterDeathObservation.cs` proves a selected enemy's death only. Owner-death evidence is new protocol work.
- It is at least two pull requests (owner-death evidence, then a bounded hunt goal), so it needs splitting once the above is decided.

**agent:claude-code/renki-plan** at 2026-11-02T16:15:34Z

Applied on 2026-11-02 by the planning pass: milestone cleared, so this ticket is unscheduled. No exit goal in docs/ROADMAP.md needs it. This supersedes any earlier note that only proposed the move. The owner can schedule it at a weekly review.
