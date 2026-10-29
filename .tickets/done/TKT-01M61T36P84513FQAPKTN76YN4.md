---
schema: 4
id: TKT-01M61T36P84513FQAPKTN76YN4
title: "Plugin: auto character select and join on launch"
type: task
status: done
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
claim: null
archive: null
created_at: 2026-10-28T17:49:28Z
updated_at: 2026-10-28T23:24:05Z
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

- [x] Launching with `+connect host:port` lands the bot's character in the world with no clicks
- [x] A server password from configuration is used when one is asked for
- [x] `logout` saves and returns to the menu via Game.Logout(true, true)

## Implementation plan

Valheim already does this: FejdStartup.ParseArguments, called from FejdStartup.Awake, handles `-joinserverwithcharacter <host:port> <character> <any>` (it selects the character file, sets instant start and calls JoinServer) and `-password <pw>`. So the launch script passes those, and the plugin needs no character-select automation. Two things are left for the plugin: logout, which already calls Game.Logout(true, true) on the controller's logout message, and keeping a deliberate logout from bouncing straight back into the server. Awake runs again whenever the menu scene loads, so the argument would rejoin. A Harmony prefix skips ParseArguments once a logout has been asked for. A rejoin after a kick or a server restart still happens, which is what a bot wants.

## Notes

**agent:claude-code/renki-groom** at 2026-10-28T18:55:50Z

Built and checked off-game; not yet run inside Valheim. In review = check it in game; docs/TESTING.md has the steps.

**agent:claude-code/renki-groom** at 2026-10-28T23:24:05Z

In the game: start-bot.ps1 launches valheim.exe with -joinserverwithcharacter <ip:port> tasleroch -password, and the bot is in the world with no clicks. This is Valheim's own argument rather than +connect: it selects the character too. It needs an IP address (start-bot.ps1 resolves the host name) and a local character (characters_local). RenkiStop's POST /control/logout saves and returns to the menu before the game closes, and AutoJoinPatch keeps it from rejoining.

## Summary

The bot joins the server unattended at launch, with the password, and logs out cleanly on stop.
