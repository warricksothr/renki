---
schema: 4
id: TKT-01M6EFGK0R9CM18FEBJW68ZKWF
title: "Lint: staticcheck for Go, and ESLint or tsc for the panel"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - tooling
  - controller
assignees: []
milestone: M4
parent: TKT-01M6DCF2ABDDNG3QF369NDPBT5
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:54:40Z
updated_at: 2026-11-02T15:54:40Z
created_by:
  id: agent:claude-code/renki-plan-m4
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m4
  name: ""
extensions: {}
---

## Description

Goal 4.1 asks for real lint. `just controller lint` runs gofmt, `node --check` on the embedded app and `go vet`. `just panel lint` runs only `prettier --check` over about 7k lines, including the 3k-line `panel/src/controller.jsx`. Neither catches an undefined name or an unused variable. Split out of TKT-01M6DCFST1 (CI: run just ci automatically, require the plugin, real lint), item 3, Go and panel half.

### What to build

- Add `staticcheck` to `controller.just`'s `lint`, pinned to a version (a `go run honnef.co/go/tools/cmd/staticcheck@vX` call, or a tools module).
- Add ESLint with at least `no-undef` and `no-unused-vars`, or `tsc --checkJs --noEmit`, to `panel/justfile`'s `lint`, as a pinned dev dependency in `panel/package.json` and `panel/package-lock.json` (`just panel lock` refreshes the lock).
- Fix what they report, or suppress each finding inline with a stated reason. Several agents edit the panel and controller at once, so prefer small fixes and inline suppressions over sweeping rewrites.

### Files

- `controller.just` (`lint`)
- `panel/justfile` (`lint`), `panel/package.json`, `panel/package-lock.json`, a new ESLint or tsconfig file in `panel/`
- whatever Go and `panel/src` files the linters flag

### Verification

- `just check controller lint` and `just check panel lint` pass.
- Planting an unused variable in `panel/src` and an unused result in a Go file makes each fail (try it locally, do not commit it).
- `just check ci` passes.

## Acceptance criteria

- [ ] just controller lint runs a pinned staticcheck and passes
- [ ] just panel lint runs ESLint (no-undef, no-unused-vars) or tsc --checkJs and passes
- [ ] Every suppression carries a stated reason
