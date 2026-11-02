---
schema: 4
id: TKT-01M62VXZCVVP0XXPDNYT5JYD51
title: "Agent: autonomy levels, owner pins, ask_owner and reminders"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - llm
  - controller
  - chat
assignees: []
milestone: M6
parent: TKT-01M62VXZA33E8JR5BR2RJD6AC1
origin: null
dependencies:
  - TKT-01M62VXZBNPYYFFFVT7QCSVTMS
  - TKT-01M62VXYR69PKZGJFEWQ1759GV
  - TKT-01M6DCFSMEWJG8Z30WR59KBG5Y
  - TKT-01M62VXZAV8CYT5K4DYAQEZC2Q
  - TKT-01M6EFC7R886ACM2932G5ZESAP
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:48Z
updated_at: 2026-11-02T15:55:06Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-plan-m6
  name: ""
extensions: {}
---

## Description

An autonomy level per bot governs what the agent may do when nobody asked:

- **ask** (the default): it proposes, and waits for an owner's yes
- **tell**: it acts and says what it did
- **quiet**: it acts and only writes to the audit log

`ask_owner(question, options)` puts a question to the owners in chat. The
answer comes back as an event, and a question nobody answers lapses.
`remind_me(after, about)` schedules an event. An owner's pin (settings
store) refuses any change by the agent at every level.

This is M6's third outcome, and the one the owner sees most: the share of proposals an owner rejects is an M6 exit goal, so every proposal's fate must be recorded. `remind_me` is split out to TKT-01M6EFH8QF (Agent: remind_me schedules a later wake) so this ticket stays one pull request.

### What exists

- The settings store already refuses agent writes to pinned keys (`ErrPinned` in `change`, `internal/settings/settings.go`) and never lets the agent touch `ownersOnly` keys (owners, allowedTasks, model provider and base URL, player profiles). Behaviour instances (`SetInstance` in `internal/settings/instances.go`) have no pin yet; that is the behaviour tools ticket's concern.
- Chat from owners is parsed by `Brain.Chat` (`internal/chat/brain.go`); there is no pending-question state.

### What to build

- An `agent.autonomy` setting (ask, tell, quiet; default ask), owner or panel only, never the agent.
- At ask, a self-started action (from a woken turn, TKT-01M62VXZBN) is not dispatched. It becomes a pending proposal put to the owners in chat with `ask_owner`, with a short id and options (yes, no, or listed choices). At tell it is dispatched and announced in chat within the same turn. At quiet it is dispatched and only logged. An action an owner asked for in chat is unaffected by the level.
- `ask_owner(question, options)` as a Social tool. An owner's reply ("yes", "no", an option's text, or the id) within a lapse window (default 5 min) resolves it and wakes the agent with an `owner_answered` event; the first owner to answer decides. Unanswered questions lapse and are logged. At most one open question per topic, and a cap on open questions (default 2).
- Each proposal writes the `agent proposal` audit record (id, action, outcome accepted, rejected, lapsed or refused, who answered, when) defined by TKT-01M6EFC7R8 (Audit: agent report of tokens, wake latency and proposal outcomes).
- The agent's settings writes go through the store as `ByAgent`, so pins hold at every level; a refusal comes back to the model as the tool result.
- Never available to the agent at any level: logout, bosses, giving items away, Safety settings (decision of 2026-10-28 on the epic).

### Files

- `internal/settings/settings.go`: `Keys`, `ownersOnly`, `ByAgent`, `ErrPinned`
- `internal/chat/brain.go`: `Chat`, `say`, `propose`, owner checks
- `internal/voice/voice.go`: the registry from TKT-01M62VXZAV, for `ask_owner`
- the wake component from TKT-01M62VXZBN
- `docs/PLAYERS.md`: how owners answer

### Verification

- `just check controller test-packages ./internal/chat ./internal/settings ./internal/voice` with fake time: at ask a woken action waits and is dispatched only after "yes"; "no" drops it; silence lapses it after the window; at tell it is dispatched and announced; at quiet only logged; a pinned key refuses the agent at all three levels; a non-owner's "yes" resolves nothing; each case writes the right `agent proposal` record.
- `just check ci`.
- Live evidence (an owner answering in game) belongs to the M6 live evidence ticket.

## Acceptance criteria

- [ ] At ask, self-started actions wait for an owner's answer, and at tell they happen and are announced
- [ ] An owner's answer to ask_owner wakes the agent with it, and unanswered questions lapse
- [ ] Pinned settings refuse the agent at every level
- [ ] Every proposal writes an agent proposal record with its outcome: accepted, rejected, lapsed or refused
- [ ] Logout, bosses, giving items away and Safety settings are never offered to the agent at any level
