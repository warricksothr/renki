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
milestone: null
parent: TKT-01M62VXZA33E8JR5BR2RJD6AC1
origin: null
dependencies: []
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

Move the voice's task catalogue (`internal/voice`) into a registry where
each tool has a group, a level, a schema and a handler.

- **Query tools** answer at once from the session and memory: status,
  inventory, nearby (perception), recall, behaviours, recent events.
  Answers are short summaries, not raw streams.
- **Action tools** are checked when they are called, by the brain's own
  checks (owner, allowedTasks, args). The result goes back to the model:
  accepted, or refused with the reason.
- Several actions per turn, up to a cap (3). More steps per turn (about 6),
  so the model can look and then act.
- Which groups are offered depends on who is speaking: a player who is not
  an owner gets Express and Query.

Every tool call, with its result, goes to the audit log.

## Acceptance criteria

- [ ] Tools are registered with a group, a level and a schema, and the voice offers them by group
- [ ] Query tools return real data from the session and memory
- [ ] An action's acceptance or refusal, with the reason, goes back to the model
- [ ] Up to three actions per turn, and every call is in the audit log
