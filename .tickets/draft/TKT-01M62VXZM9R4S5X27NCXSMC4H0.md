---
schema: 4
id: TKT-01M62VXZM9R4S5X27NCXSMC4H0
title: "Perception: scans in dense bases can miss objects past 2048 colliders"
type: bug
status: draft
status_reason: null
priority: low
due_on: null
labels:
  - plugin
assignees: []
milestone: null
parent: null
origin: null
dependencies:
  - TKT-01M62BMD18K3SE1WXJ7FDN8EPT
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

The perception scan reads colliders into a buffer of 2048. In a big base,
more colliders than that inside the radius means some objects are missed,
and the scan says it was truncated. Accepted in PR 3. Fix if the trace shows
truncation at home: a larger or growing buffer, or a smaller radius for
structures.

## Acceptance criteria

- [ ] A scan with more colliders than the buffer still reports everything in range, or grows the buffer
