---
schema: 4
id: TKT-01M6DCFS8Y4W0BW6KE3HYMZBCP
title: "Bridge: recover half-dead connections and bound the outbound queue"
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
milestone: null
parent: TKT-01M6DCF2ABDDNG3QF369NDPBT5
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T05:42:34Z
updated_at: 2026-11-02T05:42:34Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-review
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

### 3. Outbound queue has no limit (low)

`BridgeClient.cs:100-120`. If the controller's read handler stalls, writes block while pings still flow. State, perception and terrain frames then pile up in the game process without limit. Fix: coalesce state and perception frames, keeping only the latest of each kind, once the queue passes a depth.

### 4. Rates and perception size are unbounded (low)

- `internal/config/config.go:229` checks only `stateHz > 0`.
- The plugin accepts any `perceptionHz`, `radius` and `maxPerKind` from `welcome`/`config` (`RenkiPlugin.cs:336-383`).
- A large radius can produce a frame over the 1 MiB limit, which closes the connection, and the plugin then reconnects and repeats.

Fix: validate ranges in the controller config, and clamp them in the plugin.

## Acceptance criteria

- [ ] When the plugin's dead-man limit fires it reconnects and receives a fresh welcome
- [ ] A 4409 close backs off instead of reconnecting every second
- [ ] State and perception frames coalesce when the outbound queue is deep
- [ ] Controller config and plugin clamp state/perception rates, radius and maxPerKind
