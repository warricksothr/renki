---
schema: 4
id: TKT-01M647SMRM3XXAQS7Y9CW2TKZ9
title: Retry raised goals from other approaches before giving up
type: bug
status: review
status_reason: null
priority: high
due_on: null
labels:
  - plugin
  - movement
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
created_at: 2026-10-29T16:27:24Z
updated_at: 2026-10-29T16:34:49Z
created_by:
  id: agent:codex/renki-resume
  name: ""
updated_by:
  id: agent:codex/renki-resume
  name: ""
extensions: {}
---

## Description

A move_to directly beneath home failed unreachable after 0.3 s. Try bounded, navmesh-validated repositioning and replan the original destination before declaring an off-level endpoint unreachable. Preserve water refusal and cancellation.

## Acceptance criteria

- [x] An off-level endpoint starts repositioning and retries rather than failing immediately
- [x] Recovery is bounded and rejects incomplete or unsafe detour paths
- [x] Regression coverage passes and the remaining in-game check is documented

## Implementation plan

Extract a testable bounded approach policy into Core. At a partial off-level endpoint, select a complete water-safe navmesh detour away from or around the target, walk it briefly, then retry the original goal. Cap attempts and total time; trace each attempt. Cover retry, success, rejected candidates and timeout off-game, and document a raised-home game check.

## Notes

**agent:codex/renki-resume** at 2026-10-29T16:33:32Z

Implemented bounded off-level recovery in ApproachRecovery and Walker: three approaches (away, then either side), six seconds per detour and thirty seconds overall, with complete-path and per-segment deep-water validation. Original destination is replanned after every detour. Resetting recovery cancels stale strafes. Five focused policy/path tests pass. just check ci passed, including the plugin build against configured game assemblies and 122 C# tests. The raised-home, cancellation and waterside game checks are written in docs/TESTING.md and have not been performed or deployed in this session.

**agent:codex/renki-resume** at 2026-10-29T16:34:49Z

Final just check ci also passed after log-capture hardening: plugin and controller builds succeeded with zero warnings/errors, all 122 C# tests passed, Go race tests passed and the ticket store was valid. Live-game recovery validation remains pending in review.

## Summary

Implemented in the working tree with policy tests and game-build validation. An off-level endpoint now triggers bounded repositioning and replanning before unreachable. docs/TESTING.md records the remaining live-game validation; no deployment performed.
