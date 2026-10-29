---
schema: 4
id: TKT-01M62VXYR69PKZGJFEWQ1759GV
title: "Controller: one store for settings and instances, with who changed them"
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - controller
assignees: []
milestone: M2
parent: TKT-01M62VXYQB6ZDBQE4E6ERGN83C
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:claude-code/renki-groom
  branch: t3/settings
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 310e173cf9da2511451f7c043d5faca63dd952dd
  session: null
  claimed_at: 2026-10-29T05:26:46Z
  expires_at: null
archive: null
created_at: 2026-10-29T03:40:48Z
updated_at: 2026-10-29T05:38:29Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

Settings now come from the config file at start. The behaviour engine, the
panel, chat and the agent all need to change them while the bot runs, and
everyone needs to see who changed what.

One store in the controller's `dataDir` holds the runtime settings (the home
rules, the allowed tasks, the voice switch, owners, nicknames) and the
behaviour instances. Every write goes through it, records its source
(`owner:<name>`, `panel`, `agent`, `config`) and the time, and lands in the
audit log. The config file seeds it; what was changed at runtime wins until
it is reset.

An owner can **pin** a setting, so the agent cannot change it (the autonomy
ticket enforces that).

## Acceptance criteria

- [ ] Runtime settings and instances are kept in dataDir and survive a restart
- [x] Every change records its source and time and is written to the audit log
- [x] A pinned setting refuses changes from the agent
- [x] Values from the config file apply until changed at runtime, and a reset returns to them

## Implementation plan

- New package `internal/settings`. The runtime values are owners,
  nicknames, allowedTasks, the voice on or off, the home rules and
  overload.drop, each under a dotted key (`home.idleMinutes`). The config
  file is the base. Overrides by key live in `dataDir/settings.json`, each
  with who set it and when.
- `Set(key, value, by)` checks the key and the whole result, using the same
  rules as config.Validate, before saving. `Reset(key, by)` returns a key to
  the config's value. `Pin` and `Unpin` are for owners and the panel only;
  a write from `agent` to a pinned key fails with ErrPinned. Every change
  goes to the log, and so to the audit log, as "setting changed" with the
  key, the old and new values, and who made it.
- Readers take a fresh snapshot each time rather than copying at start, so
  changes apply at once:
  - the brain: owners, nicknames, voice on or off;
  - the home keeper: its rules;
  - the unloader: overload.drop;
  - the session: allowedTasks.
- allowedTasks can narrow at once, because the session refuses what has
  been dropped from the list. Widening needs the plugin's agreement, which
  comes in welcome, so a task added to the list goes live at the next
  connection. The panel ticket can add the protocol message to do it live.
- Behaviour instances are added to the same store by the engine ticket.
  This ticket gives the store the hook (a key namespace) but no instances.
- No UI here: chat and panel controls are TKT-01M628X13X.

## Notes

**agent:claude-code/renki-groom** at 2026-10-29T05:38:29Z

2026-10-29: the store is in internal/settings, backed by dataDir/settings.json. Settings survive a restart, but criterion 1 also names behaviour instances, which wait for the engine (TKT-01M62VXYSN4), so it stays unticked. The agent can never change owners or allowedTasks, pinned or not. Nothing changes settings yet except code and tests; chat and panel controls are TKT-01M628X13X.
