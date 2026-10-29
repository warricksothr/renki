---
schema: 4
id: TKT-01M61T36DY816WHX3HGPTR77SD
title: "Spike: does Valheim keep running unattended on the VM"
type: spike
status: ready
status_reason: null
priority: normal
due_on: null
labels:
  - vm
assignees: []
milestone: M1
parent: TKT-01M61T36B839WT7KEZ14TDMB8B
origin: null
dependencies:
  - TKT-01M61T36CTW15EXM47J8JQ3SJ6
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-28T17:49:28Z
updated_at: 2026-10-29T00:24:42Z
created_by:
  id: agent:claude-code/valheim-setup
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

The bot is only useful if the game keeps simulating while nobody is
looking at it. Check what happens when the window loses focus, when it's
minimized, and when the RDP session disconnects. docs/BRIDGE-PROTOCOL.md section 12, question 2.

## Acceptance criteria

- [ ] Each of the three cases is tested and the result recorded
- [x] The VM session setup that keeps the game running is written down

## Notes

**agent:claude-code/renki-groom** at 2026-10-28T23:23:28Z

Partly answered on 2026-10-28. Handing the RDP session back to the console with release-rdp.ps1 (tscon) keeps the game running and the bot playing: it followed, went home and reported state for hours with nobody connected. Not tested: the game window unfocused while someone uses the desktop, and minimised. docs/VM.md writes down the session setup (auto-logon, release-rdp, a dummy display plug).

**agent:claude-code/renki-groom** at 2026-10-29T00:24:42Z

Unfocused and minimised tested on 2026-10-28. The game window was driven from the desktop session by a scheduled task with RenkiStart's principal, calling user32 ShowWindow and SetForegroundWindow, and by Notepad taking the focus.

- **Unfocused, 19:21:16–19:22:14:** Notepad held the foreground. The plugin kept reporting state at 49–50 frames per 10 s (5 Hz), and game time advanced in step with the wall clock.
- **Minimised, 19:22:44–19:23:44:** the same, 49–50 frames per 10 s with game time advancing.

So the game keeps simulating and the bridge stays up in both cases. The bot was standing at home throughout, so this does not show that it can move while minimised. The Driver feeds Player.SetControls from a Harmony patch rather than reading the keyboard, so it should. At 19:24:27 the window was minimised again and left that way; the next order the bot obeys will show whether it moves. Once that is confirmed in the audit log, this spike is done.
