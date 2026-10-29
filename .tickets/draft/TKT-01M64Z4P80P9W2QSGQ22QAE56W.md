---
schema: 4
id: TKT-01M64Z4P80P9W2QSGQ22QAE56W
title: "Logout: publish an accurate final connection and phase state"
type: bug
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
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
created_at: 2026-10-29T23:15:23Z
updated_at: 2026-10-29T23:15:23Z
created_by:
  id: agent:codex/renki-resume
  name: ""
updated_by:
  id: agent:codex/renki-resume
  name: ""
extensions: {}
---

## Description

After the logout request at 2026-10-29T23:10:25Z, the control session ended and observation sending stopped, but the panel continued reporting connected/in_world with the last HP/position snapshot. RenkiPlugin.Update only emits state while runner.InSession, which EndSession clears before Game.Logout. VM stop later confirmed both processes stopped. Publish a final accurate phase/disconnection state without restarting control or resuming automation; distinguish stale last observations from a current live character.

## Acceptance criteria

- [ ] After a requested logout the panel no longer presents stale connected/in_world observations as current
- [ ] Logout still revokes all control and automation; no work resumes from the final state
