---
schema: 4
id: TKT-01M61T36N62DBDZEWCDVYBPWYR
title: "Plugin: movement driver with move_to, stop and look_at"
type: task
status: draft
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
updated_at: 2026-10-28T17:49:28Z
created_by:
  id: agent:claude-code/valheim-setup
  name: Claude Code
updated_by:
  id: agent:claude-code/valheim-setup
  name: Claude Code
extensions: {}
---

## Description

Drive the character from inside the game rather than faking keyboard
input, so it keeps working when the window isn't focused (docs/BRIDGE-PROTOCOL.md 7.3 and
section 10).

## Acceptance criteria

- [ ] A Harmony prefix on PlayerController.FixedUpdate feeds our inputs to Player.SetControls
- [ ] Paths come from Pathfinding.GetPath with AgentType.Humanoid, falling back to a straight line
- [ ] Stuck detection jumps, strafes and re-paths, then fails with `stuck` after 3 rounds
- [ ] `look_at` turns the character via Character.SetLookDir
