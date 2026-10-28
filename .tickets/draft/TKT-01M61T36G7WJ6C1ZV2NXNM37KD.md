---
schema: 4
id: TKT-01M61T36G7WJ6C1ZV2NXNM37KD
title: Scaffold the Go controller module and justfile
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - tooling
assignees: []
milestone: M1
parent: TKT-01M61T36B839WT7KEZ14TDMB8B
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-28T17:49:28Z
updated_at: 2026-10-28T17:56:38Z
created_by:
  id: agent:claude-code/valheim-setup
  name: Claude Code
updated_by:
  id: agent:claude-code/valheim-setup
  name: Claude Code
extensions: {}
---

## Description

Lay out the Go side the way the organisation's other Go repos do:
`cmd/controller`, packages beside it, `go 1.25.0`, and a justfile whose
recipes match what CI runs.

The module path is `github.com/warricksothr/renki`. The project is personal,
and its public mirror will live under github.com/warricksothr.

## Acceptance criteria

- [ ] `go.mod` and `cmd/controller` exist, and `just build` produces the binary
- [ ] `just build-windows` cross-compiles for the VM
- [ ] `just test` and `just vet` run clean
- [ ] `go.mod` declares module `github.com/warricksothr/renki`
