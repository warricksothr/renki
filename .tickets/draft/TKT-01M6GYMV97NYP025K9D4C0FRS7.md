---
schema: 4
id: TKT-01M6GYMV97NYP025K9D4C0FRS7
title: "Routes: save waypoint chains with durable execution progress"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - movement
  - live
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
created_at: 2026-11-03T14:57:37Z
updated_at: 2026-11-03T14:58:20Z
created_by:
  id: agent:codex/home-survey-delivery
  name: ""
updated_by:
  id: agent:codex/home-survey-delivery
  name: ""
extensions: {}
---

## Description

Owner discussion on 2026-11-03: inspect retained atlas data, create a chain of waypoints, run it as one route, and remember which leg Tas reached. Current code has named waypoints, native reusable trails and survey-specific stops/checkpoints, but no general saved itinerary with durable progress. This is a future extension; the current old-house to new-home trial uses individual native tasks.

Reuse the durable goal host and shared scheduler. Store a per-world named ordered route; accept named places or explicit candidate coordinates, and pin their coordinates/revisions at run start. Atlas terrain may propose candidates and expose unknown gaps, but cannot prove walking clearance or invent supported floors. Each leg uses checked native travel and a fresh receipt plus supported arrival before the cursor advances. Persist the last confirmed stop, next leg, total budgets and failure reason. Lost/uncertain tasks must not replay across restart; explicit resume requires reconciliation and fresh scope/readiness. Stop on owner takeover, world/session changes or route/place edits. Distinguish blocked/partial/complete and offer a reviewable replan rather than skipping failed legs. Portals remain distinct future transitions.

## Acceptance criteria

- [ ] Authenticated controls save, inspect, start, stop and explicitly resume a named route with pinned ordered stops.
- [ ] Progress identifies the last verified stop, next leg, consumed budgets and actionable failure or gap.
- [ ] Only matched native receipts and fresh supported arrival advance the durable cursor; restart uncertainty, edits and newer body orders cannot replay or skip work.
- [ ] Atlas candidate routes retain unknown terrain and floor gaps and every attempted leg receives live physical and hostile validation.
