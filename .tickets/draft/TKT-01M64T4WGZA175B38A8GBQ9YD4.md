---
schema: 4
id: TKT-01M64T4WGZA175B38A8GBQ9YD4
title: "Tools: acquire, craft, repair and upgrade harvesting equipment"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - plugin
  - live
assignees: []
milestone: M5
parent: TKT-01M6864352N8E28YCGHA19AN90
origin: null
dependencies:
  - TKT-01M6EFKWGAF7P8V6W004YECKCF
  - TKT-01M6EFKWJ63QYZ8M8BVG390CSN
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T21:48:07Z
updated_at: 2026-11-02T16:28:08Z
created_by:
  id: agent:codex/renki-resume
  name: ""
updated_by:
  id: agent:claude-code/renki-plan
  name: ""
extensions: {}
---

## Description

After the collection-and-delivery loop and single-target harvesting, give Tas a tool-readiness plan for gathering jobs. The owner requested crafting when no suitable axe exists, reporting missing supplies, and upgrading when resources become available.

Prefer a suitable unbroken carried tool, then an owner-designated stored tool. Repair at an accessible appropriate station when needed. If none exists, plan a known recipe against live/remembered supplies, station type/level and game progression. Gather available prerequisites with supported jobs, fetch permitted materials, and report exact shortages or an unavailable station/unlock instead of promising an action the body cannot perform. Do not loop forever on unobtainable inputs or bypass the game's recipe/unlock requirements.

Craft and upgrade through game APIs with exact before/after item counts. Upgrades should use an explicit owner policy (permitted tool tier, materials, reserves and spending limits) rather than consume arbitrary shared stock because it is present. Resume the original gathering job only when the tool is ready. Owner stop, retreat, death and reconnect invalidate all remaining steps. Crafting is a follow-up to M2's supplied-tool benchmark, not a prerequisite for loose flint/stone collection.
### Where the code stands (2026-11-02)

- Harvest picks a carried tool by its actual damage type, tier and durability, and a missing or unsuitable one fails `no_tool` with the required type and tier (plugin/Renki.Core/Tasks/Harvest.cs, plugin/Renki.Plugin/HarvestTask.cs).
- `interact` with `repair:true` repairs eligible carried gear at a crafting station with a per-click durability receipt (plugin/Renki.Plugin/RepairTask.cs, docs/BRIDGE-PROTOCOL.md). `gather` with `returnHome` uses it for an axe below 90 % before leaving home (docs/BEHAVIOURS.md).
- Not built: naming a recipe, station or shortfall; fetching a stored tool on its own; crafting; upgrading.

### Split for M5 (goal 5.8, docs/ROADMAP.md)

- TKT-01M6EFKWGA (Report a missing axe's recipe, station and material shortfall): criterion 1 here.
- TKT-01M6EFKWJ6 (Plugin: craft task at a station with exact material receipts): the native half of criterion 3.
- What stays on this ticket, after those two: fetching an owner-designated stored tool from a home chest before crafting (criterion 2), the controller's craft decision under an explicit owner resource policy (criterion 3, the policy half), and resuming gathering with the new tool (criterion 4). Upgrading can wait for an owner policy; see the owner decisions on the M5 plan.

### Files

- `internal/behaviour/gather.go` (`gatherAxe`, `gatherSupplied`) and `internal/behaviour/wood.go`
- `internal/goal/food.go`: a model for withdrawing a named item with matched receipts
- `internal/settings/settings.go`: where a resource policy would live
- `plugin/Renki.Plugin/RepairTask.cs`

### Verification

`just check controller test-packages ./internal/behaviour ./internal/goal` with tests for: a stored axe withdrawn before crafting, crafting refused when it would spend below the policy's reserve, gathering resuming with the new tool, and an owner stop cancelling the remaining steps. `just check ci`.

Live, holding the VM lease (TKT-01M6EF5DTG): with no carried axe, materials in a home chest and crafting allowed by policy, `Tas, go chop wood` crafts an AxeFlint, logs exact counts and completes one `gather` trip. Link the audit window.

## Acceptance criteria

- [ ] Missing or broken tools produce an actionable recipe, station and material shortfall
- [ ] Suitable carried or designated stored tools and repairs are preferred over crafting replacements
- [ ] Crafting and upgrades obey game progression and an explicit resource policy
- [ ] The gathering job resumes with a usable tool; stop and survival interruptions cancel the plan
- [ ] One live run with no carried axe crafts an AxeFlint under the owner's policy and completes one gather trip; audit window linked

## Notes

**agent:codex/renki-resume** at 2026-10-29T23:06:49Z

Supplied-tool baseline established during the approved harvesting slice: owner placed a new flint axe in rag chest 1:30592. Withdrew exactly AxeFlint 1 and retained its other stock (Stone 1, Flint 3, ArmorRagsChest 1). Pack before work: quality-1 AxeFlint at durability 100/100 and BeechSeeds 15. Evidence .checks/audit-window-Nw78Tk.jsonl. New HarvestTool policy selects eligible carried resource damage, tier and positive durability rather than guessing from prefab name; broken/inappropriate tools report no_tool with required type/tier. This does not yet fetch stored tools automatically, repair, describe material shortages, craft or upgrade, so all tool-readiness acceptance boxes remain open. The supplied axe is benchmark equipment and is not authorization to spend unrelated shared stock on upgrades. Derive recipes/station requirements from live game data in the next slice; keep an explicit upgrade/reserve policy before spending.

**agent:codex/renki-milestones** at 2026-10-31T05:15:07Z

### Equipment maintenance within a production work cycle

Owner clarified that maintenance must cover armour as well as harvesting tools, and fit between verified cargo deposit/restocking and resuming the remembered work site. Prefer repair at an appropriate accessible station before replacement; verify fresh actual durability changes per item and report partial/failed repairs. Durability contributes to a host-derived work/return pressure alongside cargo capacity/value, food, danger and travel cost. See TKT-01M6864352N8E28YCGHA19AN90 (Pressure-driven work, storage and maintenance goals), left in draft as planned follow-up. This note expands design coverage without promoting this equipment ticket.
