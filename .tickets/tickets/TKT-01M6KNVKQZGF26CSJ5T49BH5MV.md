---
schema: 4
id: TKT-01M6KNVKQZGF26CSJ5T49BH5MV
title: Start private OBS streaming alongside Tas
type: task
status: review
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
claim: null
archive: null
created_at: 2026-11-04T16:21:45Z
updated_at: 2026-11-04T16:41:30Z
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

- [x] One guarded recipe starts the panel, private stream and Tas
- [x] Repeated startup retains active private output and fresh-capture verification

## Implementation plan

Restore the installed OBS/relay/verifier/watchdog tasks without reconfiguration, wait for authenticated OBS, start only fixed private output, and verify active output under the VM lease. Expose just vm online for panel/video/game startup while retaining independent stream lifecycle. Make active-output startup idempotent and cover output destination/order with websocket tests; verify startup twice on Blight.

## Notes

**agent:codex/portal-deploy** at 2026-11-04T16:41:29Z

Reviewed bdd70e7 installed with checksum/preflight/receipt verification; capture adapter update passed in 9s. First just check vm online refused before game launch because authenticated OBS remained unavailable. The installed OBS 33 process was stuck at its unclean-shutdown dialog; its current startup log contained only that message. Upstream OBS 33 removed the old shutdown-check argument. Added a narrow recovery that restarts only that dedicated startup-only state with no control listener, archives stale portable run markers only while OBS is stopped, and preserves crash logs/configuration.

just check streaming windows-online passed in19s (.checks/20261104T163711Z-ZVVQv9.log): recovered the dialog, archived two markers, restored all four tasks, and verified active private output. Repeating through just check vm online passed in61s (.checks/20261104T163751Z-rsnInl.log): existing stream duration increased from1733ms to34533ms without resetting, then Tas joined on bdd70e7. GET /api/video subsequently reported gameplay with fresh scope1:DizzysHellheim:827096491, proving changing native capture frames passed the existing verifier. Standby was retained during offline startup; no external destination/account was configured. The recovery correction passed full just check ci in179s (.checks/20261104T163752Z-t3DOnR.log); committed review follows. This does not claim completion of the older OBS source-loss/watchdog trial ticket.

## Summary

Added guarded just vm online and independent just streaming windows-online startup. Verified recovery from the actual OBS 33 unclean-shutdown dialog, repeat startup retaining output duration, Tas joining and fresh gameplay capture. Full CI passed; final committed review is pending.
