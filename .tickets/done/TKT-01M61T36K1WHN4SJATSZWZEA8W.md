---
schema: 4
id: TKT-01M61T36K1WHN4SJATSZWZEA8W
title: "Controller: WebSocket server, handshake and token check"
type: task
status: done
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
updated_at: 2026-10-28T18:33:32Z
created_by:
  id: agent:claude-code/valheim-setup
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

The controller side of docs/BRIDGE-PROTOCOL.md sections 2 and 5: accept the plugin on `/ws/`,
check the token and version in `hello`, answer with `welcome`, and keep the
connection healthy.

## Acceptance criteria

- [x] Listens on a configured address, by default 127.0.0.1:8765
- [x] A bad token closes with code 4401; an unknown major version is refused
- [x] `welcome` is built from configuration: agentName, owners, rates, allowedTasks
- [x] `ping` and `pong` work, and a dead connection is noticed and logged
- [x] An unknown `type` gets an `error` reply with code `bad_message`, and the connection stays open

## Summary

internal/bridge serves /ws/ on the configured address (config default 127.0.0.1:8765; a non-loopback address needs a token, from the file or RENKI_TOKEN). The first frame must be hello within 10 s. A bad token closes with 4401, a wrong major version gets an `unsupported_version` error and closes with 4400, and anything else first closes with 4400. welcome is built from config by app.Welcome. The controller pings every 3 s, consumes pongs, and logs and drops a plugin silent for 15 s. An unknown type gets `error` bad_message with `re`, and the connection stays open. A newer connection replaces the old one with 4409. The close codes and timings are now in docs/BRIDGE-PROTOCOL.md section 2. Tested against a real WebSocket client in internal/bridge.
