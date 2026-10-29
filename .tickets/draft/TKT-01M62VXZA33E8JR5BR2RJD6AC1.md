---
schema: 4
id: TKT-01M62VXZA33E8JR5BR2RJD6AC1
title: "Agent: tools, events and autonomy for the LLM"
type: epic
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - llm
  - terva
  - controller
assignees: []
milestone: null
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

Today the LLM answers chat lines the grammar does not know and may propose
one of five tasks. It never sees whether that was accepted, and it never
acts unless spoken to (`internal/voice`).

This epic makes it the glue between the bot's building blocks. Code decides
**when** it wakes (events), the agent decides **what** to do, and code
carries it out. The agent picks and configures behaviours and asks owners.
It never steers a step at a time, and it never handles danger: reflexes
stay in code.

### Tool groups

| Group | Level | Tools |
|---|---|---|
| Express | free | reply text, emote, look_at |
| Query | free, read only | status, inventory, nearby, recall (places, work sites, chests and their age), behaviours, recent events |
| Act | allowed tasks | follow, go to a place, pickup, go home, and later deposit and harvest |
| Behaviours | owner or autonomy | list, start, stop, clone, edit instances |
| Memory | owner | name a place, mark a chest, a no-chop zone, a note |
| Social | free | tell_owner, ask_owner, remind_me |

### The worked example

1. A deposit sees 12 wood in the wood chest, under its target of 50, so a
   `stock_low` event fires.
2. The agent recalls the beech stand the owner chopped at, sees room in its
   pack and that chopping is allowed there.
3. It asks "Wood's low (12). Chop at the beech stand?", or, at autonomy
   **tell**, starts a gather instance until 60 and says so.
4. The behaviours do the work. The ending comes back as an event.

### Decisions so far (2026-10-28)

The default autonomy is **ask**. An owner's pin always wins. The agent can
never touch logout, bosses, giving items away, or Safety.

## Acceptance criteria

- [ ] Query tools let the agent look before it acts, and actions report whether they were accepted
- [ ] Events from the world wake the agent within a budget, with only the tools that event allows
- [ ] At autonomy ask, the agent proposes and waits for an owner, and at tell it acts and says so
