---
schema: 4
id: TKT-01M63XJ7W3PWTC56CFQCV25QK0
title: "Panel: speak to the bot as an owner"
type: task
status: in-progress
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
claim:
  actor: agent:claude-code/renki-groom
  branch: t3/m2
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 43a8d60afbf8a6b0cfbd0c86c19e4ab88a5abad7
  session: null
  claimed_at: 2026-10-29T13:28:43Z
  expires_at: null
archive: null
created_at: 2026-10-29T13:28:35Z
updated_at: 2026-10-29T13:28:44Z
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
