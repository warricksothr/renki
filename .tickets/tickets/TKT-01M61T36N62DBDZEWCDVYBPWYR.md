---
schema: 4
id: TKT-01M61T36N62DBDZEWCDVYBPWYR
title: "Plugin: movement driver with move_to, stop and look_at"
type: task
status: review
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
claim:
  actor: agent:claude-code/renki-groom
  branch: t3/groom-start-renki
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 94c0949e9abff4d3c2953c782958e7a0cbd1595c
  session: null
  claimed_at: 2026-10-28T18:50:44Z
  expires_at: null
archive: null
created_at: 2026-10-28T17:49:28Z
updated_at: 2026-10-28T18:50:49Z
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

## Summary

Driver (Renki.Plugin/Driver.cs) is a Harmony prefix on PlayerController.FixedUpdate that, while a session is up, ticks the task runner and calls Player.SetControls with the bot's inputs, converting world directions to look-relative axes (Core Controls.ToLocal). It skips LateUpdate so mouse look cannot fight the steering. It turns via Character.SetLookDir at 540°/s. Walker paths with Pathfinding.GetPath(AgentType.Humanoid) and falls back to a straight line. StuckDetector does jump, strafe, re-path over 3 rounds, then stuck (unit-tested). look_at holds a target for 4 s, and the bot keeps walking while it looks.
