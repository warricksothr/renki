---
schema: 4
id: TKT-01M6FV695N00A10NRMECFMSAHA
title: "Waypoints: refresh bot places around a selected area"
type: task
status: done
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - chat
assignees: []
milestone: M7
parent: null
origin: null
dependencies:
  - TKT-01M62VXYZMVEP74QVV9SRKDWGH
  - TKT-01M6FVN1XGRBWA75WJWFMYV2V9
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-03T04:38:00Z
updated_at: 2026-11-03T06:57:00Z
created_by:
  id: agent:codex/home-survey-planning
  name: ""
updated_by:
  id: agent:codex/home-survey-delivery
  name: ""
extensions: {}
---

## Description

Deliver explicit scoped learning during a validated familiarization survey. The owner authorized implementation after the earlier planning discussion.

### Scope and protection

Use actual fresh structures/resources observed in the original world/session and selected base circle. Classify storage, rest, cooking and resource places and support multiple same-purpose clusters. Retain entity/prefab evidence, position, timestamp and base provenance. A moved bot-observed entity can refresh its location. Background discovery stays fill-only.

Under the memory lock, validate the pinned active-base revision and preserve owner edits, bound roles, deletion suppression, other bases and old owner-managed home storage. Missing/incomplete scans never delete evidence. Preserve CAS and failed-save rollback. Return added, updated, protected, unchanged and skipped observations; learning location is not container stock/access or complete mapping. Primary sleeping/storage choices remain explicit owner bindings.

### Verification

Use just check controller test-packages for home, app and chat, then just check ci and committed just check review. Record live waypoint changes and preserved old-base records with the home-survey evidence.

## Acceptance criteria

- [x] An explicit scoped refresh selects fresh evidenced candidates in the requested world and circle, with deterministic selection around its pinned center
- [x] Concurrent owner edits win, failed saves roll back, and no candidate or incomplete evidence removes an existing place
- [x] The result distinguishes added, refreshed, protected and unchanged places without claiming complete mapping or stock knowledge
- [x] Focused checks and just check ci pass; existing passive discovery remains fill-only
- [x] Refresh updates only the selected base, preserves other bases and owner-managed places and deletion suppression, and supports multiple same-purpose sites with explicit primary-role selection

## Implementation plan

The owner explicitly authorized filing, grooming, promotion and completion while away. This supersedes the earlier planning-only hold. They estimate a 50–80 m total base span and require configurable scope rather than a fixed radius. The unmeasured route type is not permission for portal traversal; the first implementation uses native dry supported land routes and reports gaps.

Deliver the three local feature tickets together: persistent configurable named base areas with separate sleeping/storage/return role bindings and legacy fallback; scoped multi-place waypoint learning preserving owner edits, suppression and old-base records; and a durable bounded familiarization profile with measured terrain progress, per-leg native distance/trail receipts, supported arrival/route evidence, reserved return, status, stop/takeover and panel/chat controls. Circle center/radius, total duration and travel allowance are saved per base. Use conservative configurable defaults; do not infer a radius from the owner's estimated span.

Reuse the existing goal host and scheduler, with a dedicated native bounded-movement option on move_to rather than wait for the full general-purpose M7 exploration stack. This local workflow enumerates bounded nearby approaches, revalidates actual loaded terrain/objects, records blocked attempts and reports partial results. It does not claim general expedition readiness, autonomous survival, minimap exploration, portal traversal or all-object completeness. Replace the local ticket's dependencies on the broad explore runner/panel epics with the concrete local implementation; retain those tickets as references and leave their M2/M3 dependency gates unchanged.

Test persistent migration/rollback/world isolation, roles 50 m apart, multiple sites, concurrent edits and suppression; native containment/dry navigation/distance limits and receipts; scoped coverage/provenance; uncertainty/restart, budget reserve, unknown/blocked areas, floor-aware arrival, stop/takeover; API authentication/CSRF and owner grammar. Use focused just check recipes, then just check ci before commits. Run just check review on committed changes, address findings, publish a PR and deploy under the VM lease. Record actual live outcomes and remaining limitations on tickets; never tick an unearned acceptance criterion.

## Notes

**agent:codex/home-survey-delivery** at 2026-11-03T06:56:59Z

Implemented configurable named bases and separate roles, protected scoped multi-place learning and the durable bounded familiarize profile. Final just check ci passed: .checks/20261103T065317Z-7PTGdo.log (197 s), including 60/60 desktop/mobile fixture scenarios, controller race suite, plugin DLL build/Core/interop tests and all repository tooling checks. Focused survey race checks passed three consecutive runs: .checks/20261103T065308Z-LNJQsO.log. Native movement args share a Go/C# fixture; wrong floor, failed return, diminishing food, uncertain restart, scope edits, queued takeover and running native stop are covered. Docs/EXPLORATION.md records implemented commands/configuration, retained food/home constraints, provenance eviction and the temporary-here deferral. Committed review and deployment/live evidence follow; offline evidence does not establish the actual separated buildings route.

## Summary

Explicit scoped place refresh delivered during familiarization, with multiple clusters, moved-entity refresh, owner/bound-role/other-base/suppression protections and atomic rollback. Full just check ci passed; see latest note for retained evidence.
