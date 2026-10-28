---
schema: 4
id: TKT-01M61T36KKDCYVKZC4AE9Z2F9S
title: "Plugin: connect, handshake, reconnect and main-thread dispatch"
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - protocol
assignees: []
milestone: M1
parent: TKT-01M61T36B839WT7KEZ14TDMB8B
origin: null
dependencies:
  - TKT-01M61T36CCV76K49ZDHSNV04RB
  - TKT-01M61T36HB0PY76DJNZ1AGHCG2
  - TKT-01M61T36JGVFSCN0VGD93Z1RM8
blocks_on: none
references: []
moved_to: null
claim: null
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

The plugin side of docs/BRIDGE-PROTOCOL.md sections 2 and 5. The socket lives on a
background thread. Inbound messages go through a queue drained in `Update()`,
so game APIs are only ever called from Unity's main thread.

## Acceptance criteria

- [x] Dials the configured controllerUrl and sends `hello` with token and capabilities
- [x] Reconnects with backoff from 1 s up to 30 s
- [x] No game API is called from the socket thread
- [x] The C# message types pass against the shared Go fixtures
- [x] An unknown `type` gets an `error` reply with code `bad_message`, and the connection stays open

## Summary

Renki.Core.Bridge.BridgeClient: a background task that dials controllerUrl, sends hello (token, capabilities from TaskRunner), answers pings itself, and queues every other inbound message for the main thread, which drains it in RenkiPlugin.Update. Reconnects back off 1, 2, 4 … 30 s, resetting after a welcome. Sends are dropped when no session is up, so nothing from an old session reaches a new one. An undecodable frame gets error/bad_message with re, and the connection stays open. The C# protocol types pass every shared fixture. Under Mono it is the spike TKT-01M61T36CC.
