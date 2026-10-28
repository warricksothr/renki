---
schema: 4
id: TKT-01M627TCXAZM8ZF728DR11AVTD
title: Audit log of everything the bot saw, decided and did
type: task
status: review
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - plugin
  - protocol
  - tooling
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:claude-code/renki-groom
  branch: t3/groom-start-renki
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: a4058a0f5baf9515154c5e201ea3e6a93c36844f
  session: null
  claimed_at: 2026-10-28T21:49:20Z
  expires_at: null
archive: null
created_at: 2026-10-28T21:49:20Z
updated_at: 2026-10-28T22:01:14Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

When the bot misbehaves in the game, we need to reconstruct what happened
afterwards: what was said, what the controller decided, what the plugin was
told, and what the plugin decided inside a task (gait, paths, stuck
recovery). Today that is spread over a text controller log, the BepInEx log
and nothing at all for the plugin's internal choices.

### Shape

- The controller writes an append-only JSON Lines audit log, one file per
  day, pruned after `keepDays`. It records every bridge frame both ways
  (hello token redacted; ping and pong left out), connects and disconnects,
  and every controller log record at debug level and up.
- A new plugin-to-controller `trace` message carries the plugin's own
  decisions, tagged with the current task id, so they land in the same
  ordered file.
- `renki-audit` reads the files and prints a timeline, filtered by task or
  time window, with state optional.
- `just vm audit` fetches the VM's files and runs it.

## Acceptance criteria

- [x] The controller writes every bridge frame both ways, connects, disconnects and its own log records to a daily JSON Lines file, with the hello token redacted
- [ ] The plugin sends trace messages for follow gait changes, new paths and stuck recoveries, tagged with the current task
- [x] renki-audit prints a timeline of the files, filtered by task and time window
- [x] just vm audit fetches the VM's audit files and runs renki-audit on them
- [x] The protocol doc, fixtures and docs describe trace and the audit log

## Implementation plan

- internal/audit: daily JSON Lines writer (start, frame, log records),
  pruning after keepDays, a slog handler, hello token redaction, and the
  timeline reader with task and time filters.
- bridge.Options.Tap sees every frame both ways; Serve now waits for its
  connections at shutdown so their last records are written.
- The controller tees its logger into the audit log at debug level, and
  records its configuration with secrets redacted at start.
- Protocol: a plugin-to-controller `trace` message (6.7), with fixtures on
  both sides. The plugin's Tracer sends follow gait changes, paths and stuck
  recoveries, stamped with the running task.
- cmd/audit (renki-audit), `just controller audit`, `just vm audit`.
- docs/AUDIT.md, plus links from TESTING.md, VM.md and the protocol doc.

## Notes

**agent:claude-code/renki-groom** at 2026-10-28T22:01:14Z

Deployed as 5e97c84. just vm audit fetched the VM's file and showed the restart, handshake and state. Criterion 2 is implemented and covered by the trace fixture on both sides, but no in-game follow has produced a trace yet; tick it once one shows up in just vm audit.
