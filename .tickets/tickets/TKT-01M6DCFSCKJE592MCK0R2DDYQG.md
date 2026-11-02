---
schema: 4
id: TKT-01M6DCFSCKJE592MCK0R2DDYQG
title: "Behaviours: long gather waits, repeat conditions and tick clock"
type: bug
status: ready
status_reason: null
priority: high
due_on: null
labels:
  - controller
assignees: []
milestone: M2
parent: TKT-01M6DCF2ABDDNG3QF369NDPBT5
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T05:42:34Z
updated_at: 2026-11-02T05:48:47Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-review
  name: ""
extensions: {}
---

## Description

Timing problems in the behaviour engine, found in review.

### 1. Gather's 15 s and 20 s waits are cut off at 5 s (medium)

`internal/behaviour/engine.go` (Tick, the `ErrWait` branch) ends any run that has returned `ErrWait` for 5 s. Gather returns `ErrWait` for waits that docs/BEHAVIOURS.md documents as longer:

- the work-site pursuer wait of 15 s (`gather_defense.go:58-66`);
- both wildlife windows of 20 s (`gather_wildlife.go:17-27`, `:79-85`).

With `defend:true`, a Greyling targeting the bot from 25 m never gets its window: the run ends at 5 s with "fresh observation did not arrive within five seconds". The existing tests call the runner directly, so they never meet the engine's cap.

Fix: return `&Delay{Until: deadline}` for these waits. Delay is allowed up to 30 s. Add an engine-level test that runs past 5 s.

### 2. A `while`/`until` repeat condition cancels the run in progress (medium)

`engine.go`: `allowed()` checks the repeat condition, and Tick calls it on the active run every 250 ms. Take a `collect` with `repeat:{mode:"until", condition:{items:{Flint:20}}}`. It is stopped with "condition changed" the moment the 20th flint is picked up, before it stows, so the cargo stays in the pack and the run is not counted.

The docs say repeat only decides whether to run again. Fix: check the repeat condition only when picking a candidate and when a run finishes, and check the active run against `When` only.

### 3. Two clock reads per tick (low)

The engine reads `v.Now` in `e.view()` and then `now := e.o.Now()` separately. A runner's `Delay{Until}` that falls between the two reads is rejected as "invalid or expired bounded wait". This is rare, but it ends the run for no reason. Fix: take one timestamp per tick.

## Acceptance criteria

- [x] Gather's 15 s pursuer and 20 s wildlife waits survive the engine for their documented length, proven by an engine-level test
- [ ] A while/until repeat condition no longer cancels the run in progress
- [x] The engine uses one timestamp per tick

## Notes

**agent:claude-code/renki-review** at 2026-11-02T05:48:46Z

Done in the review-fixes PR: (1) gather's pursuer 15 s and wildlife 20 s waits now return Delay, with TestEngineHonoursDelaysPastTheObservationWaitLimit pinning the engine contract and the pursuer test pinning Until=+15 s. (3) The engine judges a Delay by v.Now, the clock the runner decided on. Partly: Tick still reads now separately for waitingSince, which is harmless. Still open: (2) the while/until repeat condition cancelling the run in progress. It needs a decision on intended semantics, so check docs/BEHAVIOURS.md before changing allowed().
