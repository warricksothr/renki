---
schema: 4
id: TKT-01M6DCFSCKJE592MCK0R2DDYQG
title: "Behaviours: long gather waits, repeat conditions and tick clock"
type: bug
status: done
status_reason: null
priority: high
due_on: null
labels:
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
created_at: 2026-11-02T05:42:34Z
updated_at: 2026-11-02T18:55:00Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-m4-repeat
  name: ""
extensions: {}
---

## Description

Timing problems in the behaviour engine, found in review. Items 1 and 3 are fixed (see the 2026-11-02 note). Item 2 is what is left.

### 1. Gather's 15 s and 20 s waits are cut off at 5 s (medium) — fixed

`internal/behaviour/engine.go` (Tick, the `ErrWait` branch) ended any run that had returned `ErrWait` for 5 s, while gather returned `ErrWait` for longer waits documented in docs/BEHAVIOURS.md: the work-site pursuer wait of 15 s (`gather_defense.go`) and both wildlife windows of 20 s (`gather_wildlife.go`). They now return `&Delay{Until: deadline}`, pinned by `TestEngineHonoursDelaysPastTheObservationWaitLimit`.

### 2. A `while`/`until` repeat condition cancels the run in progress (medium) — open

`engine.go`: `allowed()` checks the repeat condition, and Tick calls it on the active run every 250 ms. Take a `collect` with `repeat:{mode:"until", condition:{items:{Flint:20}}}`. It is stopped with "condition changed" the moment the 20th flint is picked up, before it stows, so the cargo stays in the pack and the run is not counted.

The intended semantics are already written down. docs/BEHAVIOURS.md (the repeat paragraph) says `while` and `until` "repeat after successful runs while that condition holds or until it holds", so repeat decides only whether to run again. Fix: check the repeat condition when picking a candidate (`candidate`) and when a run finishes (`finish`, which sets `pending`), and check the active run against `When` only. Splitting `allowed()` into a start check and a still-allowed check is one way.

### 3. Two clock reads per tick (low) — fixed

The engine judges a Delay by `v.Now`, the clock the runner decided on. Tick still reads `now` separately for `waitingSince`, which is harmless.

### Files

- `internal/behaviour/engine.go` (`allowed`, `candidate`, `finish`, Tick)
- `internal/behaviour/engine_test.go`, `internal/behaviour/delay_test.go`, `internal/behaviour/hauling_test.go`
- `docs/BEHAVIOURS.md`

### Verification

- A new engine-level test: a `collect` with an `until` condition that becomes true mid-run still stows, finishes as a success, and is not queued again.
- `just check controller test-packages ./internal/behaviour`, then `just check ci`.

## Acceptance criteria

- [x] Gather's 15 s pursuer and 20 s wildlife waits survive the engine for their documented length, proven by an engine-level test
- [x] A while/until repeat condition no longer cancels the run in progress
- [x] The engine uses one timestamp per tick
- [x] docs/BEHAVIOURS.md still matches: repeat decides only whether to run again

## Implementation plan

docs/BEHAVIOURS.md already settles it: while/until repeat after successful runs, so the repeat condition only decides whether to start (or queue) another run. Split allowed() into canStart() (enabled, when, once/maxRuns, repeat condition), used by candidate() and finish(), and stillAllowed() (enabled, when) for the active-run check in Tick. Add an engine test where an until condition (items) becomes true mid-run: the run is not stopped, finishes as a success, and is not queued again. Tighten the BEHAVIOURS.md sentence so it says the condition never interrupts a run in progress.

## Notes

**agent:claude-code/renki-review** at 2026-11-02T05:48:46Z

Done in the review-fixes PR: (1) gather's pursuer 15 s and wildlife 20 s waits now return Delay, with TestEngineHonoursDelaysPastTheObservationWaitLimit pinning the engine contract and the pursuer test pinning Until=+15 s. (3) The engine judges a Delay by v.Now, the clock the runner decided on. Partly: Tick still reads now separately for waitingSince, which is harmless. Still open: (2) the while/until repeat condition cancelling the run in progress. It needs a decision on intended semantics, so check docs/BEHAVIOURS.md before changing allowed().

**agent:claude-code/renki-plan** at 2026-11-02T16:15:35Z

Applied on 2026-11-02 by the planning pass: milestone set to M4, to match the exit goal in docs/ROADMAP.md this ticket serves. This supersedes any earlier note that only proposed the move. The owner can move it back.

**agent:claude-code/renki-m4-repeat** at 2026-11-02T18:55:00Z

allowed() is split: canStart() (enabled, when, once/maxRuns, repeat condition) gates candidate(), finish() and the idle trigger; stillAllowed() (enabled, when) is the only check on the active run. TestUntilConditionReachedMidRunLetsTheRunFinish (conditions_test.go) uses two_steps rather than collect, with until items Flint:20 reached after step one: the run is not stopped, finishes as completed, Runs=1 and is not queued again. It fails against the old check. BEHAVIOURS.md repeat paragraph now says the repeat condition never interrupts a run. No VM check made; none needed for engine timing.

## Summary

Repeat while/until conditions now decide only whether another run starts; an active run is stopped only by enabled/when. Engine test and BEHAVIOURS.md updated. All three items closed.
