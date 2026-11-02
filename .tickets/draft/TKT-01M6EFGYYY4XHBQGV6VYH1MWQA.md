---
schema: 4
id: TKT-01M6EFGYYY4XHBQGV6VYH1MWQA
title: "Chat: an overheard whisper is not addressed to the bot"
type: bug
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - chat
  - safety
  - protocol
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
created_at: 2026-11-02T15:54:52Z
updated_at: 2026-11-02T15:54:52Z
created_by:
  id: agent:claude-code/renki-plan-m4
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m4
  name: ""
extensions: {}
---

## Description

Split out of TKT-01M6DCFSAR (Protocol: NaN-safe encoding, shared reject fixtures, doc drift), item 3, because it is a trust fix that belongs with goal 4.4 rather than protocol tidying.

### Problem

Valheim's whisper is a short-range line with no recipient, and the plugin ignores `to` (`plugin/Renki.Plugin/GameTaskHost.cs` around :146-149). docs/BRIDGE-PROTOCOL.md (§ say/whisper, around line 524) and Go require `to`. `internal/chat/brain.go:531` treats every whisper it hears as addressed to the bot (`c.Channel == protocol.ChannelWhisper || ...`), so a whisper between two players standing near the bot can trigger commands.

### Fix

- Document whisper as range-limited with no recipient, and drop the `to` requirement in the doc and in Go's task validation (`internal/protocol/task.go`).
- Do not count an overheard whisper as addressed to the bot: require the bot's name, as for other channels.
- Keep `replyChannel` (`internal/chat/brain.go` around :1029) answering a whisper with a whisper.

### Files

- `internal/chat/brain.go`, `internal/chat/brain_test.go`
- `internal/protocol/task.go`, `internal/protocol/testdata`
- `plugin/Renki.Plugin/GameTaskHost.cs`, `plugin/Renki.Core/Protocol/TaskArgs.cs` if whisper args are validated there
- `docs/BRIDGE-PROTOCOL.md`

### Verification

- `just check controller test-packages ./internal/chat ./internal/protocol`.
- A brain test: a whisper without the bot's name from a nearby player issues no command.
- `just check plugin test` (fixtures round-trip in both languages) and `just check ci`.

## Acceptance criteria

- [ ] A whisper without the bot's name issues no command, covered by a brain test
- [ ] BRIDGE-PROTOCOL.md, Go and C# agree that a whisper is range-limited and needs no to
