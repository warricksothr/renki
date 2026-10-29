---
schema: 4
id: TKT-01M62N12MXJ9XXPX8A2M50HBV4
title: "Controller: corpse run after a death"
type: task
status: in-progress
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
claim:
  actor: agent:codex/renki-resume
  branch: t3/live-verification
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 2b7cad9705a40bac08002ef0c7e6d6bbe06c030a
  session: null
  claimed_at: 2026-10-29T17:07:07Z
  expires_at: null
archive: null
created_at: 2026-10-29T01:40:10Z
updated_at: 2026-10-29T17:08:15Z
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

Finish the existing on-demand corpse run and shun behavior with a live populated-grave check. The 15-seed death run revealed interaction visibility blocked at the grave. Match the game’s hover rule by ignoring the player’s own rigidbody while selecting the nearest other ray hit; retain wall/floor blocking and test the selection in Core. Give tombstones a nonempty localized name, add bounded sight diagnostics, run just check ci and just check review, then land and deploy. Retry the same untouched grave, confirm all 15 seeds reach inventory, explicitly send Tas home, and record the actual outcomes without claiming unobserved death/resume branches.

## Notes

**agent:codex/renki-resume** at 2026-10-29T16:27:40Z

claim taken from agent:claude-code/renki-groom by agent:codex/renki-resume

**agent:codex/renki-resume** at 2026-10-29T16:33:32Z

Completed the inherited shun changes: only successfully submitted orders clear the current world shun; refused/missing-target/status requests retain it, successful set-home/go-home paths clear it, and resuming a follow after death runs even while home is shunned. Added regression coverage for command success/refusal, persisted shun, per-world clearing and resuming after death. just check ci passed. Live grave retrieval/return verification is still outstanding, so acceptance criterion 1 remains unchecked.

**agent:codex/renki-resume** at 2026-10-29T16:45:29Z

Pre-landing inspection found the direct panel Stop/Go home actions also need Told after successful dispatch. Fixed both paths and extended TestPanelButtons to check successful stop/home clear the shun while missing-home refusal retains it. The focused integration test and full just check ci pass.

**agent:codex/renki-resume** at 2026-10-29T16:59:05Z

Landed in PR #9 and deployed as 6baa828. Live interact t-1 targeted Tas’s own tombstone, but it disappeared before arrival and returned target_lost; the controller correctly said the grave was gone. The user confirmed recovering and storing Tas’s items. A later get your things with no grave correctly explained there was nothing to fetch and sent no body task. Positive retrieval/return from a populated Tas tombstone and follow-after-death were not exercised live. Criterion 1 remains unchecked and the ticket remains in review. Tas was returned home at 25/25 health after movement and follow/stop verification.

**agent:codex/renki-resume** at 2026-10-29T17:01:32Z

Additional live death: Greyling killed Tas at 2026-10-29T16:59:35Z, with dead state at approximately (-27.67,30.51,40.38). That is about 35 m from home, outside DiedNearHome=30 m, so afterDeath=home correctly returned him home after respawn rather than shunning home. Respawned 16:59:45Z; task t-11 succeeded at 17:00:20.712Z, ending idle at home at 22/25 health. Inventory was empty before and after the death, so no populated tombstone was available. Positive corpse retrieval and follow-resume-through-death remain unverified live.

**agent:codex/renki-resume** at 2026-10-29T17:07:07Z

Positive seed test exposed a live blocker: interact t-18 reached the own tombstone at (-16.34,31.76,33.26) with an empty pack but stayed there through stuck recoveries without using it. The game’s FindHoverObject sorts ray hits and skips colliders attached to the local player; InteractTask.Sees uses the first Raycast hit and does not skip the player. Fix visibility to choose the nearest non-self hit while retaining wall/floor occlusion, cover that selection off-game, and retry the same populated grave. Also give tombstones a name because GetHoverName intentionally returns an empty string in Valheim.

## Summary

Corpse-run and shun code is implemented, with the latest fixes and regressions in the working tree. Automated CI passes; live retrieval of the grave and return still need validation.
