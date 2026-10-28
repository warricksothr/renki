---
schema: 4
id: TKT-01M628X106RD9FFR8YBXV7K5HT
title: Web control panel to watch and steer the bot
type: epic
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - tooling
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

A web page, served by the controller, for watching the bot live and steering
it: where it is, what it is doing and why, what it heard and said, and
switches for its behaviours. It is reached over Tailscale at the VM's
tailnet address and asks for a login token, since anything on the tailnet
can reach it. The audit log (TKT-01M627TCX) is its source for history.

## Acceptance criteria

- [ ] The panel opens from a tailnet device, behind a token
- [ ] It shows the bot's position, task, chat and recent decisions live
- [ ] It can stop the bot, send it home and log it out
- [ ] Behaviour switches change what the bot does without a restart
