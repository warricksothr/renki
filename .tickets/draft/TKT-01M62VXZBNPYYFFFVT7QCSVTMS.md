---
schema: 4
id: TKT-01M62VXZBNPYYFFFVT7QCSVTMS
title: "Agent: wake the LLM on world events, within a budget"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - llm
  - controller
assignees: []
milestone: M6
parent: TKT-01M62VXZA33E8JR5BR2RJD6AC1
origin: null
dependencies:
  - TKT-01M62VXZAV8CYT5K4DYAQEZC2Q
  - TKT-01M6EFC7R886ACM2932G5ZESAP
  - TKT-01M6DCFSMEWJG8Z30WR59KBG5Y
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:48Z
updated_at: 2026-11-02T15:54:33Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-plan-m6
  name: ""
extensions: {}
---

## Description

An event bus in the controller wakes the agent without anyone speaking:
stock low, a behaviour finished or failed, idle at home for a while,
nightfall or cold with no fire lit, inventory full with no storage in
reach, an owner died (after the reflex has dealt with it), an owner
answered a question, a reminder came due.

Each kind of event says which tool groups are offered: stock low gets
Query, Behaviours and Social, while danger gets none. Events close together
merge into one turn. There is a minimum time between turns, quiet hours,
and a share of the daily token cap. A switch turns it all off.

This is M6's second outcome: the agent wakes on events, within a budget. The budget half (minimum gap, quiet hours, the token share and an hourly cap) is split out to TKT-01M6EFG477 (Agent: wake budget with minimum gap, quiet hours and token caps). This ticket is the bus, the per-event tool groups, merging, the switch, and the audit records that let the wake latency be measured.

### Scope of this ticket

- A wake bus in the controller fed from what already exists: `behaviour.Engine` derives `inventory_full`, `nightfall`, `idle_at_home` and `drop_appeared` in `triggers` (`internal/behaviour/engine.go`), and `app.go` fans plugin events out to the brain, retreat, presence and the engine (`internal/app/app.go` around line 222). Add `behaviour_finished` and `behaviour_failed` from the engine's run lifecycle.
- Events that have no producer yet are wired when their producer lands, not invented here: `stock_low` is emitted nowhere today (it is only a name in `eventNames`, `internal/behaviour/definition.go`) and comes from TKT-01M62VXZF8 (Controller: stock targets and a stock-low event from chest memory, M6); `owner_answered` and `reminder_due` come from the autonomy ticket and its reminder child; "cold with no fire lit" needs fire perception from M5.
- A table, in code and in docs, from event kind to the tool groups offered. Danger events (damage, low health, flee) offer none and never wake the agent: reflexes stay in code.
- Events within a short window (default 5 s) merge into one turn whose prompt lists them all.
- A settings-store switch (for example `agent.wake`, default off until M6's goals are met) turns waking off and leaves chat exactly as before.
- Each turn writes the `agent wake` audit record defined by TKT-01M6EFC7R8 (Audit: agent report of tokens, wake latency and proposal outcomes): event kinds, the first event's time, the turn's start, groups offered, and the turn's tokens.
- The turn runs through the voice's tool registry (TKT-01M62VXZAV) with a synthetic speaker (the bot itself), owner authority limited by the autonomy level once that ticket lands. Until then, a woken turn may only use Express and Query.

### Files

- `internal/behaviour/engine.go` (`Event`, `drainEvents`, `triggers`) and `internal/behaviour/definition.go` (`eventNames`)
- `internal/app/app.go`: event fan-out, and where a new wake component is wired
- `internal/chat/brain.go`: `converse`, which a woken turn mirrors without a chat line
- `internal/voice/voice.go`: the registry from TKT-01M62VXZAV
- `internal/settings/settings.go`: the switch
- `docs/BEHAVIOURS.md` or a new section in docs/AUDIT.md: the event-to-groups table

### Verification

- `just check controller test-packages ./internal/behaviour ./internal/app ./internal/chat` with fake time and a fake voice: each listed event wakes one turn with its groups; two events 2 s apart make one turn; a danger event makes none; with the switch off nothing wakes and chat replies are unchanged; the `agent wake` record carries the event time and turn start.
- `just check ci`.
- Live evidence (an idle-at-home wake in the game) belongs to the M6 live evidence ticket, not this one.

## Acceptance criteria

- [ ] Listed events wake the agent with the tool groups that event allows
- [ ] A switch turns waking off, leaving chat as before
- [ ] Events within the merge window become one turn, and danger events never wake the agent
- [ ] Each woken turn writes the agent wake audit record with the event time and the turn start
