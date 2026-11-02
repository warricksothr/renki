---
schema: 4
id: TKT-01M6DCFSEGQP80TT6N087B6NH1
title: "Behaviours: stow walks avoid hostiles; marker health baseline"
type: bug
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - safety
assignees: []
milestone: M3
parent: TKT-01M6DCF2ABDDNG3QF369NDPBT5
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T05:42:34Z
updated_at: 2026-11-02T19:08:48Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-m3-marker
  name: ""
extensions: {}
---

## Description

Two small safety inconsistencies in behaviour runners. They are part of goal 3.5 in docs/ROADMAP.md (containment for the fights).

### 1. Stow walks never avoid hostiles (done)

`internal/behaviour/stow.go:207` and `:269` set `AvoidHostiles: r.p.MatchExisting`, but both walks run only when `!r.p.MatchExisting`, so the flag is always false. A gather with `return:true`, or a plain collect, walks from the work site to the chests without avoiding hostiles, while every move gather sends itself avoids them.

Fixed in the review-fixes PR (see the note): stow walks set `AvoidHostiles` from `dry`, covered by `TestDryStowWalksAvoidHostiles`. Plain collect stows are not dry and still do not avoid; making collect dry is a separate decision.

### 2. Marker food and cargo compare health with the run's start (open)

`internal/behaviour/marker_food.go` samples `r.health` once (line 126, when the marker is chosen) and checks it at line 112, never updating it. Cargo shares it through its `guard` (`markerFoodRun`, set around `internal/behaviour/marker_cargo.go:220-224`). Without injury-sequence support (`v.InjurySeq == nil`), a run that starts at 80/100, heals to 100 and is then hit to 85 never registers the injury. Gather rolls its baseline forward on every step (`internal/behaviour/gather.go:236-239`).

Fix: roll the baseline forward on each `check`, keeping the meal-refresh exception.

### Files

- `internal/behaviour/marker_food.go`, `internal/behaviour/marker_cargo.go`, `internal/behaviour/health.go` (`healthSample`)
- `internal/behaviour/marker_food_test.go`, `internal/behaviour/marker_cargo_test.go`

### Verification

- A test for food and one for cargo: start at 80, heal to 100, hit to 85, with `InjurySeq` nil; the run stops. A meal that raises max health does not count as an injury.
- `just check controller test-packages ./internal/behaviour`
- `just check ci`

## Acceptance criteria

- [x] Stow move orders avoid hostiles when the caller asks for it, covered by a test
- [x] Marker food and cargo detect an injury after healing past their start health
- [x] Tests for marker food and cargo cover heal-then-hit with InjurySeq nil; just check ci passes

## Implementation plan

Give the marker guards a baseline that ratchets up: healthSample.roll() keeps the peak health, clamped to the current maximum, and takes a sample as is only when lost() forgave it as a meal refresh under an injury sequence (shared mealRefresh helper). markerFoodRun.check (also cargo's guard) and markerRestRun.Check call it after each passing check. Tests: heal-then-hit for food, cargo and rest with InjurySeq nil; small losses add up against the peak; a rising maximum without a meal keeps the peak.

## Notes

**agent:claude-code/renki-review** at 2026-11-02T05:48:47Z

Done in the review-fixes PR: stow walks now set AvoidHostiles from dry, since the protocol requires dry for avoidHostiles. TestDryStowWalksAvoidHostiles covers both walks. Plain collect stows are not dry and still do not avoid. Making collect dry is a separate decision. Still open: marker food/cargo health baseline.

**agent:claude-code/renki-m3-marker** at 2026-11-02T18:55:16Z

Fixed: check() now sets r.health to the current sample after lost() passes. New tests TestMarkerFoodNoticesAHitAfterHealingPastItsStart and TestCargoNoticesAHitAfterHealingPastItsStart go 80/100 -> 100/100 -> 95/95 (food decay) -> 110 max with a new meal -> hit to 85, InjurySeq nil. Both fail on the old code ('a hit after healing past the start health was ignored') and pass now. just check ci passed (167 s). A first ci run failed only in internal/app TestSpeechMemoryAndDeterministicLocationQuestionsIncludingPanel (2 s timeout, machine loaded by parallel gates); it passed alone and is filed as draft TKT-01M6ESNXKF.

**agent:claude-code/renki-m3-marker** at 2026-11-02T19:07:21Z

Supersedes the earlier note's description of the fix. Review found that moving the baseline to every passing sample lets losses under lost()'s 0.15 tolerance add up step by step. The fix is now healthSample.roll(), a baseline that ratchets up (peak, clamped to the current maximum), sharing mealRefresh() with lost(). The marker rest guard had the same start-only baseline and now uses roll() too. New tests: TestMarkerFoodNoticesAHitAfterHealingPastItsStart, TestCargoNoticesAHitAfterHealingPastItsStart, TestRestNoticesAHitAfterHealingPastItsStart, TestMarkerFoodAddsUpSmallLossesAgainstThePeak, TestMarkerFoodKeepsThePeakWhenTheMaximumRisesWithoutAMeal. Each was checked to fail against the defect it covers. just check ci passed at 38288e3 (153 s). Not deployed; no live check needed for the criteria.

## Summary

Marker food, cargo and rest guards keep a baseline that ratchets up (healthSample.roll), so a hit after healing past the start health stops the run, and small losses add up against the peak. Covered by five behaviour tests; just ci and just review clean. Stow walks were fixed earlier in #156.
