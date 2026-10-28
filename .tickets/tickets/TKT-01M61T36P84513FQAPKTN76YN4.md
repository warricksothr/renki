---
schema: 4
id: TKT-01M61T36P84513FQAPKTN76YN4
title: "Plugin: auto character select and join on launch"
type: task
status: review
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - vm
assignees: []
milestone: M1
parent: TKT-01M61T36B839WT7KEZ14TDMB8B
origin: null
dependencies:
  - TKT-01M61T36HB0PY76DJNZ1AGHCG2
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:claude-code/renki-groom
  branch: t3/groom-start-renki
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 60a7cadfdcfe97db807f95dbbe7e32d6a14e992c
  session: null
  claimed_at: 2026-10-28T18:51:07Z
  expires_at: null
archive: null
created_at: 2026-10-28T17:49:28Z
updated_at: 2026-10-28T18:55:50Z
created_by:
  id: agent:claude-code/valheim-setup
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

`+connect` is parsed by FejdStartup, but character select still waits for
a click. Pick the configured character from FejdStartup.m_profiles and start
it. docs/BRIDGE-PROTOCOL.md section 12, question 3.

## Acceptance criteria

- [ ] Launching with `+connect host:port` lands the bot's character in the world with no clicks
- [ ] A server password from configuration is used when one is asked for
- [ ] `logout` saves and returns to the menu via Game.Logout(true, true)

## Implementation plan

Valheim already does this: FejdStartup.ParseArguments, called from FejdStartup.Awake, handles `-joinserverwithcharacter <host:port> <character> <any>` (it selects the character file, sets instant start and calls JoinServer) and `-password <pw>`. So the launch script passes those, and the plugin needs no character-select automation. Two things are left for the plugin: logout, which already calls Game.Logout(true, true) on the controller's logout message, and keeping a deliberate logout from bouncing straight back into the server. Awake runs again whenever the menu scene loads, so the argument would rejoin. A Harmony prefix skips ParseArguments once a logout has been asked for. A rejoin after a kick or a server restart still happens, which is what a bot wants.

## Notes

**agent:claude-code/renki-groom** at 2026-10-28T18:55:50Z

Built and checked off-game; not yet run inside Valheim. In review = check it in game; docs/TESTING.md has the steps.

## Summary

There is no character-select automation, because Valheim already has it: FejdStartup.ParseArguments (from Awake) handles `-joinserverwithcharacter <host:port> <character> <any>` and `-password <pw>`. The trailing argument is required (the game checks Length > i + 3). start-bot.ps1 passes these. The plugin handles the controller's logout with Game.Logout(true, true), and AutoJoinPatch skips ParseArguments after a deliberate logout, because Awake runs again on the menu scene and would rejoin straight away. A rejoin after a kick or a server restart still happens. The controller gained POST /control/logout to ask for that logout.
