---
schema: 4
id: TKT-01M628X134PX2DG8D7G4VJQF40
title: "Panel: stop, go home and log out from the page"
type: task
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - controller
assignees: []
milestone: null
parent: TKT-01M628X106RD9FFR8YBXV7K5HT
origin: null
dependencies:
  - TKT-01M628X12EZ1EPF0W7WPJ401XK
  - TKT-01M628X112PXEDXM3ZGDBJWXPQ
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:claude-code/renki-groom
  branch: t3/panel-controls
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 3e59b4f460f84e00abe9d7940cff4fbe528ac6d8
  session: null
  claimed_at: 2026-10-29T07:27:50Z
  expires_at: null
archive: null
created_at: 2026-10-28T22:08:14Z
updated_at: 2026-10-29T07:30:39Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

Stop, go home and log out from the panel, without chat or the VM. The controller already does all three (chat "stop" and "go home", and /control/logout on loopback); the page needs buttons and an endpoint behind its login that a forged cross-site form cannot press.

## Acceptance criteria

- [x] Stop, Go home and Log the bot out buttons in the header, enabled while the plugin is connected
- [x] Each press shows the result beside the buttons, with the reason when it could not be done
- [x] Logging the bot out asks for confirmation first
- [x] Each press is in the audit log with where it came from
- [x] A request without the panel's header is refused, so a forged cross-site form cannot press them

## Implementation plan

POST /api/act/{stop,home,logout} behind the login, needing an X-Renki-Act header; panel.Options.Act; App.PanelAct dispatches stop, calls Home.GoHome and Session.Logout, and words the errors. Buttons in the header with a result line; confirm() before logout.
