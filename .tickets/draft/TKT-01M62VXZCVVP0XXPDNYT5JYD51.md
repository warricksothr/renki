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
milestone: null
parent: TKT-01M62VXZA33E8JR5BR2RJD6AC1
origin: null
dependencies:
  - TKT-01M62VXZBNPYYFFFVT7QCSVTMS
  - TKT-01M62VXYR69PKZGJFEWQ1759GV
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

An autonomy level per bot governs what the agent may do when nobody asked:

- **ask** (the default): it proposes, and waits for an owner's yes
- **tell**: it acts and says what it did
- **quiet**: it acts and only writes to the audit log

`ask_owner(question, options)` puts a question to the owners in chat. The
answer comes back as an event, and a question nobody answers lapses.
`remind_me(after, about)` schedules an event. An owner's pin (settings
store) refuses any change by the agent at every level.

## Acceptance criteria

- [ ] At ask, self-started actions wait for an owner's answer, and at tell they happen and are announced
- [ ] An owner's answer to ask_owner wakes the agent with it, and unanswered questions lapse
- [ ] Pinned settings refuse the agent at every level
