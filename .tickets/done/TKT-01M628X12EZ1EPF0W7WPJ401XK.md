---
schema: 4
id: TKT-01M628X12EZ1EPF0W7WPJ401XK
title: "Panel: server on the tailnet, token login, live view"
type: task
status: done
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
dependencies:
  - TKT-01M62BMD18K3SE1WXJ7FDN8EPT
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-28T22:08:14Z
updated_at: 2026-10-29T05:42:44Z
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

- [x] The controller serves the panel on panel.listen (the VM's Tailscale address), and a failure to listen there does not stop the bot
- [x] Every panel request needs the token: a login form sets an HttpOnly cookie, and nothing else is served without it
- [x] The page shows connection, character, world, phase, task, health, position and home distance, live
- [x] A top-down map draws the bot's trail, nearby players and home
- [x] Chat, task statuses, the plugin's traces and the controller's decisions stream into the page as they happen
- [x] The VM firewall admits the panel port from the tailnet only, and the panel opens from a tailnet device

## Implementation plan

- internal/panel: login form and HttpOnly cookie (or bearer), embedded
  page (index.html, app.js, style.css; no build step), GET /api/status,
  GET /api/events (server-sent events: recent records, then live, plus a
  status each second). Security headers and a strict CSP.
- internal/audit: recent records, a state-a-second trail and subscribers,
  so the audit log is the panel's feed; a log with no dir still feeds.
- internal/app: PanelStatus snapshot; ServePanel on panel.listen, retrying
  every 30 s while the address is not up.
- scripts/vm/setup-panel.ps1 (just vm panel-setup) and just vm panel-token.

## Notes

**agent:claude-code/renki-groom** at 2026-10-28T22:45:26Z

Deployed as 4d357a1. From the dev box over Tailscale: / redirects to /login without the token, /api/status answers with it, and the LAN address does not answer. The page, map and feed were checked in a browser against a simulated plugin on the dev box. Criteria 4 and 6 wait on Sothr opening it from a tailnet device with the real bot. Not checked: whether Windows showed a firewall prompt on the VM desktop; the RenkiPanel program rule should prevent one.

**agent:claude-code/renki-groom** at 2026-10-28T23:24:05Z

Sothr opened the panel from a tailnet device on 2026-10-28 ('That panel looks good'). The map draws the trail and home from the real game; players and creatures wait on perception (TKT-01M62BMD1).

**agent:claude-code/renki-groom** at 2026-10-29T05:42:44Z

2026-10-29: Nimah's screenshot of the panel shows the map drawing the bot, Nimah, home, structures, resources and the left-behind diamond. It works but is hard to read at a glance; that is TKT for the map readability, filed today.

## Summary

The panel is served on the tailnet behind a token login, with a live header, a map, the inventory and a filtered feed of chat, tasks, traces and decisions. Seen in use on 2026-10-29.
