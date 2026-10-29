---
schema: 4
id: TKT-01M630V80TFBMWT1B3XY4KP2QJ
title: "Spike: compare voice models on fixed chat scenarios"
type: spike
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - llm
  - terva
  - tooling
assignees: []
milestone: null
parent: null
origin: null
dependencies:
  - TKT-01M630V7ZTEW2WR52HVWXR475M
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T05:06:42Z
updated_at: 2026-10-29T05:06:42Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

On 2026-10-28 Nimah told the overloaded bot "Can you drop something and I
will pick it up". The local gemma took 26 s, answered "Aye, I am heavy as
a boulder. I will drop the wood now. I am quite heavy with this wood. I
will drop it now so we can keep moving." (repeating itself), and proposed
`stop`: it promised what no tool it had could do.

A replay harness with scenarios taken from the audit log (an order the
grammar misses, a request it has no tool for, small talk, a non-owner
asking for something) scores each model on latency, whether it chose the
right tool or none, and whether it promised only what it could do. That
decides the default model and whether a local model is good enough.

## Acceptance criteria

- [ ] A just recipe replays a fixed set of chat scenarios against a chosen model and scores the replies
- [ ] haiku-5-5 and the local gemma are compared on latency, tool choice and honesty, and the result is recorded here
