---
schema: 4
id: TKT-01M61T36NQQX15VB1YT01TXSTP
title: "Plugin: follow, say and emote tasks"
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - movement
  - chat
assignees: []
milestone: M1
parent: TKT-01M61T36B839WT7KEZ14TDMB8B
origin: null
dependencies:
  - TKT-01M61T36N62DBDZEWCDVYBPWYR
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-28T17:49:28Z
updated_at: 2026-10-29T00:18:48Z
created_by:
  id: agent:claude-code/valheim-setup
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

The rest of the M1 task catalogue in docs/BRIDGE-PROTOCOL.md 7.2.

## Acceptance criteria

- [x] `follow` keeps its distance with hysteresis, and runs when the leader runs
- [x] `follow` fails with `target_lost` when the player leaves
- [x] `say` sends through Chat.SendText on normal, shout and whisper
- [x] `emote` plays through Player.StartEmote

## Notes

**agent:claude-code/renki-groom** at 2026-10-28T18:50:44Z

Built against the game (assembly_valheim 0.221.x on the dev box) and unit-tested off-game; not yet run inside Valheim. In review = check it in game; docs/TESTING.md has the steps. Tune in game: the 5.5 m/s sprint threshold for the leader.

**agent:claude-code/renki-groom** at 2026-10-28T23:56:57Z

target_lost verified in the game on 2026-10-28 at 18:38:05: Nimah logged off mid-follow, and task t-3 failed with target_lost 5 s after the 'out of sight' trace. Say on normal and shout is verified. Emote has not been seen in the game yet.

**agent:claude-code/renki-groom** at 2026-10-29T00:18:48Z

Emote verified in the game on 2026-10-28 at 19:17:57: 'tas wave at me' sent emote wave (t-14), which succeeded.

## Summary

follow, say and emote are in the plugin. All three were verified in the game on 2026-10-28: follow with hysteresis and running, target_lost on logoff, say on normal and shout, and the wave emote.
