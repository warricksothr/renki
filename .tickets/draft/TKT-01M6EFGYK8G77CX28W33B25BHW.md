---
schema: 4
id: TKT-01M6EFGYK8G77CX28W33B25BHW
title: "Bridge: coalesce a deep outbound queue, clamp rates and radius"
type: bug
status: draft
status_reason: null
priority: low
due_on: null
labels:
  - protocol
  - plugin
  - controller
assignees: []
milestone: M4
parent: TKT-01M6DCF2ABDDNG3QF369NDPBT5
origin: null
dependencies:
  - TKT-01M6DCFS8Y4W0BW6KE3HYMZBCP
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:54:52Z
updated_at: 2026-11-02T15:58:03Z
created_by:
  id: agent:claude-code/renki-plan-m4
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m4
  name: ""
extensions: {}
---

## Description

The two lower-severity bridge items from TKT-01M6DCFS8Y (Bridge: reconnect after a dead-man stop, back off on 4409), items 3 and 4, split out so that ticket can focus on goal 4.5's reconnect. Line numbers below refer to commit f1a5483.

### 1. The outbound queue has no limit (low)

`plugin/Renki.Core/Bridge/BridgeClient.cs` around lines 100-120. If the controller's read handler stalls, writes block while pings still flow. State, perception and terrain frames then pile up in the game process without limit. Fix: coalesce state and perception frames, keeping only the latest of each kind, once the queue passes a depth.

### 2. Rates and perception size are unbounded (low)

- `internal/config/config.go` (around :229) checks only `stateHz > 0`.
- The plugin accepts any `perceptionHz`, `radius` and `maxPerKind` from `welcome` and `config` (`plugin/Renki.Plugin/RenkiPlugin.cs` around :336-383).
- A large radius can produce a frame over the 1 MiB limit, which closes the connection; the plugin reconnects and repeats.

Fix: validate ranges in the controller config, and clamp them in the plugin, ideally in Renki.Core so the clamp is testable.

### Files

- `plugin/Renki.Core/Bridge/BridgeClient.cs`
- `plugin/Renki.Plugin/RenkiPlugin.cs` (welcome and config handling)
- `internal/config/config.go`, `internal/config` tests
- `docs/BRIDGE-PROTOCOL.md` if the accepted ranges are documented there

### Verification

- `just check plugin test-core Bridge` (add a queue-depth coalescing test and a clamp test in `plugin/Renki.Tests`).
- `just check controller test-packages ./internal/config`.
- `just check ci`.

## Acceptance criteria

- [ ] State and perception frames coalesce when the outbound queue passes a set depth, covered by a Core test
- [ ] Controller config rejects out-of-range state and perception rates, radius and maxPerKind
- [ ] The plugin clamps rates, radius and maxPerKind from welcome and config, covered by a Core test
