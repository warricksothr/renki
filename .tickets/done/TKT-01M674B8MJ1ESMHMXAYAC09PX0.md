---
schema: 4
id: TKT-01M674B8MJ1ESMHMXAYAC09PX0
title: Expose bounded runtime camera overrides to agent tools
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
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-30T19:24:50Z
updated_at: 2026-10-30T19:46:22Z
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

- [x] Camera interval and enabled state change live through authenticated agent-accessible tools
- [x] A bounded temporary override restores normal settings on expiry or release and never persists across controller restart
- [x] Status reports effective and base settings; manual changes and connection replacement displace stale overrides
- [x] Tests and live image cadence verify fast capture and automatic restoration without restarting the game

## Implementation plan

Add a controller-memory-only diagnostic camera lease, validated interval1-300s and duration1-300s, scoped to current in-world connection/world UID. Expose start/release under authenticated camera API with action header, show effective/base/override status beside image metadata, honor camera pins and let runtime settings changes displace the lease. A bounded watcher restores base on expiry, release, pin/scope change; reconnect/restart gets only persisted base settings. Serialize sends and target the original connection to avoid applying stale overrides to replacements. Provide explicit status, persistent interval and temporary diagnostic just tools alongside existing JPEG/map capture, plus docs and core/API/scope/restoration tests. Verify switching5->1->5 with the game staying online. Preserve native hard floor and bounds.

## Notes

**agent:codex/renki-milestones** at 2026-10-30T19:34:14Z

Implemented authenticated camera commands, in-memory world/connection-scoped diagnostic leases, auto-expiry/restoration, matching-ID release, normal-settings displacement and agent pin enforcement. Added explicit just vm camera-config, just vm camera-rate and just vm camera-diagnostic recipes beside existing JPEG/map snapshot tools. API exposes effective/base/override metadata; documents seconds-per-capture and restoration semantics.

Focused API/wire tests passed via just check controller test-packages ./internal/app ./internal/session ./internal/panel (.checks/20261030T193251Z-y0BPnL.log). Coverage includes auth/action header, malformed/unsupported commands, hard bounds, real timed expiry, persisted file byte equality, new-App restart, body authority unchanged, stale releases, same-value user changes, initial/new pins, world changes, disconnect/replacement and missing world UID. just check ci passed (.checks/20261030T193318Z-2SmjgG.log). Live same-connection5->1->5 evidence remains required after landing/deployment; criteria stay open until that runs.

**agent:codex/renki-milestones** at 2026-10-30T19:46:21Z

PR #101 landed as861c31c; just check vm install verified package checksums (.checks/20261030T193509Z-TBTyXL.log). Final repeatable just check vm camera-check passed (.checks/20261030T194345Z-NMluHY.log). Evidence .checks/camera-override-live-20261030T194345538838Z.json and .checks/audit-window-VxiXba.jsonl prove the same in-world connection/world throughout. Actual capture medians:5.002s before (3frames),1.003s during15-second diagnostic (15frames),5.003s after automatic expiry (6frames); worst capture14.13ms. Camera setting entries before/after are identical; lease never persisted. JPEG .checks/camera-20261030T194445257621Z.jpg and matching JSON, plus shared-renderer map .checks/map-snapshot-20261030T194445299424Z/map.html, were retrieved successfully. API/unit coverage separately verifies pins, stale IDs, restart and replacement scopes.

The live baseline initially differed from expectation: saved normal interval was1s. It was set to5s for the documented test, then explicitly restored to1s afterwards. Two helper failures were corrected before the final pass: module-relative output directory and Windows PowerShell wrapping API arrays as {value,Count}. Regression coverage now handles the wrapper and requires actual scoped timestamps in all three phases rather than inferring cadence from settings. An earlier successful native cycle is retained in .checks/audit-window-AKTKII.jsonl; its summary write failed, so final evidence is the repeatable helper pass above.

The helper always logs out/stops and leaves controller/panel online. Final game is offline, panel online, no active override; saved capture enabled1s. Full gate after helper fixes passed: just check ci (.checks/20261030T194346Z-vL0quf.log). Minimized rendering remains untested on the separate original camera ticket. M2/M3 combat/harvest acceptance remains open.

## Summary

Runtime diagnostic API and tools landed/deployed in PR #101 at861c31c. Authenticated agent controls honor pins and expose effective/base/scope/expiry metadata; temporary settings are memory-only and restore on expiry/release or displacement. Focused API/real-wire regressions and full CI pass. The live same-connection5->1->5 trial verified actual5.002/1.003/5.003s median capture intervals, unchanged saved settings and JPEG/shared-map retrieval. Final helper evidence is .checks/camera-override-live-20261030T194345538838Z.json. just vm camera-check retains bounded evidence and automatically logs out/stops, leaving the panel online. Original saved1s rate restored after the test.
