---
schema: 4
id: TKT-01M62VXYQB6ZDBQE4E6ERGN83C
title: "Behaviours: templates and instances the bot runs on its own"
type: epic
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - chat
assignees: []
milestone: M2
parent: null
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

The bot's routines are built from a few templates rather than coded one by
one. A **template** is a built-in kind of work, written in Go, that says what
parameters it takes and how it runs: `loot`, `stow`, `fuel`, `gather`. An
**instance** is a template with its parameters filled in, plus a name, a
trigger, conditions, how it repeats, a priority, and whether it is on.
Instances are kept per bot.

Owners clone an instance and change its parameters ("stow wood" becomes
"stow stone" by changing one field), and start, stop, switch on and switch
off any of them from chat, the panel or, later, the agent.

Organising the base comes down to one template: `stow` moves all of some
items to an ordered list of chests, and says when they do not fit, so
players know more chests are needed or something must be cleared out.
Grouping items into chests on its own comes later (the base upkeep epic).

### Decisions so far (2026-10-28)

- Behaviours and their switches are per bot, not per world.
- Loot pickup is on by default, and leaves an owner's deliberate drops alone.
- Stowing uses chests inside the base unless an instance names others.
- Survival and direct orders always come before every instance.

## Acceptance criteria

- [ ] Loot pickup and deliver-when-full run as instances of templates, not as hand-written routines
- [ ] An owner can clone an instance, change a parameter and start it from chat or the panel
- [ ] When items do not fit, the owners are told what and how much, and the panel shows it
