---
schema: 4
id: TKT-01M66PFAYA1NTW2JHNRDKBK42E
title: Exploration goals after M2 and M3
type: epic
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - movement
  - docs
assignees: []
milestone: null
parent: null
origin: null
dependencies:
  - TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
  - TKT-01M62N2557R401S3E6JECSAPN4
blocks_on: children
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-30T15:22:24Z
updated_at: 2026-10-30T15:22:24Z
created_by:
  id: agent:codex/renki-exploration
  name: ""
updated_by:
  id: agent:codex/renki-exploration
  name: ""
extensions: {}
---

## Description

Capture the owner's requested post-M2/M3 direction: persistent exploration goals supported by bounded native/controller routines for preparation, frontier selection, travel/survey, recovery and return. This is future design/backlog work, not permission to start autonomous exploration before the two milestones are complete.

Reuse the existing persistent goal/checkpoint machinery, per-world terrain/landmark/danger/player memory, waypoints, player permissions and panel. Extend the host with an exploration-specific specification, budgets and completion evidence; the current combat-only goal implementation does not already support exploration. Retain the existing explored-map coverage ticket rather than duplicating it.

See docs/EXPLORATION.md for the proposed first benchmark, evidence/provenance model, supporting behaviors, rollout order and open design decisions. Broader biome searches, resource prospecting, roads, portals, boats and overseas expeditions follow demonstrated bounded land exploration.

## Acceptance criteria

- [ ] M2 and M3 are complete before exploration implementation is promoted
- [ ] Bounded land exploration earns new observed coverage and returns to its designated anchor with explicit completion evidence
- [ ] Preparation, travel/survey, retry selection, recovery and return respect native safety, permissions and durable goal budgets
- [ ] Panel shows planned/observed/visited coverage, discoveries, uncertain routes, failures and remaining expedition budget
- [ ] Replay and bounded live benchmarks cover meaningful discovery, safe return, blocked frontiers, interruption and honest partial outcomes
