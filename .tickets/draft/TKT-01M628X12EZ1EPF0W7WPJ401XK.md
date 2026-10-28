---
schema: 4
id: TKT-01M628X12EZ1EPF0W7WPJ401XK
title: "Panel: server on the tailnet, token login, live view"
type: task
status: draft
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
claim: null
archive: null
created_at: 2026-10-28T22:08:14Z
updated_at: 2026-10-28T22:08:14Z
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
