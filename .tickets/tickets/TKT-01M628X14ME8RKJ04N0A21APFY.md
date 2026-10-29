---
schema: 4
id: TKT-01M628X14ME8RKJ04N0A21APFY
title: "Panel: audit timeline and per-task view in the page"
type: task
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - tooling
assignees: []
milestone: null
parent: TKT-01M628X106RD9FFR8YBXV7K5HT
origin: null
dependencies:
  - TKT-01M628X12EZ1EPF0W7WPJ401XK
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:claude-code/renki-groom
  branch: t3/panel-controls
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: bbd875cbb357ee48a8a63a01d30300213513e88a
  session: null
  claimed_at: 2026-10-29T08:12:21Z
  expires_at: null
archive: null
created_at: 2026-10-28T22:08:14Z
updated_at: 2026-10-29T08:12:22Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

Reading what happened after the fact means `just vm audit` on the dev box today. The panel should read past days itself, and follow one task from the chat that caused it to how it ended, which is what most questions about a run come down to ("why did it go under the floor?").

## Acceptance criteria

- [x] The feed can switch to a past day's audit file and page back through it, without state or perception
- [x] Clicking a task id opens everything about that task in its controller run, with how it ended and how long it took
- [x] A large day stays quick to read: state and perception lines are passed over before they are decoded
- [x] The day asked for cannot name a file outside the audit directory

## Implementation plan

audit.ReadDay streams a file through a Filter, skipping state and perception lines before decoding; audit.TaskID names a record's task. Panel: GET /api/days, /api/timeline?day&before (newest 2000), /api/task?id&at (cut to the run by start records, then Filter{Task}). Page: day picker beside the filters into a separate history list; task id links on every line open a dialog.
