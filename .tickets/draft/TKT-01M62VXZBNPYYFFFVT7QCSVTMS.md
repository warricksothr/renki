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
milestone: null
parent: TKT-01M62VXZA33E8JR5BR2RJD6AC1
origin: null
dependencies:
  - TKT-01M62VXZAV8CYT5K4DYAQEZC2Q
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:48Z
updated_at: 2026-10-29T03:40:48Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
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

## Acceptance criteria

- [ ] Listed events wake the agent with the tool groups that event allows
- [ ] Events close together merge into one turn, and turns respect a minimum gap, quiet hours and the token cap
- [ ] A switch turns waking off, leaving chat as before
