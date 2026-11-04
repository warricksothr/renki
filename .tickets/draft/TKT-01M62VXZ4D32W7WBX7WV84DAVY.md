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
updated_at: 2026-11-04T17:55:01Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:codex/portal-deploy
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

Live, holding the VM lease (TKT-01M6EF5DTG): `just vm task-check add_to '{"entity":"<home fire id>","item":"Wood","count":2}'` into a fire with room for 2 reports 2 added, and the next perception shows the fire's fuel up by at least 1.9 (fuel burns between samples). Link the audit window.

## Acceptance criteria

- [ ] Both respect wards and a Safety switch, and fail with the reasons 6.6 names
- [ ] An uncertain unit (no pack decrease or no fuel increase) stops the task without a retry, in a core test
- [ ] One live add_to of 2 Wood into a home fire that has room for 2 reports 2 added, and the next perception shows its fuel up by at least 1.9 (fuel burns between samples); audit window linked
- [ ] add_to puts fuel from the pack into a fire and reports how many
- [ ] take_from collects finished food from cooking stations

## Notes

**agent:claude-code/renki-plan** at 2026-11-02T16:15:14Z

Supersedes the earlier criteria on smelters, kilns, fermenters and beehives. The 2026-11-02 planning pass removed this ticket's dependency on TKT-01M62VXZ39 (Perception: fuel, lit, queued and finished for fires and machines), so its criteria now cover fires and cooking stations only, which need nothing new from perception. Machine support (fuel or ore into smelters and kilns, output from fermenters and beehives, furnaces fuelled only while queued) waits on the owner's decision on whether the base has those machines. If it does, file it as a follow-up that depends on TKT-01M62VXZ39.

**agent:codex/portal-deploy** at 2026-11-04T17:54:38Z

The owner rebuilt the under-floor fire with half walls directing smoke into the chimney and requested a native interaction check. Fresh observed target is hearth1274003559:2382 at[-203.66,30.37,-470.56],fuel1.9/20,lit=true; Tas already receives heat upstairs. One carried Wood was available for the test, but all meals had expired (25/25HP), so no prolonged exposed work was attempted.

Ordinary native t-30 reached the sheltered lower room at[-205.5,30.26,-466.9] around the revised structures. The camera shows the lit hearth through its opening and actual Fire/Resting/Rested effects, comfort5, with injurySeq0. The hearth remained freshly lit across observations. Evidence .checks/task-iqa8gedg.json,.checks/rebuilt-fire-room-map.json,.checks/camera-20261104T175332057774Z.jpg,.checks/rebuilt-fire-after-map.json.

Explicit normal interact t-31 targeting the freshly observed hearth was rejected not_allowed: the bot does not use that kind of thing. This is the existing InteractRule policy, not evidence that a half wall obstructs native use. Rejection happened before any fuel use or interaction approach/sight test; Wood remained1 and inventory unchanged (.checks/task-6asit0js.json,.checks/rebuilt-fire-before-inventory.json,.checks/rebuilt-fire-after-inventory.json). Do not report a successful refuel or verified interaction reach from room access, warmth or a camera view.

The immediate validation outcome is room access and short heat/rest without injury, but standalone interaction/refueling is unsupported in the installed runtime. The add_to task on this ticket is still draft; the owner asked to confirm current interaction, so no implementation was promoted, no policy was bypassed and no cooking task was used as a disguised fuel action. Once native refueling is implemented, use this exact hearth/opening as a live sight/reach test with ordinary access and one-unit pack/fuel receipts. Keep actual sleep, sustained ventilation and long-term smoke safety separate from this brief check.

**agent:codex/portal-deploy** at 2026-11-04T17:55:01Z

Correction to the preceding rebuilt-hearth note: only the carried Wood count was unchanged (1 before and after). The later inventory also contains4 Stone that were absent initially; no stone transfer/drop was ordered and their acquisition was not attributed. The entire inventory was therefore not unchanged. This does not change the refueling result: t-31 was rejected before native use, and no Wood was consumed. Retain both materials.
