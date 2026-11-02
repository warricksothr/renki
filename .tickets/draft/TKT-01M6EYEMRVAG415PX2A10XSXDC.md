---
schema: 4
id: TKT-01M6EYEMRVAG415PX2A10XSXDC
title: "Loot: reach grows while the anchor stays settled in a small area"
type: task
status: draft
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
claim: null
archive: null
created_at: 2026-11-02T20:15:45Z
updated_at: 2026-11-02T20:15:45Z
created_by:
  id: agent:claude-code/renki-wave1
  name: ""
updated_by:
  id: agent:claude-code/renki-wave1
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

- [ ] A pure settle tracker gives the settle area and loot reach from the anchor's positions over time, with defaults in LootParams
- [ ] Reach grows while the anchor stays in the area, which may drift and grow a little, and resets when they leave it completely, covered by tracker tests
- [ ] A loot test shows a far drop ignored on arrival and taken once the anchor has settled
- [ ] docs/BEHAVIOURS.md describes the model and its parameters; just check ci passes
