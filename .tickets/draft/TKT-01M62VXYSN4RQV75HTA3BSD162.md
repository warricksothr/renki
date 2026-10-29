---
schema: 4
id: TKT-01M62VXYSN4RQV75HTA3BSD162
title: "Controller: behaviour engine: templates, instances, triggers, repeat"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - controller
assignees: []
milestone: M2
parent: TKT-01M62VXYQB6ZDBQE4E6ERGN83C
origin: null
dependencies:
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

Runs behaviour instances. Each instance names a template and fills in:

- **params**, checked against the template's schema;
- **trigger**: manual, an event (inventory full, a drop appeared, stock
  low, idle at home, nightfall) or a timer;
- **when**: conditions that must hold to start or carry on: what the bot
  carries, free slots, what the chests hold (only if seen recently), time
  of day, an owner nearby, where the bot is;
- **repeat**: once, on every trigger with a cooldown, every N minutes, or
  while or until a condition holds, with a maximum number of runs;
- **priority**: one number;
- **enabled**.

A run is one template doing its work: no loops or variables in the
instance. Anything more complex is a new template in Go, which keeps every
instance checkable before it is saved.

### Running

- One body task at a time. A higher-priority instance that becomes ready
  takes over, and the one it interrupted resumes or starts again.
- Survival (flee, eat, the after-death rules) and direct orders from an
  owner always come first. While following, instances stay within a leash
  of the leader.
- Start, stop, enable, disable, clone and edit work on any instance.
  Instances live in the settings store.
- Every trigger, start, step, finish and failure is written to the audit
  log, with the instance's name.

### First templates

`loot` and `stow` (their own tickets). The existing home rules can move to
templates later. They do not have to move now.

## Acceptance criteria

- [ ] Instances are checked against their template's schema before they are saved
- [ ] Triggers, conditions, repeat and priority decide which instance runs, one at a time
- [ ] A higher-priority instance takes over, and an owner's order always does
- [ ] Start, stop, enable, disable, clone and edit work on any instance and survive a restart
- [ ] Each run's start, steps and end are in the audit log with the instance name
