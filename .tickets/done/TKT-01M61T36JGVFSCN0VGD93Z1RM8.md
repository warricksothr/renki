---
schema: 4
id: TKT-01M61T36JGVFSCN0VGD93Z1RM8
title: Define protocol v1 message types in Go with golden fixtures
type: task
status: done
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
updated_at: 2026-10-28T18:09:30Z
created_by:
  id: agent:claude-code/valheim-setup
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

Two languages implement one protocol, so the examples in docs/BRIDGE-PROTOCOL.md become JSON
fixtures that both sides test against. That turns drift into a failing test
instead of a confusing bug.

## Acceptance criteria

- [x] A Go package holds types for every M1 message in sections 3, 5, 6 and 7
- [x] Each example in the doc is a fixture under testdata/
- [x] Go tests decode and re-encode every fixture
- [x] Unknown fields are ignored and an unknown major version is rejected

## Implementation plan

Package internal/protocol. Envelope (v, type, seq, t, id, re) embedded in one struct per message. `task` means a request from the controller and a status from the plugin, so Decode takes the sender and looks the type up in that side's table: an unsupported `v` returns *VersionError, an unknown type returns ErrUnknownType (the caller replies `error` with code bad_message), and unknown fields are ignored by encoding/json. Encode stamps v and type. Task args are a RawMessage with typed structs for the six M1 tasks; DecodeArgs checks required fields and fails with bad_args. `to` is a Target (position array or entity id), and follow.run is a RunMode (false, true or "auto"). Perception and inventory are typed as well, because the doc has examples for them and the ticket asks for every example, even though they are streamed from M2. Fixtures go in testdata/plugin/ and testdata/controller/, one file per message, so the C# tests can read the same tree. The doc examples are copied verbatim. Messages the doc only sketches (logout, bye, error, ping, pong, config, the other events and M1 tasks) get full fixtures. The test round-trips every fixture and compares the result as generic JSON, so a field the types drop fails the test. testdata/lenient/ holds extra-field cases, which must decode and then encode to the matching strict fixture. testdata/reject/ holds cases that must fail.

## Notes

**agent:claude-code/renki-groom** at 2026-10-28T18:09:30Z

For the C# side (TKT-01M61T36KK): fixtures are in internal/protocol/testdata, one frame per file, under plugin/ and controller/ by sender; testdata/README.md says what each directory means. Compare as JSON values, not bytes. Optional numeric fields use omitempty, so a zero is dropped on encode: event amount and hpFrac, inventory durability, perception stars. minTool is a pointer because tier 0 is real. If a zero turns out to matter for one of the others, make it a pointer the same way. The doc examples for logout, bye, error and config are fragments without v, seq and t; the fixtures add them.

## Summary

internal/protocol has types for every message in sections 5 to 7, including perception and inventory (they stream from M2). Decode takes the sender and looks the type up in that side's table, so `task` means a request from the controller and a status from the plugin; it returns *VersionError and ErrUnknownType for the two rejections. Encode stamps v and type. DecodeArgs gives typed, validated arguments for the six M1 tasks, with ErrBadArgs and ErrUnknownTask. There are 37 fixtures in testdata/ (every doc example verbatim, plus the sketched messages), 2 lenient and 7 reject. The tests round-trip every fixture as JSON values, so a field the types drop fails them.
