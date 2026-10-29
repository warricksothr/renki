---
schema: 4
id: TKT-01M61T36N62DBDZEWCDVYBPWYR
title: "Plugin: movement driver with move_to, stop and look_at"
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - movement
assignees: []
milestone: M1
parent: TKT-01M61T36B839WT7KEZ14TDMB8B
origin: null
dependencies:
  - TKT-01M61T36MNHPAP1MG4AXHD2F6Q
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-28T17:49:28Z
updated_at: 2026-10-29T00:16:18Z
created_by:
  id: agent:claude-code/valheim-setup
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

Drive the character from inside the game rather than faking keyboard
input, so it keeps working when the window isn't focused (docs/BRIDGE-PROTOCOL.md 7.3 and
section 10).

## Acceptance criteria

- [x] A Harmony prefix on PlayerController.FixedUpdate feeds our inputs to Player.SetControls
- [x] Paths come from Pathfinding.GetPath with AgentType.Humanoid, falling back to a straight line
- [x] Stuck detection jumps, strafes and re-paths, then fails with `stuck` after 3 rounds
- [x] `look_at` turns the character via Character.SetLookDir

## Notes

**agent:claude-code/renki-groom** at 2026-10-28T18:50:44Z

Built against the game (assembly_valheim 0.221.x on the dev box) and unit-tested off-game; not yet run inside Valheim. In review = check it in game; docs/TESTING.md has the steps. Tune in game: the 540°/s turn rate, waypoint reach 1 m, and whether GetPath needs a frame or two before the navmesh tiles exist (the straight-line fallback covers that).

**agent:claude-code/renki-groom** at 2026-10-29T00:16:17Z

Stuck recovery verified in the game on 2026-10-28, on a slope north of home around (-10, 39, 318).

- **First try, 19:06:07 onwards:** the bot never gave up. Every strafe carried it about 3 m sideways, just out of the 3 m circle, and leaving the circle reset the count. It bounced between (-12.3, 317.3) and (-9.3, 317.9) on Jump and Strafe and never reached Repath.
- **The fix:** once a recovery has fired, leaving the circle only starts a new one, and the count resets only past Escape (6 m) from where the first recovery fired. The new test StrafingOverTheEdgeAndBackIsStillStuck replays the slope; it fails on the old detector and passes on the new one.
- **After deploying the fix, 19:12:39–19:13:36:** on "come to me" (t-5), the bot escaped twice by strafing and repathing up the slope, then sat stuck at (-3.9, 35.7, 315.7) through attempts 1–9. At 19:13:36 the task failed with `stuck` ("no progress after 9 recoveries"). The controller sent it home (why "after move_to failed: stuck"), and it shouted "I'm stuck. I'll try to make it home."

## Summary

The movement driver steers through Player.SetControls, paths with the game's navmesh and falls back to a straight line, and recovers from being stuck with jump, strafe and re-path before failing with stuck. Verified in the game on 2026-10-28. That test found that a strafe just over the edge of the circle reset the count; the fix on t3/m1-closeout counts progress only past 6 m from where the first recovery fired. Deep water on the straight-line fallback is filed as TKT-01M62G71Q.
