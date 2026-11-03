---
schema: 4
id: TKT-01M6FVN1XGRBWA75WJWFMYV2V9
title: "Base areas: separate sleeping, storage and return places"
type: task
status: done
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - movement
assignees: []
milestone: M5
parent: null
origin: null
dependencies:
  - TKT-01M62VXYZMVEP74QVV9SRKDWGH
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-03T04:46:04Z
updated_at: 2026-11-03T06:56:59Z
created_by:
  id: agent:codex/home-survey-planning
  name: ""
updated_by:
  id: agent:codex/home-survey-delivery
  name: ""
extensions: {}
---

## Description

Deliver configurable per-world named base areas for the owner's relocated home (roughly 50–80 m across, sleeping and storage about 50 m apart). The owner authorized grooming, promotion and implementation while away.

### Model and behavior

Save an independent center/radius (10–250 m), total survey time (120–1200 s) and travel allowance (50–4000 m), plus return, sleeping and multiple storage waypoint bindings. Reuse shared waypoint/object memory and preserve old-base markers. CAS edits and failed saves must preserve prior state. A rename/deletion invalidates a role until explicitly rebound. Legacy behavior remains when no base is selected.

Resolve wood delivery against bound storage zones and marker rest against the sleeping place, preserving local ownership, shelter and floor checks. Selected return supplies idle/return home context. Audit food/resupply constraints and document the remaining point-based assumptions; this feature does not migrate unrelated behavior definitions or enable portal traversal.

### Verification

Focused just check controller test-packages for home, behavior, goal and app cover separate buildings, elevated floors, multiple places, missing roles, legacy fallback, world isolation, concurrent edits and rollback. Run just check ci before committing and just check review afterward. Native separated-building route proof belongs to the home-survey ticket.

## Acceptance criteria

- [x] A named base has an owner-selected boundary independent of its sleeping, storage and default return places, and permits multiple same-purpose sites
- [x] Existing waypoint/object memory is reused with explicit rename, deletion, version, owner-edit, rollback and old/new base behavior
- [x] Wood delivery can select bound storage at least 50 m from sleeping without changing legacy behavior for bases without role bindings
- [x] Rest resolves the sleeping place while retaining owned-bed, floor, shelter and native sleep requirements instead of using the storage or survey center
- [x] The plan audits other home-point consumers and tests separated buildings, multiple sites, missing roles, elevated floors and legacy fallback; just check ci passes

## Implementation plan

The owner explicitly authorized filing, grooming, promotion and completion while away. This supersedes the earlier planning-only hold. They estimate a 50–80 m total base span and require configurable scope rather than a fixed radius. The unmeasured route type is not permission for portal traversal; the first implementation uses native dry supported land routes and reports gaps.

Deliver the three local feature tickets together: persistent configurable named base areas with separate sleeping/storage/return role bindings and legacy fallback; scoped multi-place waypoint learning preserving owner edits, suppression and old-base records; and a durable bounded familiarization profile with measured terrain progress, per-leg native distance/trail receipts, supported arrival/route evidence, reserved return, status, stop/takeover and panel/chat controls. Circle center/radius, total duration and travel allowance are saved per base. Use conservative configurable defaults; do not infer a radius from the owner's estimated span.

Reuse the existing goal host and scheduler, with a dedicated native bounded-movement option on move_to rather than wait for the full general-purpose M7 exploration stack. This local workflow enumerates bounded nearby approaches, revalidates actual loaded terrain/objects, records blocked attempts and reports partial results. It does not claim general expedition readiness, autonomous survival, minimap exploration, portal traversal or all-object completeness. Replace the local ticket's dependencies on the broad explore runner/panel epics with the concrete local implementation; retain those tickets as references and leave their M2/M3 dependency gates unchanged.

Test persistent migration/rollback/world isolation, roles 50 m apart, multiple sites, concurrent edits and suppression; native containment/dry navigation/distance limits and receipts; scoped coverage/provenance; uncertainty/restart, budget reserve, unknown/blocked areas, floor-aware arrival, stop/takeover; API authentication/CSRF and owner grammar. Use focused just check recipes, then just check ci before commits. Run just check review on committed changes, address findings, publish a PR and deploy under the VM lease. Record actual live outcomes and remaining limitations on tickets; never tick an unearned acceptance criterion.

## Notes

**agent:codex/home-survey-delivery** at 2026-11-03T06:56:59Z

Implemented configurable named bases and separate roles, protected scoped multi-place learning and the durable bounded familiarize profile. Final just check ci passed: .checks/20261103T065317Z-7PTGdo.log (197 s), including 60/60 desktop/mobile fixture scenarios, controller race suite, plugin DLL build/Core/interop tests and all repository tooling checks. Focused survey race checks passed three consecutive runs: .checks/20261103T065308Z-LNJQsO.log. Native movement args share a Go/C# fixture; wrong floor, failed return, diminishing food, uncertain restart, scope edits, queued takeover and running native stop are covered. Docs/EXPLORATION.md records implemented commands/configuration, retained food/home constraints, provenance eviction and the temporary-here deferral. Committed review and deployment/live evidence follow; offline evidence does not establish the actual separated buildings route.

## Summary

Persistent per-world base configuration, independent sleeping/storage/return roles, CAS/rollback and wood/rest integration delivered with documented legacy food constraints. Full just check ci passed; see latest note for retained evidence.
