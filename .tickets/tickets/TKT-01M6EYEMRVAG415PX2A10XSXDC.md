---
schema: 4
id: TKT-01M6EYEMRVAG415PX2A10XSXDC
title: "Loot: reach grows while the anchor stays settled in a small area"
type: task
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - controller
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies:
  - TKT-01M6EYEC1M3CWJR3562ZXHJGYW
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:claude-code/renki-lead
  branch: t3/adaptive-reach
  worktree: /home/sothr/.t3/worktrees/renki/t3-c2254fa1
  commit: 9414f22bc122ee5c93810df425cc5a2091ae996e
  session: null
  claimed_at: 2026-11-02T21:59:32Z
  expires_at: null
archive: null
created_at: 2026-11-02T20:15:45Z
updated_at: 2026-11-02T22:03:40Z
created_by:
  id: agent:claude-code/renki-wave1
  name: ""
updated_by:
  id: agent:claude-code/renki-lead
  name: ""
extensions: {}
---

## Description

Loot's reach should grow the longer the anchor player stays put, instead of a fixed 30 m. A player who stops to fell trees or fight should get a wider sweep, and one who is walking through should get none. The owner asked for this on 2026-11-02.

### The model

- **Settle area.** When the anchor stops, a small circle forms around them (start: 6 m radius, centred on where they stopped).
- **Drift.** While they stay inside it, the centre follows their average position and the circle may grow a little (to about 10 m). Wandering inside the area does not reset it.
- **Leaving.** Leaving the area completely (past its edge plus a little hysteresis) resets it, and loot's reach shrinks back to its base.
- **Reach.** Loot's radius starts at a base (say 8 m) and grows with the time settled, up to a cap (say 30 m, under the template's 40 m validation limit). The curve should be simple and monotonic, for example linear over the first 2 minutes.

The numbers above are starting points, to be tuned with the panel view (TKT-01M6EYEMTXMJN0GSF2Y03CDB12, Panel: show loot's anchor, settle area, reach and drop decisions). Put them in `LootParams` with defaults, so an instance can change them.

### Shape

Keep the settle tracker a pure type in `internal/behaviour` (positions and times in, area and reach out), so tests drive it with sequences of positions. Loot holds one per anchor and feeds it every tick through the hook it already has for drop history.

### Files

- `internal/behaviour/loot.go` and a new `internal/behaviour/settle.go`, with tests
- `docs/BEHAVIOURS.md`

### Verification

- Tests on the tracker: standing still grows reach to the cap and stops there; drifting within the area keeps it; one step out past the edge keeps it, a full exit resets it; walking through never grows it.
- A loot test: a drop at 20 m is ignored on arrival and taken once the anchor has been settled long enough.
- `just check controller test-packages ./internal/behaviour`, then `just check ci`.

## Acceptance criteria

- [x] A pure settle tracker gives the settle area and loot reach from the anchor's positions over time, with defaults in LootParams
- [x] Reach grows while the anchor stays in the area, which may drift and grow a little, and resets when they leave it completely, covered by tracker tests
- [x] A loot test shows a far drop ignored on arrival and taken once the anchor has settled
- [ ] docs/BEHAVIOURS.md describes the model and its parameters; just check ci passes

## Implementation plan

A pure `Settle` tracker in `internal/behaviour/settle.go`, fed positions and times. Its first sighting opens an area: a circle of `settleRadius` centred on the player. While the player stays within the edge plus `settleSlack`, the centre follows the mean of their positions, and a step into that slack band widens the circle to their distance, up to `settleMaxRadius`. A step beyond the band, or 30 s unseen (`SettleGap`), opens a fresh area. `Settled(now)` is the time since the area opened.

`LootReach(settled, p)`: `baseRadius` until `settleDelayS` has passed (so a walker never gains reach), then linear to `radius` (now the cap, default 30) over `growS`.

The shared `Loot` keeps one tracker per (player, area shape). `Observe` feeds every live commander in sight with the default shape, and any shape an instance has asked for, so the settle time accrues while Tas follows and loot cannot run. `pick` measures from the anchor with the reach its tracker gives. Trackers are dropped on a world change and after 10 minutes unseen.

New LootParams fields, where 0 means the default: `baseRadius` 8, `growS` 120, `settleDelayS` 5, `settleRadius` 6, `settleMaxRadius` 10, `settleSlack` 2. Each is validated. Saved instances pick up the defaults.

Tests: tracker sequences (standing still, drifting, one step out, full exit, walking through, a gap) and the reach curve. A loot test: a drop at 20 m is ignored on arrival and taken after settling. Existing tests whose drops lie beyond 8 m are adjusted. Docs: BEHAVIOURS.md.
