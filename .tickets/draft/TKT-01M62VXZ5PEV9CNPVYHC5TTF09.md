---
schema: 4
id: TKT-01M62VXZ5PEV9CNPVYHC5TTF09
title: "Controller: fuel template: keep fires and furnaces going"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
assignees: []
milestone: M5
parent: TKT-01M62VXZ2FWDX3D4Z6XX1MB5A6
origin: null
dependencies:
  - TKT-01M62VXYSN4RQV75HTA3BSD162
  - TKT-01M62VXZ4D32W7WBX7WV84DAVY
  - TKT-01M62VXYWJHB5BC9VGCBM0PVXD
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:48Z
updated_at: 2026-11-02T16:15:14Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-plan
  name: ""
extensions: {}
---

## Description

A template on the behaviour engine. Parameters: which fires and machines
(those in the base by default), what to burn (wood for fires, resin for
torches, coal for furnaces), when (cold, night, someone near, a cooking
station in use over it, a furnace with something queued), and a reserve of
fuel that stoking never uses up.

Fuel comes from the bot's pack first, then the base chests (withdraw). A
furnace with nothing queued gets no coal.

This is M5 goal 5.1 (docs/ROADMAP.md): with the fuel instance on, fires in the base stay lit, measured over two 2 h home soaks.

### What the code gives you

- Perception already reports each fire's `fire: {fuel, fuelMax, lit}` and each cooking station's slots (docs/BRIDGE-PROTOCOL.md, structures). Smelter state needs TKT-01M62VXZ39 (Perception: machine state); fires do not.
- State has `time.night` and `effects` (status effects such as cold). The engine already emits `nightfall` and `idle_at_home` (internal/behaviour/engine.go) and supports `timer` and `idle` triggers (docs/BEHAVIOURS.md, "Scheduling and conditions").
- Withdrawing from a chest with matched receipts already exists (the `withdraw` task; see how internal/goal/food.go withdraws food). Remembered chest contents rank which chest to try (internal/home/containers.go), but only an opened chest proves stock.
- Putting fuel in needs `add_to` from TKT-01M62VXZ4D (Plugin: add fuel and ore to machines).

### What to build

A `fuel` template, run from an idle trigger at home and a timer while at home:

- Scope: fires within a radius of home (default 32 m, the radius household food uses) or an explicit list of IDs, and the floor they are on.
- Top up a fire when, in a fresh perception sample, it is below a threshold (default half of `fuelMax`) and it is night, Tas or an owner is cold, or a cooking slot within 3 m is `cooking` or `done`. Light a fire that is out under the same conditions.
- Fuel by fire type: Wood for campfires and hearths, Resin for torches and sconces, unless the parameters say otherwise.
- Take fuel from the pack, then withdraw from home chests, never below the reserve (default 10 of each fuel counted across fresh chest snapshots and the pack). When there is none, tell the owners once per hour per fuel type.
- Smelters and kilns only once TKT-01M62VXZ39 reports their queue: fuel only while something is queued.
- Bounded per run (at most 8 fires and 3 minutes), and survival, owner orders and the body lease come first as for every instance.

### Files

- `internal/behaviour/engine.go`, `internal/behaviour/definition.go`: register the template and its params
- `internal/behaviour/marker_food.go` and `internal/behaviour/marker_rest.go`: examples of idle-at-home templates with admission checks
- `internal/behaviour/idle.go`: idle triggers
- `internal/goal/food.go`: withdraw with matched receipts
- `internal/home/containers.go`: remembered chest contents
- `internal/protocol/messages.go`: `FireStatus`
- `docs/BEHAVIOURS.md`: a section for the template

### Verification

`just check controller test-packages ./internal/behaviour` with tests for: a low fire at night topped up from the pack; a fire left alone by day with no cooking and nobody cold; withdrawal stopping at the reserve; a fire out at night relit; the once-per-hour no-fuel message; a furnace with nothing queued left alone. `just check ci`.

Live evidence is the fires section of `just vm upkeep-report` in TKT-01M6EFM4G7 (VM: two 2 h base upkeep soaks): in-scope fires lit in at least 98 % of night samples, never at fuel 0, and at least one matched fuel receipt per soak.

## Acceptance criteria

- [ ] Fires in the base are kept lit when cold or at night, and when food is cooking over them
- [ ] Fuel comes from the pack, then the chests, and never below the reserve
- [ ] When there is no fuel, the owners are told once
- [ ] The tests listed under Verification pass in just check controller test-packages ./internal/behaviour, and docs/BEHAVIOURS.md documents the template

## Notes

**agent:claude-code/renki-plan** at 2026-11-02T16:15:14Z

Supersedes the earlier criteria on smelters, kilns, fermenters and beehives. The 2026-11-02 planning pass removed this ticket's dependency on TKT-01M62VXZ39 (Perception: fuel, lit, queued and finished for fires and machines), so its criteria now cover fires and cooking stations only, which need nothing new from perception. Machine support (fuel or ore into smelters and kilns, output from fermenters and beehives, furnaces fuelled only while queued) waits on the owner's decision on whether the base has those machines. If it does, file it as a follow-up that depends on TKT-01M62VXZ39.
