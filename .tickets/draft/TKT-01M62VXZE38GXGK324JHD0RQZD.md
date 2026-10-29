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
milestone: null
parent: TKT-01M62VXZA33E8JR5BR2RJD6AC1
origin: null
dependencies:
  - TKT-01M62VXZAV8CYT5K4DYAQEZC2Q
  - TKT-01M62VXYSN4RQV75HTA3BSD162
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

The Behaviours group of agent tools, over the behaviour engine and the
settings store. Edits are checked against the template's schema like any
other, are recorded with source `agent`, and obey pins and the autonomy
level. A goal is an instance with an `until` condition: "gather wood until
the chests hold 60".

## Acceptance criteria

- [ ] The agent can list, start, stop, clone and edit instances, within its autonomy level
- [ ] Its changes are checked like any other and recorded as made by the agent
