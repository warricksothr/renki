---
schema: 4
id: TKT-01M6DCF2ABDDNG3QF369NDPBT5
title: "Code review 2026-11-01: robustness, trust and gate gaps"
type: epic
status: draft
status_reason: null
priority: high
due_on: "2026-11-22"
labels:
  - tooling
  - controller
  - plugin
assignees: []
milestone: M4
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T05:42:10Z
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

A full read-only review of the code at the tip of the open PR stack (t3/cargo-memory, #151, about 75k lines) on 2026-11-01. A seventh reviewer then covered #152 and #153, merged during the review. Six reviewers each covered one area: plugin tasks; the bridge contract; behaviours and survival; goals, memory and the atlas; the controller, chat and panel API; and the panel UI, tooling and CI. Every finding was traced through the code before it was filed. The lead reviewer re-checked the high and medium ones.

### What held up

- **Bridge:** messages match field for field between Go and C#, the fixtures round-trip in both languages, and the WebSocket server is sound.
- **Goals:** the store is fsync-safe and crash-aware.
- **Starlark:** the sandbox is bounded.
- **Panel API:** auth, the CSRF header and body limits are applied consistently.
- **Native tasks:** almost every task is bounded in time and attempts.

### The structural gaps

1. **Last line of defence.** Neither the plugin's 50 Hz task loop nor several long-running processes have one: nothing catches a thrown task, a connection can sit half-dead, and the OBS adapter leaks connections.
2. **Records with no lifecycle.** Goal records, audit bytes and the atlas retry queue grow until something refuses work.
3. **Uneven durability.** memory.json, which almost every plan depends on, has no fsync. A damaged copy stops the controller from starting.
4. **Who counts as trusted.** An owner is matched by a copyable character name. Chat from anyone reaches the model in turns the owner has authorised.
5. **The gate.** Nothing runs `just ci` automatically. It passes without compiling Renki.Plugin, and lint only checks formatting.

The child tickets each hold one agent-sized piece, with file:line pointers, a failure scenario and acceptance criteria. Line numbers refer to commit f1a5483.

## Acceptance criteria

- [ ] Every child ticket is done, or closed with a note saying why it stands
- [ ] just ci runs automatically on pull requests and fails when Renki.Plugin does not compile

## Definition of done

- [ ] 4.1 just ci runs on every PR, fails when Renki.Plugin does not compile, and lints Go, panel, plugin and scripts
- [ ] 4.2 A 24 h just vm soak passes (TKT-01M6DDFTF6)
- [ ] 4.3 Three forced controller kills during saves leave every store readable
- [ ] 4.4 Owners bind to player IDs, and a replayed chat-injection test cannot steer an owner's turn
- [ ] 4.5 After a 20 s controller pause the plugin reconnects and accepts tasks within 10 s
