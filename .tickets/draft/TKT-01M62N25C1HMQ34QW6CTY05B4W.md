---
schema: 4
id: TKT-01M62N25C1HMQ34QW6CTY05B4W
title: "Controller: danger zones from where the bot was hurt"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - movement
assignees: []
milestone: M3
parent: TKT-01M62N2557R401S3E6JECSAPN4
origin: null
dependencies:
  - TKT-01M62EP0PDAB02DNJAJ6WS3MYD
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T01:40:46Z
updated_at: 2026-10-29T01:40:46Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

From memory's deaths and damage (TKT-01M62EP0P): places that hurt the bot become danger zones that decay over days. Paths and the choice of work site avoid them when there is another way, and the panel shows them.

## Acceptance criteria

- [ ] Damage and deaths make danger zones that decay
- [ ] Choosing a site or a route prefers to avoid them
- [ ] The panel draws them
