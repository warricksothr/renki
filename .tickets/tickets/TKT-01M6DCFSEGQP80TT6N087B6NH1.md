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

Two small safety inconsistencies in behaviour runners.

### 1. Stow walks never avoid hostiles

`internal/behaviour/stow.go:207` and `:269` set `AvoidHostiles: r.p.MatchExisting`, but both walks run only when `!r.p.MatchExisting`, so the flag is always false. A gather with `return:true`, or a plain collect, walks from the work site to the chests without avoiding hostiles, while every move gather sends itself avoids them.

Fix: give `stowRun` an explicit `avoid` field, set it from gather and marker food, and default it to true when `dry` is set. Add a test asserting the order's `AvoidHostiles`.

### 2. Marker food and cargo compare health with the run's start

`internal/behaviour/marker_food.go:112,126` samples `r.health` once and never updates it. Cargo shares it through `guard` (`marker_cargo.go:224`). Without injury-sequence support (`v.InjurySeq == nil`), a run that starts at 80/100, heals to 100 and is then hit to 85 never registers the injury. Gather rolls its baseline forward on every step (`gather.go:236-239`).

Fix: roll the baseline forward on each `check`, keeping the meal-refresh exception.

## Acceptance criteria

- [x] Stow move orders avoid hostiles when the caller asks for it, covered by a test
- [ ] Marker food and cargo detect an injury after healing past their start health

## Notes

**agent:claude-code/renki-review** at 2026-11-02T05:48:47Z

Done in the review-fixes PR: stow walks now set AvoidHostiles from dry, since the protocol requires dry for avoidHostiles. TestDryStowWalksAvoidHostiles covers both walks. Plain collect stows are not dry and still do not avoid. Making collect dry is a separate decision. Still open: marker food/cargo health baseline.
