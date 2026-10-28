---
schema: 4
id: TKT-01M61T36MNHPAP1MG4AXHD2F6Q
title: "Plugin: task runner, safety config and dead-man rule"
type: task
status: ready
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
updated_at: 2026-10-28T18:20:51Z
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

- [ ] Tasks report accepted, running, then succeeded, failed or cancelled, with the reason codes in 6.6
- [ ] One body task runs at a time; instant tasks run alongside it
- [ ] BepInEx config allow* switches gate tasks, and `welcome` can only narrow them
- [ ] With no controller message for 10 s everything stops, and nothing resumes on its own
- [ ] `say` is rate-limited and a leading / is stripped
