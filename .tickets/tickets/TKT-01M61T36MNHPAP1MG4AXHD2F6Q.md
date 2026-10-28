---
schema: 4
id: TKT-01M61T36MNHPAP1MG4AXHD2F6Q
title: "Plugin: task runner, safety config and dead-man rule"
type: task
status: review
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

## Summary

Renki.Core.Tasks.TaskRunner holds one body task and runs instants alongside it. It reports accepted/running/succeeded/failed/cancelled, or rejected with unsupported, not_allowed, bad_args or the host's reason. A new body task supersedes the old one, stop cancels everything, timeoutS is enforced, and death fails the body task with died. SafetyConfig is the BepInEx [Safety] section: Enabled, AllowMovement, AllowChat and the doc's allow* switches. stop is always permitted, and welcome.allowedTasks is intersected with it, never added. Dead-man: RenkiPlugin.Update ends the session after 10 s with no controller message, the bot stands, and nothing runs until the next welcome. say: 1 per 2 s with a burst of 3, one line, leading slashes stripped, 200 characters. Unit-tested in Renki.Tests (TaskRunnerTests).
