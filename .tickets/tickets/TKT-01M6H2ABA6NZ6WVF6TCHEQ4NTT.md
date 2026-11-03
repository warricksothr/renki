---
schema: 4
id: TKT-01M6H2ABA6NZ6WVF6TCHEQ4NTT
title: Search nearby or listed chests for requested items
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - plugin
  - chat
  - live
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/home-survey-delivery
  branch: t3/map-home-location
  worktree: /home/sothr/.t3/worktrees/renki/t3-b02dd361
  commit: cc5f323f40b598ada0f105f7458156e2fa92f495
  session: null
  claimed_at: 2026-11-03T16:01:48Z
  expires_at: null
archive: null
created_at: 2026-11-03T16:01:47Z
updated_at: 2026-11-03T16:01:48Z
created_by:
  id: agent:codex/home-survey-delivery
  name: ""
updated_by:
  id: agent:codex/home-survey-delivery
  name: ""
extensions: {}
---

## Description

The owner asked Tas to look for wood in a nearby chest, but it did not inspect storage. Existing household food code probes chests with withdrawals and is not a general owner-requested search.

Add a reusable bounded search_chests behavior for a pinned explicit chest list or nearby radius, with deterministic chat entry for looking for an item. Native read-only inspection must request ordinary chest access, publish a task-correlated current snapshot, and move no items. Search records found counts/locations and checked, skipped, failed and unknown candidates; it must not equate stale remembered stock or a partial candidate set with complete absence. It is cancellable, yields to newer orders/retreat, and is available to event/manual behavior triggers.

## Acceptance criteria

- [ ] Read-only native inspection respects ordinary access and movement controls, publishes correlated contents and never transfers items.
- [ ] Search accepts an explicit chest list or configurable radius, bounds work and distinguishes checked matches, checked absence, inaccessible and unobserved storage.
- [ ] Owner chat requests such as look for wood in a nearby chest invoke the behavior, report results and preserve newer orders and retreat.
- [ ] Regression tests, full CI and committed review pass; native searches and unchanged inventories are retained separately.

## Implementation plan

Add inspect_container as a native body task sharing chest access/interaction travel with transfers, but returning before all item moves. Strict arguments and feature/allowlist advertisement identify support. Add search_chests as a one-run behavior with pinned scope, bounded radius/list/time/travel/candidate count, guarded health/threat/freshness checks and per-candidate task-correlated inspection results. Add deterministic owner chat and standard behavior configuration for manual/event triggers, plus concise result reports. Regress access/argument parity, false/stale receipts, failure continuation, scope limits, cancellation and chat authorization. Run focused checks, CI, committed review and checksum-verified deployment, then use only safe fresh idle native probes.
