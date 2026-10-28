---
schema: 4
id: TKT-01M61T36M470JHP01RXXVRF05A
title: "Plugin: state stream, chat and event reporting"
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - chat
assignees: []
milestone: M1
parent: TKT-01M61T36B839WT7KEZ14TDMB8B
origin: null
dependencies:
  - TKT-01M61T36KKDCYVKZC4AE9Z2F9S
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-28T17:49:28Z
updated_at: 2026-10-28T23:42:30Z
created_by:
  id: agent:claude-code/valheim-setup
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

Send what the controller needs for M1: the bot's own state (docs/BRIDGE-PROTOCOL.md 6.1),
chat lines (6.4) and the events in 6.5. Perception and inventory streams are
M2.

## Acceptance criteria

- [x] `state` goes out at stateHz with the fields in 6.1
- [x] Chat arrives via a postfix on Chat.OnNewChatMessage, with the bot's own lines excluded
- [x] `died`, `respawned`, `damaged`, `player_joined` and `player_left` events are sent

## Notes

**agent:claude-code/renki-groom** at 2026-10-28T18:50:44Z

Built against the game (assembly_valheim 0.221.x on the dev box) and unit-tested off-game; not yet run inside Valheim. In review = check it in game; docs/TESTING.md has the steps.

**agent:claude-code/renki-groom** at 2026-10-28T23:24:05Z

In the game on 2026-10-28: state at 5 Hz (now with character and world), chat from Nimah and jarl, and the events damaged, player_joined and player_left all reached the controller. died and respawned were not seen in the game; the plugin's Character.m_onDeath hook is the same mechanism as damaged.

**agent:claude-code/renki-groom** at 2026-10-28T23:42:30Z

died and respawned seen in the game at 18:39:19 and 18:39:29 on 2026-10-28, with low_health at 18:37:36 before them; this supersedes the earlier note that they were not seen.

## Summary

The plugin streams state, chat (without the bot's own lines) and events; seen in the game, except died and respawned, which share damaged's hook.
