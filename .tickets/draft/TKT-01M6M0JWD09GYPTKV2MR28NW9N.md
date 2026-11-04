---
schema: 4
id: TKT-01M6M0JWD09GYPTKV2MR28NW9N
title: Unify behavior decision audits and immutable inspection snapshots
type: task
status: draft
status_reason: null
priority: high
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
created_at: 2026-11-04T19:29:13Z
updated_at: 2026-11-04T19:29:13Z
created_by:
  id: agent:codex/explore-night
  name: ""
updated_by:
  id: agent:codex/explore-night
  name: ""
extensions: {}
---

## Description

Implement the highest-leverage follow-ups from the owner-requested static subagent review in docs/BEHAVIOUR-REVIEW.md: make Engine.List/Loot.Inspect observationally pure via cached immutable scheduler snapshots; standardize scoped versioned decision/transition audit events consumed by the inspector; add deterministic semantic scenarios for decisions, takeover, reconnect, restart and persistence failure. Preserve existing Runner/Template, durable host adapters, automation leases and native authority; avoid a behavior-tree/DSL rewrite. Current food changes provide local pure explained decisions and a side-effect-free workload ownership getter, but do not solve the cross-routine contract or existing mutating status reads. Split into concrete dependent tickets when selected.
