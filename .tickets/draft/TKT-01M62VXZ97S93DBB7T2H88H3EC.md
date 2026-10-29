---
schema: 4
id: TKT-01M62VXZ97S93DBB7T2H88H3EC
title: "Controller: suggest chest assignments from contents and signs"
type: task
status: draft
status_reason: null
priority: low
due_on: null
labels:
  - controller
assignees: []
milestone: null
parent: TKT-01M62VXZ2FWDX3D4Z6XX1MB5A6
origin: null
dependencies:
  - TKT-01M62VXZ84SA9TSJC6NJNPV8KJ
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

Later: the bot suggests what each chest is for from what it already holds
(a chest that is mostly wood is the wood chest), and from a sign placed next
to it. It offers the suggestions to an owner, who accepts them into the
assignment table. It never assigns on its own.

## Acceptance criteria

- [ ] The bot suggests assignments from a chest's contents and a sign beside it
- [ ] Suggestions take effect only when an owner accepts them
