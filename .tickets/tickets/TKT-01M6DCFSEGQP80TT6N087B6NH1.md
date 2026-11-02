---
schema: 4
id: TKT-01M6DCFSEGQP80TT6N087B6NH1
title: "Behaviours: stow walks avoid hostiles; marker health baseline"
type: bug
status: ready
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
updated_at: 2026-11-02T15:57:05Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m3
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
- [ ] Marker food and cargo detect an injury after healing past their start health
- [ ] Tests for marker food and cargo cover heal-then-hit with InjurySeq nil; just check ci passes

## Notes

**agent:claude-code/renki-review** at 2026-11-02T05:48:47Z

Done in the review-fixes PR: stow walks now set AvoidHostiles from dry, since the protocol requires dry for avoidHostiles. TestDryStowWalksAvoidHostiles covers both walks. Plain collect stows are not dry and still do not avoid. Making collect dry is a separate decision. Still open: marker food/cargo health baseline.
