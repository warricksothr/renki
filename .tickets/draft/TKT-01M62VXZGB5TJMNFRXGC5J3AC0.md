---
schema: 4
id: TKT-01M62VXZGB5TJMNFRXGC5J3AC0
title: "Voice: a lasting terva session per bot, and Valheim lore"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - terva
  - llm
assignees: []
milestone: null
parent: TKT-01M62VXZA33E8JR5BR2RJD6AC1
origin: null
dependencies:
  - TKT-01M62VXZAV8CYT5K4DYAQEZC2Q
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:49Z
updated_at: 2026-10-29T03:40:49Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

Each voice turn starts a fresh terva runtime with a summary. Give each bot
a lasting session, so it remembers what it asked or promised an hour ago,
pruned so it stays within budget. Put lasting Valheim knowledge (fuel
types, what each machine makes, which tools break what) into the bot's
tervaHome as lore or skills rather than into every prompt.

## Acceptance criteria

- [ ] Turns for one bot share a session that survives between turns and is kept within a size limit
- [ ] Valheim knowledge lives in the bot's tervaHome and reaches the model without being in the prompt
