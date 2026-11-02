---
schema: 4
id: TKT-01M6EFQPF8Z9RM4PBAFWVP8F3N
title: Start, stop and ask about expeditions from panel and chat
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - chat
assignees: []
milestone: M7
parent: TKT-01M66PMAV9P2Z8QMR9N90T0TH1
origin: null
dependencies:
  - TKT-01M6EFQPDBH4N2D0A074WWTJYY
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:58:33Z
updated_at: 2026-11-02T15:58:44Z
created_by:
  id: agent:claude-code/renki-plan-m7
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m7
  name: ""
extensions: {}
---

## Description

Rollout step 5, control. Start, stop and resume an expedition from the panel and from an addressed chat request ("Tas, explore around the meadow camp and return"), using existing role, global and native permissions. Locate-only access does not grant travel. Read-only questions ("what did you discover?", "how much remains unknown?") answer from the goal summary and coverage query. Only implemented capabilities are offered.

### Files
- internal/app/goals.go, internal/app/control.go
- internal/chat/ (addressed commands), internal/access/ (permissions)
- panel/src/controller.jsx

### Verification
- `just check controller test-packages ./internal/app ./internal/chat ./internal/access`
- `just check panel ci`
- `just check ci`

## Acceptance criteria

- [ ] An owner can start, stop and resume an expedition from the panel and from chat; a locate-only player's start request is refused with a reason (tests)
- [ ] A start names a waypoint and its pinned version; an edited waypoint requires an explicit new start (test)
- [ ] The two read-only questions answer from the summary in one chat line each without starting any task (test)
- [ ] Stop from chat ends the goal and its continuation (test)
