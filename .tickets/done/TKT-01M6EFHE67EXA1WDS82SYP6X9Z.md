---
schema: 4
id: TKT-01M6EFHE67EXA1WDS82SYP6X9Z
title: "Controller: sum drops per cell and prefab with their trend"
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - controller
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:55:08Z
updated_at: 2026-11-02T19:28:29Z
created_by:
  id: agent:claude-code/renki-plan-m2
  name: ""
updated_by:
  id: agent:claude-code/renki-m2-drops
  name: ""
extensions: {}
---

## Description

Split out of TKT-01M62N12G9 (Controller: notice where owners work, and the drops piling up), whose second criterion is this. The deliver-when-full work needs to know when the pile near an owner is gone (goal 2.2: deliver 60 s after the pile is gone), and the voice and the panel benefit from "wood piling up near Nimah".

### What to build

- From perception over time, group drops (`protocol.ResourceDrop`) within the bot's perception radius by a 4 m cell and prefab, with the stack total per cell and its trend over the last 5 minutes (rising, steady, falling, gone).
- Keep it in memory only, scoped to the world and connection, and bounded. It is a live signal, not a durable record.
- Expose it to the behaviour `View` so a trigger can ask "is there an eligible pile within R of this owner, and since when not". Answer "unknown", not "gone", for any part of that circle outside the bot's perception radius, so the deliver-when-gone timer (TKT-01M6EFHE4E) never starts on ground the bot cannot see.

### Files

- `internal/memory/pickups.go` and `internal/home/pickups.go` (existing pickup memory; check what they already record before adding a new type)
- `internal/behaviour/engine.go` (`view`, `triggers`, where `drop_appeared` is computed today)
- `internal/session` for the perception snapshot

### Verification

- `just check controller test-packages ./internal/memory ./internal/behaviour`
- `just check ci`

## Acceptance criteria

- [x] Drops within the bot's perception radius are grouped per 4 m cell and prefab with a stack total and a 5-minute trend, in a test
- [x] The grouping is scoped to world and connection and bounded in size, in a test
- [x] The behaviour View can ask whether an eligible pile is near an owner and since when not, and answers unknown rather than gone for ground outside the perception radius, in a test

## Implementation plan

The existing pickup memory (`internal/memory/pickups.go`, `internal/home/pickups.go`) records measured gains the bot itself picked up, persisted for 14 days. It says nothing about drops lying on the ground, so this adds a new, memory-only type rather than extending it.

### Approach

- New package `internal/piles` (depends only on `protocol`): a `Tracker` that takes each fresh perception frame with the bot position and the welcome's perception bounds. It keeps a ring of frames (at most 600, last 5 minutes), each holding the drops grouped by 4 m cell (flat x/z) and prefab with stack totals and the frame's sight reach (radius, shortened to the farthest drop when the list is full, as `burden.forgetGone` does). A new connection or world resets it. Piles per frame are capped.
- `Snapshot` (immutable, shares frames) answers `Piles()`: each cell+prefab with its total and a 5-minute trend (rising, steady, falling, gone, or unknown when the cell is out of sight now), and `Near(pos, r, eligible)`: present (with total), absent since when, or unknown when any part of the circle is outside the current sight reach. "Since" walks back through frames that covered the circle with no eligible pile in it; a gap in perception or a frame not covering the circle ends the walk.
- `internal/behaviour`: `View.Piles` field; engine holds a tracker and feeds it in `view()` only from a fresh perception (same 5 s rule `triggers` uses). No change to `triggers` or `drop_appeared`.
- Tests in `internal/piles` and one engine test showing the View exposes it.

## Notes

**agent:claude-code/renki-m2-drops** at 2026-11-02T19:25:52Z

For the deliver-when-gone timer (TKT-01M6EFHE4E, Behaviours: deliver when full): call `v.Piles.Near(v.Now, ownerPos, R, eligible)`. It answers `present` (with the eligible total), `absent` with `Since` (the start of an unbroken run of frames that saw the whole circle empty, capped at the 5-minute window), or `unknown`. Start the 60 s timer from `Since` only on `absent`; `unknown` means part of the circle is beyond perception reach (radius, or the farthest listed drop when the list is full), the newest frame is more than 5 s old, or there is no welcome radius. Owner position has to come from `v.Perception.Characters` (players by name); the tracker does not resolve owners.

Pile membership in `Near` is by flat distance from the pile's stack-weighted centroid; coverage is 3D distance from the bot. Piles are keyed by the drop's `Prefab`, not `Yield.Prefab`, matching burden. Nothing is surfaced on the panel or voice yet.

**agent:claude-code/renki-m2-drops** at 2026-11-02T19:28:29Z

After review: perception carries no scan origin, so the engine pairs a frame only with a state at most 1 s newer and shrinks its reach by max(15 m/s, |vel|) x lag. Piles(now) now treats a newest frame older than 5 s as out of view (trend unknown). A scan origin in the perception message would remove the guesswork; that is a plugin protocol change, not done here.

## Summary

Landed as internal/piles (memory-only tracker, 4 m cells, 5-minute trends, bounded to 600 frames and 128 piles per frame, reset per connection and world) and View.Piles in internal/behaviour, fed from fresh perception in view(). engine.go gained a field and one call; triggers and drop_appeared are untouched.
