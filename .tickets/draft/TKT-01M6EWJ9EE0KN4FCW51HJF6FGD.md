---
schema: 4
id: TKT-01M6EWJ9EE0KN4FCW51HJF6FGD
title: "Flaky gate: memory test races a replaced connection under -race"
type: bug
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - tooling
assignees: []
milestone: M4
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T19:42:47Z
updated_at: 2026-11-02T19:42:47Z
created_by:
  id: agent:claude-code/renki-m4-ci
  name: ""
updated_by:
  id: agent:claude-code/renki-m4-ci
  name: ""
extensions: {}
---

## Description

`TestPerceptionMemoryExcludesPreStateMenuAndReplacedConnections` in internal/memory/landmarks_test.go failed once on the first CI runs for TKT-01M6EFGJY8, under `go test -race` (run https://github.com/warricksothr/renki/actions/runs/41511564549):

    landmarks_test.go:149: unscoped frame accepted

It passed locally, and it passed on the other CI attempts.

### Likely cause
After `p2 := bridgetest.Connect(...)`, the test waits on `s.State() != nil`. That is already true from the first connection, so the wait can return before the session has registered the replacement connection and dropped the old observation. `assertNoObservation` then sees the old connection's observation. The wait should check something that only the second connection makes true.
