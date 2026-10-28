---
schema: 4
id: TKT-01M628X12EZ1EPF0W7WPJ401XK
title: "Panel: server on the tailnet, token login, live view"
type: task
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - tooling
assignees: []
milestone: null
parent: TKT-01M628X106RD9FFR8YBXV7K5HT
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:claude-code/renki-groom
  branch: t3/groom-start-renki
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 71a7b5d37ee0c284de90e7106cc8f61d34cba82f
  session: null
  claimed_at: 2026-10-28T22:34:26Z
  expires_at: null
archive: null
created_at: 2026-10-28T22:08:14Z
updated_at: 2026-10-28T22:34:26Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

The panel's server and live view. The controller serves the page on its own
listen address (the VM's Tailscale address), separate from the loopback
bridge, with a login token. Static files are embedded in the binary; live
updates come over server-sent events: state, the running task, chat, and
the audit records as they are written. No build step for the page.

## Acceptance criteria

- [ ] The controller serves the panel on panel.listen (the VM's Tailscale address), and a failure to listen there does not stop the bot
- [ ] Every panel request needs the token: a login form sets an HttpOnly cookie, and nothing else is served without it
- [ ] The page shows connection, character, world, phase, task, health, position and home distance, live
- [ ] A top-down map draws the bot's trail, nearby players and home
- [ ] Chat, task statuses, the plugin's traces and the controller's decisions stream into the page as they happen
- [ ] The VM firewall admits the panel port from the tailnet only, and the panel opens from a tailnet device
