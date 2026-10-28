---
schema: 4
id: TKT-01M61T36G7WJ6C1ZV2NXNM37KD
title: Scaffold the Go controller module and justfile
type: task
status: done
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
updated_at: 2026-10-28T18:06:08Z
created_by:
  id: agent:claude-code/valheim-setup
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
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

- [x] `go.mod` and `cmd/controller` exist, and `just build` produces the binary
- [x] `just build-windows` cross-compiles for the VM
- [x] `just test` and `just vet` run clean
- [x] `go.mod` declares module `github.com/warricksothr/renki`

## Implementation plan

Module github.com/warricksothr/renki at go 1.25.0, no dependencies yet. cmd/controller/main.go is a stub that parses -version and logs that the server is not written yet (that is TKT-01M61T36K1); the version is stamped with -ldflags -X main.version from git describe. justfile modelled on git-ticket: the buildvcs worktree detection, build to ./controller (already gitignored), build-windows to dist/controller.exe with GOOS=windows GOARCH=amd64, test, test-race, vet, lint (gofmt + vet), fmt, tidy, ci, clean. Packages go under internal/ as later tickets need them.

## Notes

**agent:claude-code/renki-groom** at 2026-10-28T18:04:08Z

Grooming, 2026-10-28. There is no CI in this repository yet, so "recipes match what CI runs" has nothing to match. Agents work in linked worktrees here, so the build recipe needs the same -buildvcs=false detection the git-ticket justfile uses, or `go build` fails with "error obtaining VCS status".

## Summary

go.mod (github.com/warricksothr/renki, go 1.25.0), a stub cmd/controller with a stamped -version, and a justfile with build, build-windows (dist/controller.exe), test, test-race, vet, lint, fmt, tidy, ci and clean. The build recipes carry the git-ticket worktree -buildvcs detection. `just ci` is the gate a future CI should run.
