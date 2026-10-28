---
schema: 4
id: TKT-01M61TY9CH2YCQ7KBPQ0FDEVRP
title: "Controller: session state and task dispatch"
type: task
status: done
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
updated_at: 2026-10-28T18:33:32Z
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

- [x] Keeps the latest `state` per connection and drops one with an older `seq`
- [x] Issues task IDs, and refuses to send a task outside the intersection of `capabilities` and `allowedTasks`
- [x] Tracks each task through accepted, running and its final status, and exposes the outcome and reason to callers
- [x] A disconnect marks every in-flight task as stopped, and nothing is re-sent after reconnect

## Implementation plan

internal/session.Session implements bridge.Handler. It keeps the connection, the newest state (dropping a lower seq on the same connection) and a table of tasks by id. Dispatch(ctx, Order) refuses a task outside welcome.allowedTasks ∩ hello.capabilities or one whose args fail DecodeArgs, gives it the id t-N, and sends it. Each Task records its status, reason and detail as the plugin reports them, has Done() and Wait, and carries an opaque Origin so the caller can explain a failure to the right player. Final statuses go to a Listener, which also gets chat and events. On disconnect every unfinished task becomes cancelled/stopped ("disconnected") and is reported, and nothing is re-sent.

## Summary

internal/session.Session is the bridge Handler. It keeps the newest state per connection by seq. Dispatch refuses a task outside welcome.allowedTasks ∩ hello.capabilities (ErrNotAllowed) or with args DecodeArgs rejects (ErrBadArgs), assigns ids t-N, and tracks each Task through the statuses the plugin reports. Done/Wait and an opaque Origin let the caller explain a failure. Final statuses go to the Listener once. A disconnect turns every unfinished task into cancelled/stopped ("disconnected") and re-sends nothing. internal/bridge/bridgetest is a fake plugin for tests.
