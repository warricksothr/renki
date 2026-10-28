---
schema: 4
id: TKT-01M61T36P84513FQAPKTN76YN4
title: "Plugin: auto character select and join on launch"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - vm
assignees: []
milestone: M1
parent: TKT-01M61T36B839WT7KEZ14TDMB8B
origin: null
dependencies:
  - TKT-01M61T36HB0PY76DJNZ1AGHCG2
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-28T17:49:28Z
updated_at: 2026-10-28T17:49:28Z
created_by:
  id: agent:claude-code/valheim-setup
  name: Claude Code
updated_by:
  id: agent:claude-code/valheim-setup
  name: Claude Code
extensions: {}
---

## Description

`+connect` is parsed by FejdStartup, but character select still waits for
a click. Pick the configured character from FejdStartup.m_profiles and start
it. docs/BRIDGE-PROTOCOL.md section 12, question 3.

## Acceptance criteria

- [ ] Launching with `+connect host:port` lands the bot's character in the world with no clicks
- [ ] A server password from configuration is used when one is asked for
- [ ] `logout` saves and returns to the menu via Game.Logout(true, true)
