---
schema: 4
id: TKT-01M6KNVKQZGF26CSJ5T49BH5MV
title: Start private OBS streaming alongside Tas
type: task
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - vm
  - tooling
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/portal-deploy
  branch: t3/renki-portal-travel-startup
  worktree: /home/sothr/.t3/worktrees/renki/t3-6c0da0c0
  commit: 501ca8d3ca755d7de1888f92f3a1e27e92bd6d7e
  session: null
  claimed_at: 2026-11-04T16:23:00Z
  expires_at: null
archive: null
created_at: 2026-11-04T16:21:45Z
updated_at: 2026-11-04T16:25:59Z
created_by:
  id: agent:codex/portal-deploy
  name: ""
updated_by:
  id: agent:codex/portal-deploy
  name: ""
extensions: {}
---

## Description

The owner requested starting OBS and the private stream alongside Tas. After the VM reboot the installed OBS, relay, verifier and watchdog tasks were stopped. Add an idempotent combined startup recipe that restores existing private video tasks, waits for authenticated OBS, verifies local output and starts the panel/game under the VM lease. Retain independent stream lifecycle and fresh-capture standby behavior. Verify repeat startup without resetting output.

## Acceptance criteria

- [ ] One guarded recipe starts the panel, private stream and Tas
- [ ] Repeated startup retains active private output and fresh-capture verification

## Implementation plan

Restore the installed OBS/relay/verifier/watchdog tasks without reconfiguration, wait for authenticated OBS, start only fixed private output, and verify active output under the VM lease. Expose just vm online for panel/video/game startup while retaining independent stream lifecycle. Make active-output startup idempotent and cover output destination/order with websocket tests; verify startup twice on Blight.
