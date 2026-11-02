---
schema: 4
id: TKT-01M6DCFSAR4VZEW67TEGEDDES8
title: "Protocol: NaN-safe encoding, shared reject fixtures, doc drift"
type: bug
status: draft
status_reason: null
priority: low
due_on: null
labels:
  - protocol
  - plugin
  - docs
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
updated_at: 2026-11-02T15:57:05Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m4
  name: ""
extensions: {}
---

## Description

Contract drift between Go, C# and docs/BRIDGE-PROTOCOL.md, found in review. None of it is unsafe today, but each item turns into a confusing `bad_args` or a lost frame.

### 1. NaN or Infinity drops whole frames

`plugin/Renki.Core/Protocol/Codec.cs:64`. Newtonsoft's default writes non-finite floats as the strings `"NaN"` and `"Infinity"`, which Go cannot decode into `float64`. The whole perception or state frame is then rejected as `bad_message`. Unguarded sources include `HpFrac` (`Perceiver.cs:143`) and the durability and fuel values. Fix: `FloatFormatHandling.DefaultValue`, or a finiteness check in `Encode`. Add a Core test that encodes a NaN.

### 2. Task-argument validation differs between the two sides

| Case | Go (`internal/protocol/task.go:380,403,422`) | C# (`plugin/Renki.Core/Protocol/TaskArgs.cs:257,289,329`) |
|---|---|---|
| Negative `stopDistance`, or follow `distance` | accepts | rejects |
| Two-element position `[x,z]` | zero-fills to `[x,z,0]` | rejects |
| `"searchRadius": null` | treated as absent | rejects |
| Escort name length | counted in bytes | counted in characters |

Fix: add shared *reject* fixtures that both suites must refuse, and line the bounds up.

### 3. Whisper `to` is ignored

Moved to TKT-01M6EFGYYY (Chat: an overheard whisper is not addressed to the bot).

### 4. Doc and fake plugin drift

- §7.2 lists attack style `ranged`, which both sides reject.
- §7.1's list of instant tasks leaves out `drop`.
- `envelope.go:102-104`: the `Decode` comment says it returns the envelope with the error, but it returns nil.
- There are no `task_harvest`, `task_use_item` or `reject/controller/version_missing` fixtures.
- `cmd/fakeplugin` sends no `accepted` before instant results, answers unknown tasks with `bad_args` instead of `unsupported`, and ignores `allowedTasks`.

### Goal

Goal 4.6 (review closed). Item 3 (whisper) moved to TKT-01M6EFGYYY (Chat: an overheard whisper is not addressed to the bot), because it is a trust fix for goal 4.4. Item 1 also protects the soak: a NaN in `HpFrac` drops a whole state frame.

### Files

Line numbers refer to commit f1a5483.

- `plugin/Renki.Core/Protocol/Codec.cs`, `plugin/Renki.Plugin/Perceiver.cs`
- `internal/protocol/task.go`, `plugin/Renki.Core/Protocol/TaskArgs.cs`
- `internal/protocol/testdata` (its `reject`, `controller` and `plugin` folders and README), `plugin/Renki.Tests/FixtureTests.cs`
- `internal/protocol/envelope.go`, `docs/BRIDGE-PROTOCOL.md` (§7.1, §7.2)
- `cmd/fakeplugin`

### Verification

- `just check plugin test-core Fixture` and a new Core test that encodes a NaN.
- `just check controller test-packages ./internal/protocol ./cmd/fakeplugin`.
- `just check plugin test` (both suites refuse the same reject fixtures) and `just check ci`.

Size: about a day. If the reject fixtures grow past that, split item 4 (doc and fakeplugin drift) into its own child.

## Acceptance criteria

- [ ] A NaN or Infinity in a plugin message does not drop the frame
- [ ] Shared reject fixtures exist and both Go and C# refuse the same bad task args
- [ ] BRIDGE-PROTOCOL.md, envelope comments and fakeplugin match the implementations
