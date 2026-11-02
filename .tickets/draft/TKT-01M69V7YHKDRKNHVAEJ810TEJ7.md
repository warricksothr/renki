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
milestone: M6
parent: TKT-01M62VXZA33E8JR5BR2RJD6AC1
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-31T20:43:28Z
updated_at: 2026-11-02T15:53:57Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m6
  name: ""
extensions: {}
---

## Description

Shouted `tas polo` currently falls through to the conversational model because presence.Chat recognizes only bare `polo` and addressed `<alias> marco`. On 2026-10-31, Castile Rackham joined at 20:38:07Z and correctly triggered Marco two seconds later while out of sight. Both jarl's and Nimah's subsequent shouted `tas polo` received conversational "I'm right here" replies despite fresh heard positions about 1 km away. Audit: .checks/audit-window-pFAxI6.jsonl. The join trigger itself is working as configured.

Accept addressed Polo using the configured nickname/character-name normalization, preserving the two-minute Marco response window, throttling, world/session scope and public-response permission boundary. It should produce the same measured caller-to-bot direction and distance as bare Polo, consume recognized location requests before the model, and avoid inventing proximity. Add focused presence checks for punctuation/case, stale/out-of-window requests, unrelated addressed speech and permission changes. This is follow-up chat work; M2/M3 escape reliability remains higher priority.

### Why it is in M6

The fallback is a voice honesty failure: the model answered "I'm right here" about a player 1 km away. M6 is where the voice is made truthful (see TKT-01M630V829, Voice: never promise an action no tool can carry out), and every line the grammar or presence handles exactly is one the model cannot get wrong.

### Where it happens

`Manager.Chat` in `internal/presence/presence.go` (around lines 126-146): `polo` must match the whole normalized text, while the alias loop accepts only `<alias> marco`. Addressed `<alias> polo` matches neither, returns false, and the brain hands the line to the voice. The answer path (`kind == "answer"`) already measures direction and distance and respects `marcoUntil`.

### Files

- `internal/presence/presence.go`: `Chat`, `normalized`, the Marco window and reply throttling
- `internal/presence/presence_test.go`
- `docs/PLAYERS.md`: the Marco/Polo section

### Verification

- `just check controller test-packages ./internal/presence ./internal/chat`.
- `just check ci`.
- Offline only: the presence tests reproduce the 2026-10-31 case from the audit window above.

## Acceptance criteria

- [ ] Shouted <alias> polo, for any configured nickname or the character name and with any case or trailing punctuation, is handled as bare polo inside the Marco window and never reaches the voice
- [ ] Addressed polo outside the two-minute window, from another world or session, or after the public-response permission is withdrawn gets no invented direction or distance
- [ ] Unrelated addressed speech such as <alias> polo stick still goes to the grammar and voice
- [ ] docs/PLAYERS.md describes addressed polo
