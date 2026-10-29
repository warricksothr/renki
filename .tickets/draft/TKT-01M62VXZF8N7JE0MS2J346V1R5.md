---
schema: 4
id: TKT-01M62VXZF8N7JE0MS2J346V1R5
title: "Controller: stock targets and a stock-low event from chest memory"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - chat
assignees: []
milestone: null
parent: TKT-01M62VXZA33E8JR5BR2RJD6AC1
origin: null
dependencies:
  - TKT-01M62VXYWJHB5BC9VGCBM0PVXD
  - TKT-01M62VXYSN4RQV75HTA3BSD162
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:49Z
updated_at: 2026-10-29T03:40:49Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

Owners set targets, `<name> keep 50 wood`, per bot. From chest memory, the
controller sums the base's stock of each targeted item and fires
`stock_low` when it falls below the target. It only uses counts seen
recently enough, and says how old they are. The event can trigger a
behaviour instance directly (restock with gather) or wake the agent.

## Acceptance criteria

- [ ] Owners set and clear stock targets from chat
- [ ] stock_low fires when remembered stock, seen recently enough, falls below a target
- [ ] An instance or the agent can act on stock_low
