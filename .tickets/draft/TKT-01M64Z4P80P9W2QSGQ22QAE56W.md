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
  - live
assignees: []
milestone: M4
parent: TKT-01M628X106RD9FFR8YBXV7K5HT
origin: null
dependencies:
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T23:15:23Z
updated_at: 2026-11-02T15:57:06Z
created_by:
  id: agent:codex/renki-resume
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m4
  name: ""
extensions: {}
---

## Description

After the logout request at 2026-10-29T23:10:25Z, the control session ended and observation sending stopped, but the panel continued reporting connected/in_world with the last HP/position snapshot. RenkiPlugin.Update only emits state while runner.InSession, which EndSession clears before Game.Logout. VM stop later confirmed both processes stopped. Publish a final accurate phase/disconnection state without restarting control or resuming automation; distinguish stale last observations from a current live character.

### Goal

Goal 4.6, through the panel epic TKT-01M628X106 (Web control panel to watch and steer the bot): its second criterion is that the panel shows the bot's state live. Today, after a logout, it shows stale state as current. It also matters to the soak (goal 4.2), which reads panel status once a minute.

At 0b932ef `RenkiPlugin.Update` sends state while `runner.InSession || loggingOut` (`plugin/Renki.Plugin/RenkiPlugin.cs:296`), which may already cover part of this. Check what the panel shows after a logout before changing anything.

### Files

- `plugin/Renki.Plugin/RenkiPlugin.cs` (`EndSession` at :398, `Update`, the `loggingOut` flag)
- `internal/session/session.go` (`Logout` at :480), `internal/app/panel.go` (`PanelStatus`)
- `panel/src/status.js`, `panel/src/controller.jsx` (how the header presents connection and phase)

### Verification

- `just check controller test-packages ./internal/session ./internal/app` with a test that a logout marks the last observation stale.
- `just check panel test` and `just check panel ui-test` with a fixture for the post-logout state.
- Live, under the VM lease (TKT-01M6EF5DTG): `just vm deploy`, press Log out in the panel, and confirm the header no longer shows connected/in_world with the last HP; record it in a note. No work resumes afterwards.

## Acceptance criteria

- [ ] After a requested logout the panel no longer presents stale connected/in_world observations as current
- [ ] Logout still revokes all control and automation; no work resumes from the final state
