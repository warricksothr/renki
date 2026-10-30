---
schema: 4
id: TKT-01M674B8MJ1ESMHMXAYAC09PX0
title: Expose bounded runtime camera overrides to agent tools
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
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-milestones
  branch: t3/camera-browser-evidence
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 43a4add5f2cff1ec53d75cc1b91d1a849b3d2435
  session: null
  claimed_at: 2026-10-30T19:27:24Z
  expires_at: null
archive: null
created_at: 2026-10-30T19:24:50Z
updated_at: 2026-10-30T19:34:14Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

The owner requests agent-accessible runtime camera configuration beside JPEG and shared map retrieval, so an agent can temporarily increase diagnostic capture frequency and return to the normal rate without restarting the game. Existing screenshot.intervalS settings already switch live. Add explicit recipes and an authenticated controller-owned temporary override with an expiry, leaving persisted settings unchanged so interruption/restart cannot strand the fast rate. Use seconds per frame: 1 is fastest, default5, valid1-300. Allow temporary disable/slow intervals for overhead diagnosis. Expose effective/base settings and remaining override metadata, allow matching-ID early release, and let explicit normal configuration changes displace a temporary override. Preserve all native hard bounds and current image scoping.

## Acceptance criteria

- [ ] Camera interval and enabled state change live through authenticated agent-accessible tools
- [ ] A bounded temporary override restores normal settings on expiry or release and never persists across controller restart
- [ ] Status reports effective and base settings; manual changes and connection replacement displace stale overrides
- [ ] Tests and live image cadence verify fast capture and automatic restoration without restarting the game

## Implementation plan

Add a controller-memory-only diagnostic camera lease, validated interval1-300s and duration1-300s, scoped to current in-world connection/world UID. Expose start/release under authenticated camera API with action header, show effective/base/override status beside image metadata, honor camera pins and let runtime settings changes displace the lease. A bounded watcher restores base on expiry, release, pin/scope change; reconnect/restart gets only persisted base settings. Serialize sends and target the original connection to avoid applying stale overrides to replacements. Provide explicit status, persistent interval and temporary diagnostic just tools alongside existing JPEG/map capture, plus docs and core/API/scope/restoration tests. Verify switching5->1->5 with the game staying online. Preserve native hard floor and bounds.

## Notes

**agent:codex/renki-milestones** at 2026-10-30T19:34:14Z

Implemented authenticated camera commands, in-memory world/connection-scoped diagnostic leases, auto-expiry/restoration, matching-ID release, normal-settings displacement and agent pin enforcement. Added explicit just vm camera-config, just vm camera-rate and just vm camera-diagnostic recipes beside existing JPEG/map snapshot tools. API exposes effective/base/override metadata; documents seconds-per-capture and restoration semantics.

Focused API/wire tests passed via just check controller test-packages ./internal/app ./internal/session ./internal/panel (.checks/20261030T193251Z-y0BPnL.log). Coverage includes auth/action header, malformed/unsupported commands, hard bounds, real timed expiry, persisted file byte equality, new-App restart, body authority unchanged, stale releases, same-value user changes, initial/new pins, world changes, disconnect/replacement and missing world UID. just check ci passed (.checks/20261030T193318Z-2SmjgG.log). Live same-connection5->1->5 evidence remains required after landing/deployment; criteria stay open until that runs.
