---
schema: 4
id: TKT-01M62N12MXJ9XXPX8A2M50HBV4
title: "Controller: corpse run after a death"
type: task
status: review
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - movement
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies:
  - TKT-01M62EP0PDAB02DNJAJ6WS3MYD
  - TKT-01M62PRDRKEHTQ1WC12FQ3S11J
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T01:40:10Z
updated_at: 2026-10-29T16:59:05Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:codex/renki-resume
  name: ""
extensions: {}
---

## Description

Agreed on 2026-10-28 for M2: after a death, the bot's things are in a tombstone where it died. Memory records each death's place (TKT-01M62EP0P); the corpse run walks back there, empties the tombstone (interact), and returns, if afterDeath allows and the place is not dangerous. A haul that dies otherwise loses the load and the axe.

## Acceptance criteria

- [ ] After a respawn the bot can go back to its tombstone, empty it and return
- [x] It does not go back into a place that just killed it without being told

## Implementation plan

Lean version, ahead of full memory (TKT-01M62EP0P), because the bot lost its gear to three graves at home on 2026-10-29. Plugin: perception reports graves with something in them as structure kind 'tombstone', with mine on the bot's own (TombStone.Awake postfix, since they are not pieces). Controller: the home keeper remembers each death per world (the last 5) in the store. A death within 30 m of home keeps the bot from going back after the respawn, and it says so. Chat: 'get your things' empties its own grave in sight, else walks to the last one and empties it there. A grave gone is forgotten. The bot never goes back on its own, which is criterion 2.

## Notes

**agent:codex/renki-resume** at 2026-10-29T16:27:40Z

claim taken from agent:claude-code/renki-groom by agent:codex/renki-resume

**agent:codex/renki-resume** at 2026-10-29T16:33:32Z

Completed the inherited shun changes: only successfully submitted orders clear the current world shun; refused/missing-target/status requests retain it, successful set-home/go-home paths clear it, and resuming a follow after death runs even while home is shunned. Added regression coverage for command success/refusal, persisted shun, per-world clearing and resuming after death. just check ci passed. Live grave retrieval/return verification is still outstanding, so acceptance criterion 1 remains unchecked.

**agent:codex/renki-resume** at 2026-10-29T16:45:29Z

Pre-landing inspection found the direct panel Stop/Go home actions also need Told after successful dispatch. Fixed both paths and extended TestPanelButtons to check successful stop/home clear the shun while missing-home refusal retains it. The focused integration test and full just check ci pass.

**agent:codex/renki-resume** at 2026-10-29T16:59:05Z

Landed in PR #9 and deployed as 6baa828. Live interact t-1 targeted Tas’s own tombstone, but it disappeared before arrival and returned target_lost; the controller correctly said the grave was gone. The user confirmed recovering and storing Tas’s items. A later get your things with no grave correctly explained there was nothing to fetch and sent no body task. Positive retrieval/return from a populated Tas tombstone and follow-after-death were not exercised live. Criterion 1 remains unchecked and the ticket remains in review. Tas was returned home at 25/25 health after movement and follow/stop verification.

## Summary

Corpse-run and shun code is implemented, with the latest fixes and regressions in the working tree. Automated CI passes; live retrieval of the grave and return still need validation.
