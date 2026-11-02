---
schema: 4
id: TKT-01M62VXZAV8CYT5K4DYAQEZC2Q
title: "Voice: tool registry with groups, query tools and real results"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - llm
  - terva
  - controller
assignees: []
milestone: M6
parent: TKT-01M62VXZA33E8JR5BR2RJD6AC1
origin: null
dependencies:
  - TKT-01M6DCFSMEWJG8Z30WR59KBG5Y
  - TKT-01M630V8292NGHWAX3R7N8W7K1
  - TKT-01M6EFC7R886ACM2932G5ZESAP
  - TKT-01M630V80TFBMWT1B3XY4KP2QJ
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:48Z
updated_at: 2026-11-02T15:52:51Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-plan-m6
  name: ""
extensions: {}
---

## Description

Move the voice's task catalogue (`internal/voice`) into a registry where each tool has a group, a level, a schema and a handler. This is the base every other M6 agent tool builds on, and the first M6 outcome: the agent can look before it acts and learns whether its actions were accepted.

Today `voice.Reply` builds one terva runtime per turn with `MaxSteps: 2` and a `menu` of task tools whose `Execute` only records the first proposal (`recorder`). The brain checks that proposal after the turn ends (`chat.Brain.propose`), so the model never hears whether it was accepted.

- **Query tools** answer at once from the session and memory. This ticket brings `status` and `inventory`, which the brain can already produce (`internal/chat/status.go`, `Inventory()` on the brain's dependencies). The rest (nearby, recall, behaviours, recent events) are TKT-01M6EFCK6H (Voice: query tools for nearby, recall, behaviours and recent events). Answers are short summaries, not raw streams.
- **Action tools** are checked when they are called, by the brain's own checks (owner, allowedTasks, args), through a callback the brain hands the voice. The result goes back to the model: accepted, or refused with the reason.
- Several actions per turn, up to a cap (3). More steps per turn (about 6), so the model can look and then act.
- Which groups are offered depends on who is speaking: a player who is not an owner gets Express and Query.

Every tool call, with its result, goes to the audit log, as the `agent tool` record defined by TKT-01M6EFC7R8 (Audit: agent report of tokens, wake latency and proposal outcomes).

### Ordering

This ticket gives the model more tools under an owner's authority, so it waits on TKT-01M6DCFSME (Chat: untrusted lines in owner turns, and name-only owner identity): fenced untrusted lines, follow targets limited to the speaker or a commander, and proposals checked against the offered menu. The registry must keep those checks: an action is checked against the tools offered in this turn, not all of `AllowedTasks()`.

TKT-01M630V829 (Voice: never promise an action no tool can carry out) also edits `voice.go` and the prompt. Land it first or rebase onto it.

### Files

- `internal/voice/voice.go`: `menu`, `catalogue`, `tool`, `recorder`, `Reply` (MaxSteps), `prompt`
- `internal/voice/voice_test.go`: the fake runtime the tests already use
- `internal/chat/brain.go`: `Voice`, `Request`, `Reply`, `Proposal`, `converse`, `propose`
- `internal/session/session.go`: `Dispatch` and `ErrNotAllowed`, the checks an action goes through

### Verification

- `just check controller test-packages ./internal/voice ./internal/chat`: a scripted fake runtime calls a query, then an action that is accepted, then one that is refused, then a fourth action over the cap; the test asserts what the model was told each time and the audit records.
- A test that a non-owner turn offers only Express and Query tools.
- `just check ci`.

## Acceptance criteria

- [ ] Tools are registered with a group, a level and a schema, and the voice offers them by group
- [ ] An action's acceptance or refusal, with the reason, goes back to the model
- [ ] Up to three actions per turn, and every call is in the audit log
- [ ] status and inventory query tools return real data from the session
- [ ] A non-owner's turn is offered only Express and Query tools, and an action is checked against the tools offered in that turn
