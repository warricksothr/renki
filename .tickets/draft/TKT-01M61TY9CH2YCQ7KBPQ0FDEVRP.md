---
schema: 4
id: TKT-01M61TY9CH2YCQ7KBPQ0FDEVRP
title: "Controller: session state and task dispatch"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - protocol
assignees: []
milestone: M1
parent: TKT-01M61T36B839WT7KEZ14TDMB8B
origin: null
dependencies:
  - TKT-01M61T36K1WHN4SJATSZWZEA8W
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-28T18:04:16Z
updated_at: 2026-10-28T18:04:16Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

The controller needs one place that knows what the plugin is doing:
the latest `state`, the task it sent last, and what the plugin said about it.
The chat grammar and the terva replies both read from it, and both send tasks
through it. docs/BRIDGE-PROTOCOL.md 6.1, 6.6 and 7.1.

The plugin holds at most one body task, so the queue here can stay trivial
for M1: one body task in flight, instant tasks fire and forget their status
into the log.

## Acceptance criteria

- [ ] Keeps the latest `state` per connection and drops one with an older `seq`
- [ ] Issues task IDs, and refuses to send a task outside the intersection of `capabilities` and `allowedTasks`
- [ ] Tracks each task through accepted, running and its final status, and exposes the outcome and reason to callers
- [ ] A disconnect marks every in-flight task as stopped, and nothing is re-sent after reconnect
