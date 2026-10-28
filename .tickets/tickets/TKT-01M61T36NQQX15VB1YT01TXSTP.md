---
schema: 4
id: TKT-01M61T36NQQX15VB1YT01TXSTP
title: "Plugin: follow, say and emote tasks"
type: task
status: review
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

The rest of the M1 task catalogue in docs/BRIDGE-PROTOCOL.md 7.2.

## Acceptance criteria

- [x] `follow` keeps its distance with hysteresis, and runs when the leader runs
- [x] `follow` fails with `target_lost` when the player leaves
- [x] `say` sends through Chat.SendText on normal, shout and whisper
- [x] `emote` plays through Player.StartEmote

## Notes

**agent:claude-code/renki-groom** at 2026-10-28T18:50:44Z

Built against the game (assembly_valheim 0.221.x on the dev box) and unit-tested off-game; not yet run inside Valheim. In review = check it in game; docs/TESTING.md has the steps. Tune in game: the 5.5 m/s sprint threshold for the leader.

## Summary

follow (FollowTask) uses Core FollowPace: it stops inside the distance and waits for 1 m of slack before moving again; it walks (Valheim's jog) under twice the distance and runs beyond it, and in auto mode runs when the leader sprints. The leader's pace is measured from their position, because IsRunning is not synced for remote players. It re-paths when the leader moves 2 m, and fails with target_lost after 5 s without the player. say goes through Chat.SendText as Normal, Shout or Whisper; Valheim's whisper is proximity speech, so `to` is not used. emote parses the name into the game's Emotes enum and plays it through Emote.DoEmote, which calls Player.StartEmote with the right one-shot flag.
