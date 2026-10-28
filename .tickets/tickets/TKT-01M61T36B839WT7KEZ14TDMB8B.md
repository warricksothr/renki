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
updated_at: 2026-10-28T23:24:20Z
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

- [x] The bot joins the server from the launch script with nobody at the VM
- [ ] It follows an owner across 200 m of mixed terrain and recovers from getting stuck
- [x] `follow`, `stop` and `status` work from chat, from owners only
- [ ] Chat addressed to the bot is answered in character through terva
- [ ] Killing the controller stops the bot within 10 seconds
- [x] The plugin refuses any task outside the M1 whitelist

## Notes

**agent:claude-code/renki-groom** at 2026-10-28T23:24:20Z

Evidence from 2026-10-28 on the bot VM. (1) RenkiStart joins the server with nobody at the VM, every time. (3) follow, stop, wait here, status, set home and go home worked from chat for the owner Nimah; the non-owner jarl was refused at 17:42 with nothing sent. (6) The plugin's TaskRunner refuses anything outside capabilities and welcome.allowedTasks (TaskRunner tests); the controller's session refuses the same before sending. Still to see: a 200 m follow with a stuck recovery during the follow (recoveries were seen on the way home), the dead-man stop when the controller dies mid-follow, and the terva voice, which is not configured on the VM.
