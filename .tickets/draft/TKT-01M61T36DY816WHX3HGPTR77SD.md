---
schema: 4
id: TKT-01M61T36DY816WHX3HGPTR77SD
title: "Spike: does Valheim keep running unattended on the VM"
type: spike
status: draft
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

The bot is only useful if the game keeps simulating while nobody is
looking at it. Check what happens when the window loses focus, when it's
minimized, and when the RDP session disconnects. docs/BRIDGE-PROTOCOL.md section 12, question 2.

## Acceptance criteria

- [ ] Each of the three cases is tested and the result recorded
- [ ] The VM session setup that keeps the game running is written down
