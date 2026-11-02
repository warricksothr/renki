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
updated_at: 2026-11-02T06:10:52Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-review
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

Valheim's whisper is a short-range line with no recipient, and the plugin ignores `to` (`GameTaskHost.cs:146-149`). The doc (§ say/whisper, around line 524) and Go require it. `internal/chat/brain.go:531` treats every whisper it hears as addressed to the bot, so a whisper between two nearby players can trigger commands. Fix: document whisper as range-limited, drop the `to` requirement, and do not count an overheard whisper as addressed.

### 4. Doc and fake plugin drift

- §7.2 lists attack style `ranged`, which both sides reject.
- §7.1's list of instant tasks leaves out `drop`.
- `envelope.go:102-104`: the `Decode` comment says it returns the envelope with the error, but it returns nil.
- There are no `task_harvest`, `task_use_item` or `reject/controller/version_missing` fixtures.
- `cmd/fakeplugin` sends no `accepted` before instant results, answers unknown tasks with `bad_args` instead of `unsupported`, and ignores `allowedTasks`.

## Acceptance criteria

- [ ] A NaN or Infinity in a plugin message does not drop the frame
- [ ] Shared reject fixtures exist and both Go and C# refuse the same bad task args
- [ ] Whisper semantics in the doc, Go and the chat brain match what Valheim does
- [ ] BRIDGE-PROTOCOL.md, envelope comments and fakeplugin match the implementations
