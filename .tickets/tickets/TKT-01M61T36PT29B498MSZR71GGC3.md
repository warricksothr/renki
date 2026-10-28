---
schema: 4
id: TKT-01M61T36PT29B498MSZR71GGC3
title: "VM launch script: start and stop the bot on demand"
type: task
status: review
status_reason: null
priority: normal
due_on: null
labels:
  - vm
  - tooling
assignees: []
milestone: M1
parent: TKT-01M61T36B839WT7KEZ14TDMB8B
origin: null
dependencies:
  - TKT-01M61T36CTW15EXM47J8JQ3SJ6
  - TKT-01M61T36P84513FQAPKTN76YN4
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

So the bot is around only when it's wanted: one script brings it up,
another takes it down cleanly.

## Acceptance criteria

- [ ] A PowerShell script launches Valheim via Steam with +connect and joins
- [x] It starts the controller too, when the controller runs on the VM
- [ ] A stop script logs the bot out cleanly before closing the game

## Implementation plan

scripts/vm/start-bot.ps1 starts the controller (when its exe sits beside the script) and launches Valheim through Steam with -joinserverwithcharacter and -password from a bot.local.psd1 settings file. scripts/vm/stop-bot.ps1 asks the controller's new loopback control API (POST /control/logout) to log the bot out, waits on GET /control/status until the plugin reports the menu or disconnects, then closes the game and the controller. `just package` builds dist/renki-vm/ with controller.exe, the plugin DLLs, the scripts and example configs, ready to copy to the VM.

## Notes

**agent:claude-code/renki-groom** at 2026-10-28T18:55:50Z

Built and checked off-game; not yet run inside Valheim. In review = check it in game; docs/TESTING.md has the steps.

## Summary

scripts/vm: start-bot.ps1 starts controller.exe when it sits beside the script (logs go to scripts/vm/logs), launches valheim.exe with -joinserverwithcharacter/-password from bot.local.psd1 (template: bot.example.psd1), and waits for /control/status to report in_world. stop-bot.ps1 POSTs /control/logout, waits for the menu phase, closes the game (CloseMainWindow, then Stop-Process after 30 s) and stops the controller. The controller's /control/ API answers loopback only and wants the token as a bearer when one is configured. `just package` builds dist/renki-vm/ with everything for the VM. Both scripts parse under Windows PowerShell 5.1. The Windows controller.exe was run on this Windows host and its status and logout endpoints answered through common.ps1. Not yet run against the game.
