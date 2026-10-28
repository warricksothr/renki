---
schema: 4
id: TKT-01M61T36B839WT7KEZ14TDMB8B
title: "M1: bot joins, follows, stops, reports status and chats"
type: epic
status: ready
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - controller
assignees: []
milestone: M1
parent: null
origin: null
dependencies: []
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

The first milestone: a second Valheim client on the Windows VM, driven by
the C# plugin and the Go controller, that joins the server unattended, follows
an owner, stops, reports its status and holds a conversation in chat.

Perception, inventory, gathering and combat are M2 and later. The protocol is
docs/BRIDGE-PROTOCOL.md; the tickets below cite its sections.

## Acceptance criteria

- [ ] The bot joins the server from the launch script with nobody at the VM
- [ ] It follows an owner across 200 m of mixed terrain and recovers from getting stuck
- [ ] `follow`, `stop` and `status` work from chat, from owners only
- [ ] Chat addressed to the bot is answered in character through terva
- [ ] Killing the controller stops the bot within 10 seconds
- [ ] The plugin refuses any task outside the M1 whitelist
