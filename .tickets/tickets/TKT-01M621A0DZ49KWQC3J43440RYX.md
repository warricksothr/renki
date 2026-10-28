---
schema: 4
id: TKT-01M621A0DZ49KWQC3J43440RYX
title: Remote control of the bot VM over SSH
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
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:claude-code/renki-groom
  branch: t3/groom-start-renki
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: a13120e7f465a835a60ff50699a865fe378716dd
  session: null
  claimed_at: 2026-10-28T19:55:31Z
  expires_at: null
archive: null
created_at: 2026-10-28T19:55:31Z
updated_at: 2026-10-28T20:07:02Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

So the tooling on the dev box can deploy, start, stop and inspect the bot on the Windows VM without anyone at it. docs/VM.md is the walkthrough.

## Acceptance criteria

- [x] setup-remote.ps1 enables OpenSSH (key-only, PowerShell shell), authorises the dev box key, and registers RenkiStart/RenkiStop tasks in the desktop session
- [x] `just vm check|start|stop|restart|status|logs|deploy|shell` drive the VM from the dev box
- [x] `just vm deploy` packages, stops, copies, installs the plugin and starts the bot again
- [ ] One full deploy and restart has run against the real VM

## Notes

**agent:claude-code/renki-groom** at 2026-10-28T19:55:31Z

All scripts parse under Windows PowerShell 5.1, and the just recipes list and refuse cleanly without a VM. Nothing has run against the VM yet; criterion 4 waits on the owner finishing docs/VM.md. Untested assumptions: scp writes to Windows paths in the /C:/renki form, and CloseMainWindow from a scheduled task in the same desktop session closes Valheim (stop-bot.ps1 falls back to Stop-Process).

**agent:claude-code/renki-groom** at 2026-10-28T20:07:02Z

Defender on the VM flagged controller.exe as Trojan:Win32/Bearfoos.A!ml (an ML heuristic). Four builds scanned clean on the dev box, so it is likely a cloud reputation verdict at first run. The owner added a Defender exclusion on the VM (2026-10-28). If it returns, the options are the -tags novoice build or running the controller on Linux off the VM.
