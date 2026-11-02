---
schema: 4
id: TKT-01M69V7YHKDRKNHVAEJ810TEJ7
title: Accept addressed Polo replies without conversational fallback
type: bug
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - chat
  - controller
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
created_at: 2026-10-31T20:43:28Z
updated_at: 2026-10-31T20:43:28Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

Shouted `tas polo` currently falls through to the conversational model because presence.Chat recognizes only bare `polo` and addressed `<alias> marco`. On2026-10-31, Castile Rackham joined at20:38:07Z and correctly triggered Marco two seconds later while out of sight. Both jarl's and Nimah's subsequent shouted `tas polo` received conversational "I'm right here" replies despite fresh heard positions about1km away. Audit: .checks/audit-window-pFAxI6.jsonl. The join trigger itself is working as configured.

Accept addressed Polo using the configured nickname/character-name normalization, preserving the two-minute Marco response window, throttling, world/session scope and public-response permission boundary. It should produce the same measured caller-to-bot direction and distance as bare Polo, consume recognized location requests before the model, and avoid inventing proximity. Add focused presence checks for punctuation/case, stale/out-of-window requests, unrelated addressed speech and permission changes. This is follow-up chat work; M2/M3 escape reliability remains higher priority.
