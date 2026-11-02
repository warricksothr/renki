---
schema: 4
id: TKT-01M62VXZE38GXGK324JHD0RQZD
title: "Agent: tools to list, start, stop, clone and edit behaviours"
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
  - TKT-01M62VXYSN4RQV75HTA3BSD162
  - TKT-01M62VXYR69PKZGJFEWQ1759GV
  - TKT-01M6DCFSMEWJG8Z30WR59KBG5Y
  - TKT-01M62VXZCVVP0XXPDNYT5JYD51
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:48Z
updated_at: 2026-11-02T15:55:28Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-plan-m6
  name: ""
extensions: {}
---

## Description

The Behaviours group of agent tools, over the behaviour engine and the
settings store. Edits are checked against the template's schema like any
other, are recorded with source `agent`, and obey pins and the autonomy
level. A goal is an instance with an `until` condition: "gather wood until
the chests hold 60".

This is how M6's agent does real work: it picks and configures behaviours and never steers a step at a time (epic TKT-01M62VXZA3, Agent: tools, events and autonomy for the LLM). It is the last step of the epic's worked example (stock low, ask, start a gather instance until the target).

### What exists

- `behaviour.Engine` already has `List`, `Put` (validates against the template schema before saving), `Action` (start, stop, enable, disable, delete) and `Clone`, each taking `by` (`internal/behaviour/engine.go`). Repeat modes include `until` (`internal/behaviour/definition.go`).
- The panel already drives them through `internal/app/behaviours.go`.
- Instances are stored by `SetInstance` (`internal/settings/instances.go`) with who changed them, but there is no pin for an instance, so nothing stops the agent editing one an owner wants left alone.

### What to build

- Tools `behaviour_list` (Query), and `behaviour_start`, `behaviour_stop`, `behaviour_clone`, `behaviour_edit` (Behaviours group), registered in the voice's registry (TKT-01M62VXZAV). They call the engine with `by = agent`.
- Instance pins: an owner or the panel can pin an instance; the agent's edit, delete, enable or disable of a pinned instance is refused with `ErrPinned` and the reason goes back to the model. Starting and stopping a pinned instance follow the autonomy level, not the pin, unless the owner decides otherwise (owner decision recorded on the ticket).
- The autonomy level (TKT-01M62VXZCV) decides what happens when a woken turn calls start, stop, clone or edit: proposed at ask, done and announced at tell, done and logged at quiet. Calls in an owner's own chat turn are that owner's order.
- The agent cannot delete instances, and cannot create one from a template it was not offered.

### Files

- `internal/behaviour/engine.go` (`List`, `Put`, `Action`, `Clone`) and `internal/behaviour/definition.go`
- `internal/settings/instances.go`: instance pins
- `internal/app/behaviours.go`: the panel's pin control and API
- `internal/voice/voice.go`: registry from TKT-01M62VXZAV
- `docs/BEHAVIOURS.md`: agent edits and instance pins

### Verification

- `just check controller test-packages ./internal/behaviour ./internal/settings ./internal/app ./internal/voice`: an agent edit with a bad parameter is refused by the schema; a good edit is saved with source `agent`; a pinned instance refuses agent edits; at ask a woken `behaviour_start` becomes a proposal and at tell it starts and is announced; the agent cannot delete.
- `just check ci`.
- The live worked example belongs to the M6 live evidence ticket.

## Acceptance criteria

- [ ] The agent can list, start, stop, clone and edit instances, within its autonomy level
- [ ] Its changes are checked like any other and recorded as made by the agent
- [ ] Owners and the panel can pin an instance, and the agent's edits to a pinned instance are refused with the reason returned to the model
- [ ] The agent cannot delete instances
