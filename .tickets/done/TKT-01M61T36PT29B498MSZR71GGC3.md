---
schema: 4
id: TKT-01M61T36PT29B498MSZR71GGC3
title: "VM launch script: start and stop the bot on demand"
type: task
status: done
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
  - TKT-01M621A0DZ49KWQC3J43440RYX
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

So the bot is around only when it's wanted: one script brings it up,
another takes it down cleanly.

## Acceptance criteria

- [x] A PowerShell script launches Valheim via Steam with +connect and joins
- [x] It starts the controller too, when the controller runs on the VM
- [x] A stop script logs the bot out cleanly before closing the game

## Implementation plan

scripts/vm/start-bot.ps1 starts the controller (when its exe sits beside the script) and launches Valheim through Steam with -joinserverwithcharacter and -password from a bot.local.psd1 settings file. scripts/vm/stop-bot.ps1 asks the controller's new loopback control API (POST /control/logout) to log the bot out, waits on GET /control/status until the plugin reports the menu or disconnects, then closes the game and the controller. `just package` builds dist/renki-vm/ with controller.exe, the plugin DLLs, the scripts and example configs, ready to copy to the VM.

## Notes

**agent:claude-code/renki-groom** at 2026-10-28T18:55:50Z

Built and checked off-game; not yet run inside Valheim. In review = check it in game; docs/TESTING.md has the steps.

**agent:claude-code/renki-groom** at 2026-10-28T23:24:05Z

start-bot.ps1 runs valheim.exe directly rather than through Steam (Steam just has to be running), with -joinserverwithcharacter; it joined every time on 2026-10-28. stop-bot.ps1 logs out through the control API, then closes the game; just vm stop and restart ran a dozen times.

## Summary

start-bot.ps1 and stop-bot.ps1 start the controller and the game, join unattended, and log out cleanly before closing; run many times on the bot VM.
