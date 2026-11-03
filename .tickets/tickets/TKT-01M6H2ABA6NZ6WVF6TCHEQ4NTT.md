---
schema: 4
id: TKT-01M6H2ABA6NZ6WVF6TCHEQ4NTT
title: Search nearby or listed chests for requested items
type: task
status: review
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
claim: null
archive: null
created_at: 2026-11-03T16:01:47Z
updated_at: 2026-11-03T16:40:45Z
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

- [x] Read-only native inspection respects ordinary access and movement controls, publishes correlated contents and never transfers items.
- [x] Search accepts an explicit chest list or configurable radius, bounds work and distinguishes checked matches, checked absence, inaccessible and unobserved storage.
- [x] Owner chat requests such as look for wood in a nearby chest invoke the behavior, report results and preserve newer orders and retreat.
- [ ] Regression tests, full CI and committed review pass; native searches and unchanged inventories are retained separately.

## Implementation plan

Add inspect_container as a native body task sharing chest access/interaction travel with transfers, but returning before all item moves. Strict arguments and feature/allowlist advertisement identify support. Add search_chests as a one-run behavior with pinned scope, bounded radius/list/time/travel/candidate count, guarded health/threat/freshness checks and per-candidate task-correlated inspection results. Add deterministic owner chat and standard behavior configuration for manual/event triggers, plus concise result reports. Regress access/argument parity, false/stale receipts, failure continuation, scope limits, cancellation and chat authorization. Run focused checks, CI, committed review and checksum-verified deployment, then use only safe fresh idle native probes.

## Notes

**agent:codex/home-survey-delivery** at 2026-11-03T16:22:40Z

Implemented native inspect_container with strict shared arguments, feature advertisement, ordinary chest access/ownership and checked dry travel. It publishes a task-correlated fresh snapshot, then returns before stack transfer. Controller search_chests pins radius or explicit candidates, bounds time/travel/chest count, stops on injury/threat/stale data/new orders, and counts only new matching inspection snapshots. Real native travel receipts debit detours; a missing non-rejection receipt stops continuation. Manual runs do not resume after restart; ordinary event/timer definitions also work. Chat supports owner requests including look for wood in a nearby chest and named/radius/list variants, preserving colon IDs and punctuation after Tas.

Focused Go/native protocol, engine, stale/correlated contents, travel accounting, admission/cancellation and chat checks passed; .checks/20261103T162143Z-4MDyZq.log and .checks/20261103T161946Z-AGIa9e.log. Documentation in BEHAVIOURS.md and BRIDGE-PROTOCOL.md. Full gate/review/install follow. Native search and before/after unchanged inventory remain unearned because Tas is emergency-offline during an actual pursuit.

**agent:codex/home-survey-delivery** at 2026-11-03T16:26:59Z

Interrupted searches retain their partial findings and unchecked count in behavior status, including newer owner stop/retreat; no stale continuation or stop is sent to the new owner. A bridge integration regression checks inspection, reporting without withdrawal, and partial-result retention on interruption. Shared inspection fixtures also reject unknown nested scope fields on both Go and C#.

**agent:codex/home-survey-delivery** at 2026-11-03T16:40:45Z

Code landed in2e8c6d1. Full just check ci passed170s (.checks/20261103T163500Z-YLtrJQ.log); committed just check review passed8s with no findings (.checks/20261103T163822Z-KdkkbC.log). just check vm install passed16s with installed controller/Core/plugin checksum verification (.checks/20261103T163834Z-MJnAw1.log). just vm panel-start restored the independent panel without starting the game. Read-only delivered observation confirms offline; Loot remains disabled. The normal settings API added inspect_container while preserving all18 existing task permissions; readback confirmed it. Evidence .checks/new-home-delivered-observation.json, .checks/new-home-delivered-settings.json and .checks/new-home-delivered-behaviours.json. Native survival, double doors, chest inventories, survey/return and bed ownership remain unearned. No game rejoin or dangerous probe was attempted.

## Summary

Implemented, tested, reviewed and installed bounded read-only chest inspection/search with radius/list/chat entry, fresh correlated counts, measured travel and retained partial results. inspect_container is enabled in the existing runtime allowlist. Native searches and before/after unchanged-inventory evidence remain unearned; AC4 stays unchecked while Tas is emergency-offline during an actual pursuit.
