---
schema: 4
id: TKT-01M62N12KFFJ7M0345AXB0MEVH
title: "Controller: no-chop zones the bot leaves alone"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - chat
assignees: []
milestone: null
parent: null
origin: null
dependencies:
  - TKT-01M62N12JW8TFN12MAT8M1ADZ8
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T01:40:10Z
updated_at: 2026-10-29T01:40:23Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

Later functionality, after M2's benchmark (Nimah, 2026-10-28): places the owner wants left alone (trees around the base, a garden, a scenic grove). Marked by command where the owner stands ("don't chop here", with a radius), kept per world, shown on the panel, and respected when choosing trees.

## Acceptance criteria

- [ ] An owner can mark and clear a no-chop zone where they stand
- [ ] Choosing trees to chop skips anything inside a zone
- [ ] The panel draws the zones
