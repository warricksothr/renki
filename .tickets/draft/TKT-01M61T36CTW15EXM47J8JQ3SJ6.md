---
schema: 4
id: TKT-01M61T36CTW15EXM47J8JQ3SJ6
title: "Set up the bot VM: Steam account, Valheim, BepInEx"
type: chore
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

The bot needs its own Steam account that owns Valheim. Family Sharing can't
run the same game for two people at once. Install Valheim and the
BepInExPack_Valheim pack on the Windows VM, match the server's mod set, and
create the bot's character.

## Acceptance criteria

- [ ] A second Steam account owning Valheim is signed in on the VM
- [ ] Valheim and BepInExPack_Valheim are installed there
- [ ] The VM's mod profile matches the server's (r2modman export/import)
- [ ] The bot's character exists and joins the server by hand
