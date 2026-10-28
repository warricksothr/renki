---
schema: 4
id: TKT-01M61T36KKDCYVKZC4AE9Z2F9S
title: "Plugin: connect, handshake, reconnect and main-thread dispatch"
type: task
status: draft
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

The plugin side of docs/BRIDGE-PROTOCOL.md sections 2 and 5. The socket lives on a
background thread. Inbound messages go through a queue drained in `Update()`,
so game APIs are only ever called from Unity's main thread.

## Acceptance criteria

- [ ] Dials the configured controllerUrl and sends `hello` with token and capabilities
- [ ] Reconnects with backoff from 1 s up to 30 s
- [ ] No game API is called from the socket thread
- [ ] The C# message types pass against the shared Go fixtures
