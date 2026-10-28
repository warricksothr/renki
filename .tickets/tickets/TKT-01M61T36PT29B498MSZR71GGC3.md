---
schema: 4
id: TKT-01M61T36PT29B498MSZR71GGC3
title: "VM launch script: start and stop the bot on demand"
type: task
status: ready
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

So the bot is around only when it's wanted: one script brings it up,
another takes it down cleanly.

## Acceptance criteria

- [ ] A PowerShell script launches Valheim via Steam with +connect and joins
- [ ] It starts the controller too, when the controller runs on the VM
- [ ] A stop script logs the bot out cleanly before closing the game
