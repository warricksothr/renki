---
schema: 4
id: TKT-01M6EFKWJ63QYZ8M8BVG390CSN
title: "Plugin: craft task at a station with exact material receipts"
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
parent: TKT-01M64T4WGZA175B38A8GBQ9YD4
origin: null
dependencies:
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:56:28Z
updated_at: 2026-11-02T15:56:28Z
created_by:
  id: agent:claude-code/renki-plan-m5
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m5
  name: ""
extensions: {}
---

## Description

M5 goal 5.8 (docs/ROADMAP.md): Tas can craft a missing tool at a station, through the game's own crafting, with exact counts. No craft task exists today. The nearest precedent is `interact` with `repair:true` (plugin/Renki.Plugin/RepairTask.cs), which opens a crafting station through its normal interaction and presses the game's repair button with a per-click durability receipt. This is the second slice of TKT-01M64T4WG (Tools: acquire, craft, repair and upgrade harvesting equipment).

### What to build

A `craft` task: `{station, item, quality}`.

- Walk to and open the station as `repair` does (normal interaction, ward access, sight and reach, station level).
- Craft one item through the game's crafting path, so recipe knowledge, station type and level, materials and no-cost mode rules all come from the game.
- Succeed only with an exact receipt: each material's pack count before and after, the new item's prefab, quality and durability, and nothing else changed. Fail with `missing_item` (naming the material and the shortfall), `invalid_target` (wrong or too-low station), `not_allowed` (ward, or a Safety switch off) or `refused`.
- Close the station UI on completion or cancellation. Never retry an uncertain craft.
- A Safety switch for crafting, off by default, and a session permission like the other tasks.

The controller side (when to craft, the resource policy) stays on TKT-01M64T4WG.

### Files

- `plugin/Renki.Plugin/RepairTask.cs` and `plugin/Renki.Tests/RepairTests.cs`: the station interaction to reuse
- `plugin/Renki.Core/Tasks/Safety.cs`: the new switch
- `plugin/Renki.Core/Protocol/Messages.cs`, `internal/protocol/messages.go` and `docs/BRIDGE-PROTOCOL.md`: the task and its reasons
- `plugin/Renki.Plugin/GameTaskHost.cs`: wiring

### Verification

`just check plugin test-core RepairTests` plus new core tests for the receipt policy (exact counts, an extra change fails, a missing material names its shortfall). `just check controller test-packages ./internal/protocol`. `just check ci`.

Live, holding the VM lease (TKT-01M6EF5DTG): with the materials in the pack and the switch on, `just vm task-check craft` crafts one AxeFlint at the home workbench; the trace shows the before and after counts. Link the audit window.

## Acceptance criteria

- [ ] craft succeeds only with each material's before and after pack count and the new item's prefab, quality and durability
- [ ] Missing materials, a wrong or low station, a ward and the Safety switch fail with the reasons docs/BRIDGE-PROTOCOL.md names, in core tests
- [ ] An uncertain craft is never retried and the station UI is closed on completion or cancellation
- [ ] One live AxeFlint craft at the home workbench through just vm task-check craft shows exact counts; audit window linked
