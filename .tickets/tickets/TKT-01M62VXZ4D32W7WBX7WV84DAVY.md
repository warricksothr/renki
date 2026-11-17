---
schema: 4
id: TKT-01M62VXZ4D32W7WBX7WV84DAVY
title: "Plugin: add fuel and ore to machines and take their output"
type: task
status: review
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
updated_at: 2026-11-04T18:14:31Z
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
- [x] An uncertain unit (no pack decrease or no fuel increase) stops the task without a retry, in a core test
- [x] One live add_to of 2 Wood into a home fire that has room for 2 reports 2 added, and the next perception shows its fuel up by at least 1.9 (fuel burns between samples); audit window linked
- [x] add_to puts fuel from the pack into a fire and reports how many
- [ ] take_from collects finished food from cooking stations

## Implementation plan

Implement the owner's selected fireplace slice: bounded add_to for native Fireplace fuel, including hearths, torches and braziers. Validate exact fuel metadata, normal ward/reach access and living safe footing; use ordinary UseItem once per unit and require pack decrease plus synchronized fuel increase before issuing another. Share the single-unit receipt with cooking while preserving cooking's empty-fire and heat requirements. Add default-off AllowStations plus session task permission, protocol validation and focused receipt/permission tests. Run focused checks, real plugin build, full CI and committed review; deploy under the VM lease and test the rebuilt half-wall hearth with real inventory/fuel evidence. Keep Tas and private video online outside the required restart. take_from and other machines remain outside this slice, with their acceptance criteria unticked.

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

**agent:codex/portal-deploy** at 2026-11-04T18:08:05Z

Implemented the owner-selected fireplace add_to slice, not take_from or general machines. Native task uses exact ordinary fuel UseItem, live ward/sight/reach and health/food/threat/fire/smoke guards, bounded interaction approaches and one-unit pack plus synchronized fuel receipts. Added default-off AllowStations and session permission. Shared FuelUnit also protects the existing empty cooking-fire path. Fueled-but-unlit fires stop further additions; fuel and lit state are reported separately. New just vm stations preserves existing Safety settings and inserts the known flag before the first upgraded launch.

Passed focused Fuel and CookingTests, controller protocol/config/settings/session/deployment tests, real plugin build and full just check ci (227s, .checks/20261104T180339Z-7nnsAy.log). Native deployment/test remains pending. Unit tests earn only uncertain-unit criterion2; output collection and both-task coverage remain unearned.

Predeployment resupply: first t-33 withdrawal was superseded by ordinary automatic flee from the nearby boar; t-36 reached safety34m away with injurySeq0. t-38 withdrew3 Sausages and3 BoarJerky, reporting missing CarrotSoup honestly; t-41 withdrew3 QueensJam; t-43 withdrew2 Wood from mapped barrel1:15599. t-44 returned upstairs to[-204.35,32.97,-467.56] with three active meals, full115.4HP and injurySeq0. Pack now2 each Sausages/BoarJerky/QueensJam,3 Wood and4 Stone. Evidence .checks/task-zet106oa.json,.checks/hearth-resupply-flee.json,.checks/task-55w4xxwr.json,.checks/task-kxuqqjfa.json,.checks/task-nm3uajxl.json,.checks/task-_2k3r8b3.json,.checks/hearth-predeploy-inventory.json. No harvesting or invented supplies.

**agent:codex/portal-deploy** at 2026-11-04T18:14:30Z

Deployed1c1d36c with just check vm install: candidate preflight and installed checksums/deployment receipt passed; durable settings/data retained (.checks/20261104T180836Z-KWYTZz.log). just vm stations read absent=false, inserted true while offline, then read true. Preserved existing allowedTasks and appended add_to before the new game session. just check vm online restored the panel/private OBS stack and Tas in78s (.checks/20261104T180939Z-ISCenW.log). Post-test /api/video confirms verified gameplay, scope1:DizzysHellheim:827096491.

Native new-session t-1 add_to(Stone,1) against hearth1274003559:2382 was rejected invalid_target before movement/consumption (.checks/task-xy4fy2om.json). t-2 reached the lower room at[-205.54,30.26,-466.87] using the north staircase/door (.checks/task-05kfvpse.json). t-3 add_to(Wood,2) succeeded through the owner's half-wall opening from[-205.54,30.26,-466.89], without an alternate approach or injury. Actual paired receipts: pack3->2 with fuel1.69052184->2.69052172; then pack2->1 with fuel2.69052172->3.69052172. The native task reported added2/2 Wood; fuel3.691/20,lit=True. Next perception confirms1.7->3.7/20, exceeding the required1.9 increase. Inventory confirms Wood3->1, Stone4 unchanged and meals retained. This earns criteria3/4. No smoke/fire injury was observed (injurySeq0); this short check does not establish sustained ventilation in every weather state.

Audit window2026-11-04T18:11:50Z..18:12:05Z: .checks/audit-window-VvrnUU.jsonl. Exact task request/receipt traces and final status: .checks/task-v5io_9bo.json. Before/after .checks/hearth-two-wood-{before,after}-{map,inventory}.json and after-status.json retain observations. Ordinary t-4 returned upstairs (.checks/task-zbsqagkt.json). Explicit t-5 closed the open home Wood Door1:14823; fresh perception confirms closed (.checks/task-e5g1xg9_.json,.checks/hearth-final-home-map.json). This manual closure does not earn automatic double-door restoration. Storage double doors remain observed open; no extra traversal was made to close them.

Final18:14Z Tas is idle/grounded/sheltered upstairs at[-202.99,32.97,-467.42],109.5/109.5HP,injurySeq0, three active meals plus2 carried each Sausages/BoarJerky/QueensJam,1 Wood and4 Stone. Bed1274003559:489 remains mine/owned, hearth lit3.7/20, private stream verified gameplay. Loot stays disabled; idleMinutes0,afterDeath stay,afterOutage false. Leave Tas online. The fireplace slice is delivered; criteria1/5 remain unticked because take_from is not implemented. Keep output collection separate from this owner's hearth request.

## Summary

Fireplace add_to delivered on PR179 and deployed1c1d36c. Native rebuilt-hearth test verified2 Wood consumed, fuel1.7->3.7/20 and lit=true through half-wall opening, without injury. Full CI and committed review passed. take_from remains unimplemented; criteria1/5 intentionally unearned.
