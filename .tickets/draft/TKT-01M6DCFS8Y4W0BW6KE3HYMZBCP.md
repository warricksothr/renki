---
schema: 4
id: TKT-01M6DCFS8Y4W0BW6KE3HYMZBCP
title: "Bridge: reconnect after a dead-man stop, back off on 4409"
type: bug
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - protocol
  - plugin
  - controller
assignees: []
milestone: M4
parent: TKT-01M6DCF2ABDDNG3QF369NDPBT5
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T05:42:34Z
updated_at: 2026-11-02T15:57:56Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m4
  name: ""
extensions: {}
---

## Description

The plugin and the controller decide whether the link is alive independently, so a half-dead connection can last indefinitely.

### 1. Dead-man stop without reconnect (medium)

`plugin/Renki.Plugin/RenkiPlugin.cs:268` and `plugin/Renki.Core/Bridge/BridgeClient.cs:150-205`. After 10 s with nothing from the controller, the plugin stops its tasks and accepts nothing "until the next welcome". It keeps the socket open, though, and `welcome` is only sent on a new connection. Pings still flow from the socket thread. If the controller host pauses for 10–25 s (WSL2 or VM stall, sleep, debugger), it resumes inside its own 15 s limit, and from then on every task is rejected `not_allowed` until something restarts.

Fix: when the dead-man limit fires, abort the socket so the client reconnects and receives a fresh `welcome`. As a backstop, the controller closes a connection that answers pings but sends no `state` for N seconds.

### 2. Replaced connections flap once a second (low–medium)

`BridgeClient.cs:129,140`. A close with `4409` (replaced) is treated like any other disconnect, and the backoff resets to 1 s. Two plugin instances pointed at one controller replace each other forever, and each swap cancels every task. Fix: read `CloseStatus`. On `4409`, keep backing off and log loudly. Treat `4401` the same way.

### 3 and 4. Outbound queue limit and rate bounds

Moved to TKT-01M6EFGYK8 (Bridge: coalesce a deep outbound queue, clamp rates and radius).

### Goal

Goal 4.5 (the link heals itself): after a 20 s controller pause the plugin reconnects and accepts tasks within 10 s. Items 3 and 4 moved to TKT-01M6EFGYK8 (Bridge: coalesce a deep outbound queue, clamp rates and radius). The live pause is TKT-01M6EFGYNM (VM: pause the controller 20 s and time the plugin's reconnect).

### Files

Line numbers refer to commit f1a5483; at 0b932ef the dead-man check is at `RenkiPlugin.cs:278-283` and `DeadManSeconds` at :31.

- `plugin/Renki.Plugin/RenkiPlugin.cs` (the dead-man rule and `EndSession`)
- `plugin/Renki.Core/Bridge/BridgeClient.cs` (reconnect, close status handling)
- `plugin/Renki.Core/Tasks/TaskRunner.cs` (the dead-man summary at :113)
- `internal/bridge/server.go` (the controller backstop: close a connection that answers pings but sends no `state`)
- `plugin/Renki.Tests/InteropTests.cs` (runs the real controller binary)

### Verification

- `just check plugin test-core Bridge` for close-status handling (4409 and 4401 back off).
- An interop test in `plugin/Renki.Tests/InteropTests.cs` stops the controller process for 20 s (SIGSTOP and SIGCONT on Linux), then asserts the client reconnects, receives a fresh `welcome`, and has a task accepted within 10 s of resume. Run with `just check plugin test`.
- `just check controller test-packages ./internal/bridge` for the backstop.
- `just check ci`.

## Acceptance criteria

- [ ] When the plugin's dead-man limit fires it reconnects and receives a fresh welcome
- [ ] A 4409 close backs off instead of reconnecting every second
- [ ] An interop test stops the controller 20 s, and the plugin reconnects, gets a fresh welcome and has a task accepted within 10 s of resume
- [ ] The controller closes a connection that answers pings but sends no state for a set time, covered by a bridge test
