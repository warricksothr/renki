---
schema: 4
id: TKT-01M61T36HB0PY76DJNZ1AGHCG2
title: Scaffold the BepInEx plugin project and its build
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - tooling
assignees: []
milestone: M1
parent: TKT-01M61T36B839WT7KEZ14TDMB8B
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-28T17:49:28Z
updated_at: 2026-10-28T17:49:28Z
created_by:
  id: agent:claude-code/valheim-setup
  name: Claude Code
updated_by:
  id: agent:claude-code/valheim-setup
  name: Claude Code
extensions: {}
---

## Description

A C# BepInEx 5 plugin project under `plugin/`. Valheim's managed
assemblies can't be redistributed, so the build references them from a
configured game install rather than from the repository.

## Acceptance criteria

- [ ] `plugin/` builds against a game path given by configuration, with no game DLLs committed
- [ ] It references BepInEx 5, Harmony and assembly_valheim
- [ ] A just recipe builds the DLL, and another copies it into a configured BepInEx/plugins folder
- [ ] The plugin logs its version on load in a real game
