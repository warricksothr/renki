---
schema: 4
id: TKT-01M62VXZ39Z020DAFQARB68K5M
title: "Perception: fuel, lit, queued and finished for fires and machines"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - protocol
  - live
assignees: []
milestone: M5
parent: TKT-01M62VXZ2FWDX3D4Z6XX1MB5A6
origin: null
dependencies:
  - TKT-01M62BMD18K3SE1WXJ7FDN8EPT
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:48Z
updated_at: 2026-11-02T15:59:48Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-plan-m5
  name: ""
extensions: {}
---

## Description

Perception (6.2) lists fires, cooking stations and smelters. Part of this is done: fires report `fire: {fuel, fuelMax, lit}` and cooking stations report `cooking: {heated, supported, slots}` with each slot's item, state (`empty`, `cooking`, `done`, `burnt`, `unknown`) and elapsed time (docs/BRIDGE-PROTOCOL.md, structures; docs/COOKING.md). What is left:

- **Smelters, kilns, blast furnaces, charcoal kilns, the spinning wheel and the like** (`Smelter` components, kind `smelter`): the fuel and its maximum, what is queued and how much, how much output is waiting, and whether they are working.
- **Fermenters and beehives:** they are not reported at all today, because `Classify` in Perceiver.cs only knows doors, beds, portals, cooking and crafting stations, smelters, fires and containers. Add them (a new structure kind, or `station` with a `machine` block) with whether they are ready, and for a fermenter what it holds.

Read from the ZDO, as fires and doors are. The protocol doc and fixtures are updated on both sides. Scan time stays within budget.

This matters to M5 only if the base has these machines. Goals 5.1 and 5.2 (docs/ROADMAP.md) use fires and cooking stations, which are already reported. Whether smelters, kilns, fermenters and beehives are in M5 is an owner decision.

### Files

- `plugin/Renki.Plugin/Perceiver.cs`: `Classify` and the per-part `switch` where fire and cooking state are filled
- `plugin/Renki.Core/Protocol/Messages.cs`: `PerceivedStructure`, `FireStatus`
- `internal/protocol/messages.go`: the Go mirror (`FuelMax`, `StructureFire`)
- `plugin/Renki.Tests/FixtureTests.cs` and `plugin/Renki.Tests/Fixtures/`: protocol fixtures
- `docs/BRIDGE-PROTOCOL.md`: section 6.2 structures

### Verification

`just check plugin test-core FixtureTests` and `just check controller test-packages ./internal/protocol` for the new fields both ways. `just check ci`.

Live, holding the VM lease (TKT-01M6EF5DTG): a perception frame near home shows a real smelter's or beehive's state matching the game, and the perception `scan` trace's `avgMs` stays at or under 2 ms over 5 minutes at home. Check with `just vm audit-summary` over the window.

## Acceptance criteria

- [x] Fires report their fuel and whether they are lit
- [ ] Smelters and kilns report their fuel, their queue and whether they are working
- [ ] Cooking stations, fermenters and beehives report what is ready or burning
- [ ] 6.2 and its fixtures describe the new fields, and scans stay within 2 ms
- [ ] Fermenters and beehives appear in perception at all, with a fixture
