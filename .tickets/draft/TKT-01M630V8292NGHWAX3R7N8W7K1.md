---
schema: 4
id: TKT-01M630V8292NGHWAX3R7N8W7K1
title: "Voice: never promise an action no tool can carry out"
type: bug
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - llm
  - chat
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

Seen 2026-10-28 (23:09): asked to drop something, the voice said it
would drop the wood and proposed `stop`, since drop was not in its tools.
Players believe what the bot says. The prompt should say so, and the
brain can catch the worst cases: a reply that promises an action while
the proposal is empty or a different task.

## Acceptance criteria

- [ ] The prompt tells the model to agree to an action only when it calls the tool for it
- [ ] A reply that says it will do something while proposing nothing, or something else, is caught and replaced
- [ ] Requests it has no tool for get an honest answer, such as that it cannot do that yet
