---
schema: 4
id: TKT-01M62VXZA33E8JR5BR2RJD6AC1
title: "Agent: tools, events and autonomy for the LLM"
type: epic
status: draft
status_reason: null
priority: normal
due_on: "2026-12-20"
labels:
  - llm
  - terva
  - controller
assignees: []
milestone: M6
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:48Z
updated_at: 2026-11-02T15:56:16Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-plan-m6
  name: ""
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

## Definition of done

- [ ] 6.1 The default voice model is chosen from voice-eval runs (3 runs of at least 20 scenarios against haiku-5-5 and the local gemma), and it has tool accuracy at least 90 %, no false promises and p95 reply latency at most 10 s
- [ ] 6.2 In the look-first scenarios a query precedes the first action in at least 80 % of runs, and every action call returns accepted or refused and has an agent tool audit record
- [ ] 6.3 No reply promises an action without a matching accepted call, across the scenario set and the live owner sessions
- [ ] 6.4 Over a 4 h wake soak, woken turns average at most 4,000 tokens per hour, no clock hour exceeds 8,000, at most 6 woken turns per hour, and no danger event wakes the agent
- [ ] 6.5 Over the same soak, event to turn start p95 is at most 7 s and event to first chat line p95 at most 20 s
- [ ] 6.6 At ask no self-started action is dispatched without an owner's yes, at tell every one is announced, and pins refuse the agent at every level
- [ ] 6.7 Over at least 2 owner sessions totalling at least 2 h with at least 10 answered proposals, owners reject at most 30 %
- [ ] 6.8 The stock-low worked example completes once at ask and once at tell
- [ ] 6.9 Every M6 child is done, or has a note saying what is left and is moved to a later milestone
