---
schema: 4
id: TKT-01M61T36NQQX15VB1YT01TXSTP
title: "Plugin: follow, say and emote tasks"
type: task
status: draft
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

The rest of the M1 task catalogue in docs/BRIDGE-PROTOCOL.md 7.2.

## Acceptance criteria

- [ ] `follow` keeps its distance with hysteresis, and runs when the leader runs
- [ ] `follow` fails with `target_lost` when the player leaves
- [ ] `say` sends through Chat.SendText on normal, shout and whisper
- [ ] `emote` plays through Player.StartEmote
