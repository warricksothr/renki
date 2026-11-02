---
schema: 4
id: TKT-01M62VXZ4D32W7WBX7WV84DAVY
title: "Plugin: add fuel and ore to machines and take their output"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - protocol
  - safety
  - live
assignees: []
milestone: M5
parent: TKT-01M62VXZ2FWDX3D4Z6XX1MB5A6
origin: null
dependencies:
  - TKT-01M62PRDRKEHTQ1WC12FQ3S11J
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
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

Tasks to work a station as a player does, through the game's own calls so wards still decide. M5 goal 5.1 (fires stay lit, docs/ROADMAP.md) needs the fire part; the rest serves the machine half of the base-upkeep epic.

### What exists

The only fuelling today is inside native cooking: a cold station may first use one carried `Wood` on its uniquely matched empty supporting fire, with wood, fuel and heat receipts and no retry on an uncertain result (plugin/Renki.Plugin/CookingFire.cs, plugin/Renki.Core/Tasks/CookingFuel.cs, docs/COOKING.md). It only fires when the fire is empty and only one wood. `interact` refuses fires as `not_allowed` and refuses a smelter as `invalid_target` (docs/BRIDGE-PROTOCOL.md, `interact`). Beehives can already be used through `interact` with `allowGather`.

### What to build

- `add_to {entity, item, count}`: put fuel from the pack into a fire (including hearths, torches and braziers), one item per game use, up to `count` or until the game reports it full. Generalise the `CookingFuel` receipt: each unit needs the pack decrease and the synchronized fuel increase; an uncertain unit stops the task without retrying. Report how many went in.
- `take_from {entity}`: take what is ready from a cooking station, with a pickup and inventory receipt as cooking's output path has. The cooking-station half overlaps TKT-01M62VXZ6Z (tend template), which already collects its own cooked output; reuse that receipt code rather than writing a second one.
- Both walk to the station first using the existing interaction approach, and fail with `invalid_target`, `inventory_full`, `missing_item`, `not_allowed` or `unreachable`.
- A Safety switch for using stations (off by default), and a session permission for each task.

Fires are what goal 5.1 measures, and they need nothing new from perception. Smelters, kilns, fermenters and beehives are out of scope until the owner decides whether the base has them; that follow-up depends on TKT-01M62VXZ39 (Perception: machine state), and this ticket no longer does. If the PR grows past a day, land `add_to` for fires alone and file `take_from` as a follow-up.

### Files

- `plugin/Renki.Core/Tasks/CookingFuel.cs` and `plugin/Renki.Plugin/CookingFire.cs`: the fuel receipt to generalise
- `plugin/Renki.Plugin/InteractTask.cs`, `plugin/Renki.Plugin/InteractionApproach.cs`, `plugin/Renki.Core/Tasks/InteractRule.cs`: approach and what may be used
- `plugin/Renki.Plugin/CookingDelivery.cs` and `plugin/Renki.Plugin/PickupObservation.cs`: output pickup receipts for `take_from`
- `plugin/Renki.Core/Tasks/Safety.cs`: the switch
- `plugin/Renki.Core/Protocol/Messages.cs`, `internal/protocol/messages.go`, `docs/BRIDGE-PROTOCOL.md`: tasks, args and reasons
- `plugin/Renki.Plugin/GameTaskHost.cs`: wiring

### Verification

`just check plugin test-core CookingTests` plus new core tests for the generalised receipt (a full fire stops, a missing fuel increase stops without retrying, each unit is counted). `just check controller test-packages ./internal/protocol`. `just check ci`.

Live, holding the VM lease (TKT-01M6EF5DTG): `just vm task-check add_to '{"entity":"<home fire id>","item":"Wood","count":2}'` at home reports 2 added, and the next perception shows the fire's fuel up by 2. Link the audit window.

## Acceptance criteria

- [ ] Both respect wards and a Safety switch, and fail with the reasons 6.6 names
- [ ] An uncertain unit (no pack decrease or no fuel or queue increase) stops the task without a retry, in a core test
- [ ] One live add_to of 2 Wood into a home fire reports 2 added and the next perception shows its fuel up by 2; audit window linked
- [ ] add_to puts fuel from the pack into a fire and reports how many
- [ ] take_from collects finished food from cooking stations

## Notes

**agent:claude-code/renki-plan** at 2026-11-02T16:15:14Z

Supersedes the earlier criteria on smelters, kilns, fermenters and beehives. The 2026-11-02 planning pass removed this ticket's dependency on TKT-01M62VXZ39 (Perception: fuel, lit, queued and finished for fires and machines), so its criteria now cover fires and cooking stations only, which need nothing new from perception. Machine support (fuel or ore into smelters and kilns, output from fermenters and beehives, furnaces fuelled only while queued) waits on the owner's decision on whether the base has those machines. If it does, file it as a follow-up that depends on TKT-01M62VXZ39.
