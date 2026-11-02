---
schema: 4
id: TKT-01M6EFKWGAF7P8V6W004YECKCF
title: Report a missing axe's recipe, station and material shortfall
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - controller
  - protocol
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

M5 goal 5.8 (docs/ROADMAP.md): when Tas has no usable axe, he says exactly what it would take to get one instead of only failing. Today a harvest without a suitable tool fails with `no_tool` and the required tool type and tier (plugin/Renki.Core/Tasks/Harvest.cs, plugin/Renki.Plugin/HarvestTask.cs), and gathering explains that a tool is missing. Nothing names a recipe, a station or the missing materials. This is the first slice of TKT-01M64T4WG (Tools: acquire, craft, repair and upgrade harvesting equipment).

### What to build

- **Plugin:** a read-only `recipe` query (a task or an event reply, documented in docs/BRIDGE-PROTOCOL.md with a fixture) that, for an item prefab, returns what the game itself says: the recipe's resources and amounts per quality level, the crafting station and minimum station level, and whether the bot's character knows the recipe. It reads `ObjectDB` and the player's known recipes and changes nothing.
- **Controller:** when gathering refuses for `no_tool`, pick the lowest-tier axe that would do, ask for its recipe, and compare against the pack and remembered chest contents near home (internal/home/containers.go, with their ages). Reply in one chat line within 10 s naming the axe, the station (and whether a matching station is remembered near home), and each missing material with the count short. Remembered contents are labelled as remembered, never as certain.
- A stored axe in a home chest is named first ("there is an AxeFlint in chest X, seen 5 min ago") rather than a recipe.

### Files

- `plugin/Renki.Core/Tasks/Harvest.cs` and `plugin/Renki.Plugin/HarvestTask.cs`: the `no_tool` reason
- `plugin/Renki.Core/Protocol/Messages.cs` and `internal/protocol/messages.go`: the new message
- `plugin/Renki.Plugin/GameTaskHost.cs`: where tasks are wired
- `internal/behaviour/wood.go` and `internal/behaviour/gather.go`: the chat reply for the gather command
- `internal/home/containers.go`: remembered chest contents
- `docs/BRIDGE-PROTOCOL.md`

### Verification

`just check plugin test-core` for the message shape and fixture; `just check controller test-packages ./internal/protocol ./internal/behaviour ./internal/chat` for: no axe with all materials, no axe with a shortfall, a stored axe preferred, a broken axe, no station remembered, and stale chest contents labelled. `just check ci`.

Live, holding the VM lease (TKT-01M6EF5DTG): with the axe stored away, `Tas, go chop wood` gets the reply within 10 s, and the recipe it names matches the game's crafting menu. Link the audit window.

## Acceptance criteria

- [ ] The plugin answers a recipe query with resources per quality, station, station level and whether the recipe is known, with a protocol fixture
- [ ] On no_tool, the owner gets one chat line within 10 s naming the axe, the station and each missing material with its count, in a controller test
- [ ] A stored axe in a remembered home chest is named before a recipe, with the snapshot's age
- [ ] One live run with the axe stored away gets the reply within 10 s and the recipe matches the game's crafting menu; audit window linked
