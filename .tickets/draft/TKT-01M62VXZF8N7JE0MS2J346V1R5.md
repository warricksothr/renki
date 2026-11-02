---
schema: 4
id: TKT-01M62VXZF8N7JE0MS2J346V1R5
title: "Controller: stock targets and a stock-low event from chest memory"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - chat
assignees: []
milestone: M6
parent: TKT-01M62VXZA33E8JR5BR2RJD6AC1
origin: null
dependencies:
  - TKT-01M62VXYWJHB5BC9VGCBM0PVXD
  - TKT-01M62VXYSN4RQV75HTA3BSD162
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:49Z
updated_at: 2026-11-02T16:15:35Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-plan
  name: ""
extensions: {}
---

## Description

Owners set targets, `<name> keep 50 wood`, per bot. From chest memory, the
controller sums the base's stock of each targeted item and fires
`stock_low` when it falls below the target. It only uses counts seen
recently enough, and says how old they are. The event can trigger a
behaviour instance directly (restock with gather) or wake the agent.

This is not on an M5 exit goal. It is the general form of what household food already does for food alone, and it is what lets the agent (M6) wake on low stock.

### What exists

- The engine already accepts `stock_low` as an event trigger, but nothing emits it (internal/behaviour/definition.go, docs/BEHAVIOURS.md "Scheduling and conditions").
- `when.stock` conditions test `container`, `prefab`, `minCount` and `maxAgeS` against exact chest snapshots, and fail on missing, old or future-dated ones (internal/behaviour/conditions.go).
- Household food has its own per-food targets and minimums, checked by inspecting chests (internal/goal/food_stock.go, docs/FOOD.md "Pantry refill thresholds"). Do not replace that; it needs fresh inspection, not memory.
- Remembered chest contents persist with ages (internal/home/containers.go).

### What to build

- Targets in the settings store, per bot and world: item or selector → count, with the chests that count (default: remembered chests within 32 m of home) and a maximum age (default 30 min).
- Chat for owners: `<name> keep 50 wood`, `<name> stop keeping wood`, `<name> stock` (each target with its count and the age of the oldest snapshot used).
- The controller recomputes on each new container snapshot and once a minute, and fires `stock_low` with the item, count, target and age once per crossing below target (with a 10 % hysteresis before it can fire again). Totals that rely on a snapshot older than the maximum age are unknown and fire nothing.

### Files

- `internal/behaviour/definition.go` and `internal/behaviour/engine.go`: events
- `internal/behaviour/conditions.go`: the existing stock condition
- `internal/home/containers.go`
- `internal/settings/settings.go`
- `internal/chat/brain.go`
- `docs/BEHAVIOURS.md`

### Verification

`just check controller test-packages ./internal/behaviour ./internal/home ./internal/settings ./internal/chat` with tests for: a target crossed fires once, hysteresis prevents a repeat, a stale snapshot makes the count unknown, an instance with `trigger.event: stock_low` starts, and a non-owner refused. `just check ci`.

## Acceptance criteria

- [ ] Owners set and clear stock targets from chat
- [ ] stock_low fires when remembered stock, seen recently enough, falls below a target
- [ ] An instance or the agent can act on stock_low

## Notes

**agent:claude-code/renki-plan** at 2026-11-02T16:15:35Z

Applied on 2026-11-02 by the planning pass: milestone set to M6, to match the exit goal in docs/ROADMAP.md this ticket serves. This supersedes any earlier note that only proposed the move. The owner can move it back.
