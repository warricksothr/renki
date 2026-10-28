---
schema: 4
id: TKT-01M61T36M470JHP01RXXVRF05A
title: "Plugin: state stream, chat and event reporting"
type: task
status: ready
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
updated_at: 2026-10-28T18:20:51Z
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

- [ ] `state` goes out at stateHz with the fields in 6.1
- [ ] Chat arrives via a postfix on Chat.OnNewChatMessage, with the bot's own lines excluded
- [ ] `died`, `respawned`, `damaged`, `player_joined` and `player_left` events are sent
