---
schema: 4
id: TKT-01M61T36K1WHN4SJATSZWZEA8W
title: "Controller: WebSocket server, handshake and token check"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - protocol
assignees: []
milestone: M1
parent: TKT-01M61T36B839WT7KEZ14TDMB8B
origin: null
dependencies:
  - TKT-01M61T36G7WJ6C1ZV2NXNM37KD
  - TKT-01M61T36JGVFSCN0VGD93Z1RM8
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

The controller side of docs/BRIDGE-PROTOCOL.md sections 2 and 5: accept the plugin on `/ws/`,
check the token and version in `hello`, answer with `welcome`, and keep the
connection healthy.

## Acceptance criteria

- [ ] Listens on a configured address, by default 127.0.0.1:8765
- [ ] A bad token closes with code 4401; an unknown major version is refused
- [ ] `welcome` is built from configuration: agentName, owners, rates, allowedTasks
- [ ] `ping` and `pong` work, and a dead connection is noticed and logged
