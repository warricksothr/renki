---
schema: 4
id: TKT-01M6EFQN74X0EDYJAVBJJK2DY8
title: "Atlas: first-seen time and a measured-coverage query"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
assignees: []
milestone: M7
parent: TKT-01M62VXZ0J9Q2WH4XN4E6N7F27
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:58:32Z
updated_at: 2026-11-02T15:58:44Z
created_by:
  id: agent:claude-code/renki-plan-m7
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m7
  name: ""
extensions: {}
---

## Description

Exploration success needs one number: how much ground inside an authorized circle the bot measured for the first time since a given instant. The atlas keeps one row per canonical 4 m sample (`samples` in internal/atlas/store.go) with its latest `observed` and `received` times. It can say what is known, not what is new.

Add a first-received time to `samples` (schema 2, migrated in one transaction from schema 1; existing rows get first = received), and a bounded query that counts samples first received at or after `since`, from one source, inside a circle of radius at most 250 m (the goal record's own radius cap in internal/goal/store.go). Each canonical sample stands for 16 m2, so coverage in m2 is count x 16.

This is measured terrain coverage: loaded heightmap sampled within at most 64 m of the bot (plugin/Renki.Plugin/TerrainReporter.cs passes min(64, perceiver radius); the default perceiver radius is 40 m, so 3 tiles). It does not prove every object was visible. Name it "measured terrain coverage" in the API and docs, distinct from map knowledge (minimap) and visited ground.

### Why it matters to M7
Goal 7.3 measures new m2 inside the authorized area, and the explore goal's success predicate needs the same figure. Both read this one query, so the host and the trial supervisor cannot disagree.

### Limits to keep honest
- An equal fresh confirmation, a resend, or a re-confirmed stale sample does not change first-received.
- A sample removed by TTL or capacity eviction and later re-measured gets a new first-received. Document this limit.
- TKT-01M6EFQN91 (Report and keep the ground the bot actually walked) (visited ground) also migrates the atlas schema. Whichever lands second takes the next schema version.

### Files
- internal/atlas/store.go: schema, migration, `Ingest` sets first-received on insert only, new `Coverage` query
- internal/atlas/store_test.go
- internal/app/atlas.go and internal/app/atlas_test.go: `GET /api/atlas/coverage` (authenticated, read-only; internal/panel/panel.go already forwards `GET /api/atlas/`)
- docs/ATLAS.md

### Verification
- `just check controller test-packages ./internal/atlas ./internal/app`
- `just check controller benchmark-atlas`: ingest cost stays within 20 % of the recorded ~7.6 ms per 25-sample batch
- `just check ci`

## Acceptance criteria

- [ ] A schema-1 atlas migrates to schema 2 in one transaction; a test opens a schema-1 file and finds every sample's first-received equal to its received time, and a future schema version is still refused
- [ ] The coverage query counts only samples first received at or after `since` inside the circle; a test re-ingests the same samples (equal fresh, resent, stale re-confirmed) and the count stays 0
- [ ] Samples from another world key, epoch or source are not counted; a radius above 250 m or a non-finite centre is rejected
- [ ] `GET /api/atlas/coverage?world=&x=&z=&r=&since=` returns count, m2 (count x 16), the circle and `since`, and refuses unauthenticated requests (test)
- [ ] docs/ATLAS.md describes measured terrain coverage, its sensor radius and the eviction limit
