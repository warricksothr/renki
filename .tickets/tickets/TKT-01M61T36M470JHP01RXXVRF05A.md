---
schema: 4
id: TKT-01M61T36M470JHP01RXXVRF05A
title: "Plugin: state stream, chat and event reporting"
type: task
status: review
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

## Summary

Reporter (Renki.Plugin/Reporter.cs) sends state at welcome.rates.stateHz with every field in 6.1: flags from the Character/Player predicates, near_fire from the CampFire status effect, and equipped from Humanoid's item slots as prefab names. Chat comes from a postfix on Chat.OnNewChatMessage, skipping lines whose sender is the local character, with owner set from welcome.owners. Events: damaged and died from Character.m_onDamaged/m_onDeath, respawned when a new local player appears after a death, player_joined/left from ZNet's player list (the first list is the baseline), and also low_health and entered_biome from the 6.5 table.
