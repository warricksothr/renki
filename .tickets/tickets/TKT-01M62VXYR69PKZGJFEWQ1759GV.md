---
schema: 4
id: TKT-01M62VXYR69PKZGJFEWQ1759GV
title: "Controller: one store for settings and instances, with who changed them"
type: task
status: ready
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
claim: null
archive: null
created_at: 2026-10-29T03:40:48Z
updated_at: 2026-10-29T05:26:35Z
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
- [ ] Every change records its source and time and is written to the audit log
- [ ] A pinned setting refuses changes from the agent
- [ ] Values from the config file apply until changed at runtime, and a reset returns to them
