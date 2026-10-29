---
schema: 4
id: TKT-01M63XJ7W3PWTC56CFQCV25QK0
title: "Panel: speak to the bot as an owner"
type: task
status: done
status_reason: null
priority: high
due_on: null
labels:
  - controller
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
created_at: 2026-10-29T13:28:35Z
updated_at: 2026-10-29T13:55:41Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

Owners and agents testing the bot from the panel need to give it the same commands players give in chat, without being in the game. Words typed into the panel reach the brain as the chosen owner's, answered in the panel's feed rather than the game's chat. Commands that need where the speaker stands use their character's position when the bot sees it, and are refused otherwise.

## Acceptance criteria

- [x] Words typed into the panel reach the bot as the chosen owner's, with no need for its name
- [x] The bot answers on the panel, not in the game
- [x] Come here, set home and pick that up use the owner's position when seen, and are refused otherwise

## Summary

POST /api/say and a command box above the feed. Words reach the brain as the chosen owner's, answered in the feed. Position commands and follow need the owner in sight in a fresh perception. Checked on the VM 2026-10-29: status from the panel answered in the feed only.
