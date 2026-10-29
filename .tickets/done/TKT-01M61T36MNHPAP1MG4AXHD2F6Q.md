---
schema: 4
id: TKT-01M61T36MNHPAP1MG4AXHD2F6Q
title: "Plugin: task runner, safety config and dead-man rule"
type: task
status: done
status_reason: null
priority: high
due_on: null
labels:
  - plugin
  - safety
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
updated_at: 2026-10-28T23:59:31Z
created_by:
  id: agent:claude-code/valheim-setup
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

The task lifecycle from docs/BRIDGE-PROTOCOL.md 6.6 and 7.1 and the safety rules in section 8.
This is what makes it safe to point a controller, or an LLM behind one, at a
character on a shared server.

## Acceptance criteria

- [x] Tasks report accepted, running, then succeeded, failed or cancelled, with the reason codes in 6.6
- [x] One body task runs at a time; instant tasks run alongside it
- [x] BepInEx config allow* switches gate tasks, and `welcome` can only narrow them
- [x] With no controller message for 10 s everything stops, and nothing resumes on its own
- [x] `say` is rate-limited and a leading / is stripped

## Notes

**agent:claude-code/renki-groom** at 2026-10-28T18:50:44Z

Built against the game (assembly_valheim 0.221.x on the dev box) and unit-tested off-game; not yet run inside Valheim. In review = check it in game; docs/TESTING.md has the steps.

**agent:claude-code/renki-groom** at 2026-10-28T23:59:24Z

Killing the controller verified in the game on 2026-10-28.

- At 18:58:16.8 I killed controller.exe on the VM while the bot was running after Nimah (task t-1, follow).
- The plugin logged "bridge: disconnected; retrying in 16 s" and kept following on its last order. It ran about 28 m, from (12.3, 32.0, 35.7) to (38.9, 32.3, 25.5).
- It then logged "no controller message for 10 s; stopping". After that the log has no walker or follow lines, only "retrying in 30 s": it stayed stopped, and nothing resumed on its own.
- At 18:58:51 I restarted with RenkiStart. At 18:59:04.8 the plugin reconnected, and the controller sent move_to home (why "after an outage") and "Something cut me off. Heading home."

The 10 s count starts at the plugin's last message from the controller, which came no later than the kill, so the stop came within 10 s of the kill.

## Summary

The task runner, safety config and dead-man rule are in the plugin, covered by the C# tests and interop against the real controller. In the game on 2026-10-28, killing the controller mid-follow stopped the bot within 10 s, and it stayed stopped until the controller came back.
