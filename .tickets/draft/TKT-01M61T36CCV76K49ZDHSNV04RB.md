---
schema: 4
id: TKT-01M61T36CCV76K49ZDHSNV04RB
title: "Spike: WebSocket client inside Valheim's Mono runtime"
type: spike
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - plugin
  - protocol
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

Everything depends on the plugin holding a WebSocket open from inside the
game. Find out whether `System.Net.WebSockets.ClientWebSocket` works under
Valheim's Mono, or whether we need websocket-sharp or another library.
docs/BRIDGE-PROTOCOL.md section 12, question 1.

A throwaway plugin is fine. It doesn't need the VM; any Valheim install with
BepInEx will do.

## Acceptance criteria

- [ ] A test plugin in a running game exchanges messages both ways with a test server
- [ ] It reconnects after the server restarts
- [ ] The choice of library and the reason are recorded in a note
