---
schema: 4
id: TKT-01M6FV6E40SBSMJEHT227XKRK2
title: "Home survey: familiarize with an area and update its places"
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - chat
  - movement
  - live
assignees: []
milestone: M7
parent: TKT-01M66PFAYA1NTW2JHNRDKBK42E
origin: null
dependencies:
  - TKT-01M6FV695N00A10NRMECFMSAHA
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/home-survey-delivery
  branch: t3/map-home-location
  worktree: /home/sothr/.t3/worktrees/renki/t3-b02dd361
  commit: 0411381e7faf1eded3bc256781caa6e8ab4ab9de
  session: null
  claimed_at: 2026-11-03T06:29:51Z
  expires_at: null
archive: null
created_at: 2026-11-03T04:38:05Z
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

Deliver a bounded local familiarization profile for the relocated home through the existing durable goal host. The owner authorized implementation and delivery while away, estimates a 50–80 m base span and requires configurable scope. The broader M7 exploration dependency gates remain unchanged.

### Controls and scope

Configure and select a named base in Places & storage. Tas, familiarize yourself with home (also this area, the area, or survey home) starts that saved base. Tas, survey status reports progress; Tas, stop, panel Stop survey or a newer body order wins. Chat can edit radius and sleeping/storage/return bindings. The panel supports center, radius/diameter, total time/travel and multiple storage places. Temporary here anchors and per-run overrides are deferred; this area means the saved selected base.

Before persisting work, validate fresh idle body, original world/observer, bounded_move_v1, existing move/stop permission, supported dry footing, supplies, health and absence of nearby observed threat. Pin base/role versions and fail closed on edits, world/session changes, injury or stale observations. This profile never harvests, opens chests, crosses portals or grants combat permission.

### Evidence and return

Visit the independent return, sleeping and storage approaches in both directions, then nearby fresh measured-ground approaches inside the circle. Record directed native legs, 3D travel including detours, floor-aware post-receipt arrival and bounded trails. Remember failed approaches and reserve return time/travel/tasks before outbound work. Revalidate reverse visited stops and pinned final return. Missing receipts and uncertain restart never replay a leg or reset budgets.

Count canonical 4 m terrain samples in the circle as measured/new/refreshed/unknown/stale, using immutable first-receipt observer provenance (with documented retention limits). Keep visited ground and observed objects separate. Fresh observations refresh scoped bot places with owner/other-base/suppression protection. Full refreshed terrain, all role visits, no failed/unresolved legs and supported return earn success; otherwise report partial/gaps, or a diagnostic without claiming return. No result proves every room, floor, container or door was inspected.

### Verification and live evidence

Focused just check controller test-packages cover home, atlas, behavior, goal, app, chat and protocol. just check plugin test-core BoundedTravel covers native budgets and shared args fixtures; real plugin DLL must build. Integration checks exercise queued takeover and running stop. Run just check ci, commit and just check review before pushing. Deploy under the VM lease and retain safe live before/after coverage, waypoint diff, gaps and return evidence. A 50 m separated-building native route remains a distinct earned criterion; leave it unticked if the client/start/roles do not permit it.

## Acceptance criteria

- [x] Status separates measured terrain, new terrain, visited ground and observed objects, and reports gaps without equating terrain coverage with complete base inspection
- [x] The profile reuses the goal host and bounded noncombat survey and return runners, preserving elapsed/travel budgets and existing permissions
- [x] Fresh discoveries refresh bot-managed places and report protected old-base markers without changing home or erasing historical knowledge
- [x] Stop, takeover, world changes and uncertain restart cannot dispatch another outbound leg or reset budgets; offline checks and just check ci pass
- [ ] Live new-home evidence under the VM lease records coverage before/after, waypoint changes, unresolved gaps and checked arrival at the pinned return anchor
- [x] Survey boundary and return point are independent of sleeping and storage; all selected places and their versions are pinned, with no assumed 50 m radius
- [ ] A separated-buildings trial checks sleeping/storage approaches and connecting paths in both directions, preserving actual floor heights and reporting gaps independently of terrain coverage
- [x] Owner commands and panel start a validated bounded survey of the saved selected base; missing capabilities or stale scope refuse before persisting work (temporary here anchors are deferred)

## Implementation plan

The owner explicitly authorized filing, grooming, promotion and completion while away. This supersedes the earlier planning-only hold. They estimate a 50–80 m total base span and require configurable scope rather than a fixed radius. The unmeasured route type is not permission for portal traversal; the first implementation uses native dry supported land routes and reports gaps.

Deliver the three local feature tickets together: persistent configurable named base areas with separate sleeping/storage/return role bindings and legacy fallback; scoped multi-place waypoint learning preserving owner edits, suppression and old-base records; and a durable bounded familiarization profile with measured terrain progress, per-leg native distance/trail receipts, supported arrival/route evidence, reserved return, status, stop/takeover and panel/chat controls. Circle center/radius, total duration and travel allowance are saved per base. Use conservative configurable defaults; do not infer a radius from the owner's estimated span.

Reuse the existing goal host and scheduler, with a dedicated native bounded-movement option on move_to rather than wait for the full general-purpose M7 exploration stack. This local workflow enumerates bounded nearby approaches, revalidates actual loaded terrain/objects, records blocked attempts and reports partial results. It does not claim general expedition readiness, autonomous survival, minimap exploration, portal traversal or all-object completeness. Replace the local ticket's dependencies on the broad explore runner/panel epics with the concrete local implementation; retain those tickets as references and leave their M2/M3 dependency gates unchanged.

Test persistent migration/rollback/world isolation, roles 50 m apart, multiple sites, concurrent edits and suppression; native containment/dry navigation/distance limits and receipts; scoped coverage/provenance; uncertainty/restart, budget reserve, unknown/blocked areas, floor-aware arrival, stop/takeover; API authentication/CSRF and owner grammar. Use focused just check recipes, then just check ci before commits. Run just check review on committed changes, address findings, publish a PR and deploy under the VM lease. Record actual live outcomes and remaining limitations on tickets; never tick an unearned acceptance criterion.

## Notes

**agent:codex/home-survey-delivery** at 2026-11-03T06:56:59Z

Implemented configurable named bases and separate roles, protected scoped multi-place learning and the durable bounded familiarize profile. Final just check ci passed: .checks/20261103T065317Z-7PTGdo.log (197 s), including 60/60 desktop/mobile fixture scenarios, controller race suite, plugin DLL build/Core/interop tests and all repository tooling checks. Focused survey race checks passed three consecutive runs: .checks/20261103T065308Z-LNJQsO.log. Native movement args share a Go/C# fixture; wrong floor, failed return, diminishing food, uncertain restart, scope edits, queued takeover and running native stop are covered. Docs/EXPLORATION.md records implemented commands/configuration, retained food/home constraints, provenance eviction and the temporary-here deferral. Committed review and deployment/live evidence follow; offline evidence does not establish the actual separated buildings route.
