---
schema: 4
id: TKT-01M62N12G98VW2CYW94FG5RCTB
title: "Controller: notice where owners work, and the drops piling up"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - controller
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies:
  - TKT-01M62EP0PDAB02DNJAJ6WS3MYD
  - TKT-01M62BMD18K3SE1WXJ7FDN8EPT
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

From perception over time (the attention design on TKT-01M62EP0P): when trees vanish and logs, stumps and wood drops appear near an owner, that spot is a work site: remember it per world with what was gathered there and when. Drops are grouped per cell and prefab with their trend, so "wood piling up near Nimah" is a signal the hauling routine can act on.

## Acceptance criteria

- [ ] Felling, splitting and drops near an owner mark a work site, kept per world with what and when
- [ ] Drops near the bot are summed per cell and prefab with their trend over the last minutes
- [ ] The panel shows work sites, and the voice summary can mention them
