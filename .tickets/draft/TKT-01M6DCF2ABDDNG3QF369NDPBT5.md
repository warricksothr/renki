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
updated_at: 2026-11-02T17:45:02Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-plan
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

### Goal map (docs/ROADMAP.md, M4)

- 4.1 The gate runs itself: TKT-01M6DCFST1, with children TKT-01M6EFGJY8 (workflow), TKT-01M6EFGJZK (required plugin), TKT-01M6EFGK0R and TKT-01M6EFGK2M (lint). Related: TKT-01M61V9F91 (Gate pushes on a local vuoro review).
- 4.2 A 24 h soak: TKT-01M6DDFTF6 builds `just vm soak`; TKT-01M6EFGYV6 runs it. The run waits on TKT-01M6DCFSHH, TKT-01M6DCFSNT, TKT-01M6DCFS8Y and TKT-01M6DDBX5T.
- 4.3 Stores survive a hard stop: TKT-01M6DCFSHH (offline kill test), plus the live kills in TKT-01M6EFGYV6. TKT-01M6EFGYQR (world UID) is not needed for 4.3.
- 4.4 Only owners command it: TKT-01M6DCFSME (untrusted lines, follow targets, refusal wording), TKT-01M6EFGYX3 (owners bound to player IDs), TKT-01M6EFGYYY (overheard whispers).
- 4.5 The link heals itself: TKT-01M6DCFS8Y (reconnect and the offline pause test), TKT-01M6EFGYNM (the live pause). TKT-01M6EFGYK8 (queue and clamps) is 4.6 only.
- 4.6 Review closed: every other child, including TKT-01M6DCFS5D, TKT-01M6DCFS77, TKT-01M6DCFSAR, TKT-01M6DCFSK1, TKT-01M6DCFSQ6, TKT-01M6DCFSRN, TKT-01M6DCFSVE, TKT-01M6DCFSWV and TKT-01M6DDBX7K.

### Order

Start TKT-01M6EFGJY8 first. Live tickets (label `live`) depend on TKT-01M6EF5DTG (VM: exclusive lease for deploys and live trials) and run one at a time after the day's M3 trials.

## Acceptance criteria

- [ ] Every child ticket is done, or closed with a note saying why it stands
- [ ] just ci runs automatically on pull requests and fails when Renki.Plugin does not compile

## Definition of done

- [ ] 4.1 just ci runs on every pull request, lints Go, panel, Renki.Core and Python scripts, and either compiles Renki.Plugin or fails loudly where it is skipped, per the owner's choice of plugin build (TKT-01M6DCFST1 and children)
- [ ] 4.2 A 24 h just vm soak passes with no controller crash or restart, <=100 open TCP connections, audit growth <=2 GB/day, no store write error and no plugin internal failure (TKT-01M6EFGYV6)
- [ ] 4.3 A crash test plus 3 live controller kills during saves, before the soak window, leave memory.json, goals.json and settings readable with no lost home (TKT-01M6DCFSHH, TKT-01M6EFGYV6)
- [ ] 4.4 Owners bind to player IDs, and a replayed chat-injection test cannot steer an owner's turn (TKT-01M6DCFSME, TKT-01M6EFGYX3, TKT-01M6EFGYYY)
- [ ] 4.5 After a 20 s controller pause the plugin reconnects and accepts tasks within 10 s of resume, in a bridge test and one live pause (TKT-01M6DCFS8Y, TKT-01M6EFGYNM)
- [ ] 4.6 Every child of this epic is done, or has a note saying why it stands, which includes a move to a later milestone; the panel epic's criteria are ticked or moved

## Notes

**agent:claude-code/renki-plan** at 2026-11-02T16:35:32Z

Goal 4.6 now counts a child moved to a later milestone with a note saying why as standing. As of 2026-11-02 that covers TKT-01M6EFGYQR (world UID, moved to M5) and TKT-01M6DDFTDD (haul report, M2). TKT-01M6DCFSEG and TKT-01M6DCFSG0 are in M3 and must be done.

**agent:claude-code/renki-plan** at 2026-11-02T16:35:45Z

Supersedes the previous note's list: TKT-01M6DDFTDD (haul report) is a child of the M2 epic, not this one. The only child of this epic in a later milestone is TKT-01M6EFGYQR (world UID, M5), and its move note says why it stands. TKT-01M6DCFSEG and TKT-01M6DCFSG0 are in M3 and must be done.
