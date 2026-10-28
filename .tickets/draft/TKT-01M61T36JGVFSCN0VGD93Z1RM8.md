---
schema: 4
id: TKT-01M61T36JGVFSCN0VGD93Z1RM8
title: Define protocol v1 message types in Go with golden fixtures
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - protocol
  - controller
assignees: []
milestone: M1
parent: TKT-01M61T36B839WT7KEZ14TDMB8B
origin: null
dependencies:
  - TKT-01M61T36G7WJ6C1ZV2NXNM37KD
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-28T17:49:28Z
updated_at: 2026-10-28T17:49:28Z
created_by:
  id: agent:claude-code/valheim-setup
  name: Claude Code
updated_by:
  id: agent:claude-code/valheim-setup
  name: Claude Code
extensions: {}
---

## Description

Two languages implement one protocol, so the examples in docs/BRIDGE-PROTOCOL.md become JSON
fixtures that both sides test against. That turns drift into a failing test
instead of a confusing bug.

## Acceptance criteria

- [ ] A Go package holds types for every M1 message in sections 3, 5, 6 and 7
- [ ] Each example in the doc is a fixture under testdata/
- [ ] Go tests decode and re-encode every fixture
- [ ] Unknown fields are ignored and an unknown major version is rejected
