---
schema: 4
id: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
title: "M2: the bot hauls and gathers wood with its owner"
type: epic
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - plugin
  - controller
assignees: []
milestone: M2
parent: null
origin: null
dependencies:
  - TKT-01M62VXYQB6ZDBQE4E6ERGN83C
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T01:39:24Z
updated_at: 2026-10-29T13:23:51Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

The benchmark, in Nimah's words (2026-10-28): "I could be chopping wood at a location and the bot is picking up what is dropped and running it back to base to store. Then later I can tell it to go chop wood and because it knows I was collecting there it can go there but it can also decide based on where trees are."

M2 is the bot as a working helper: it carries, stores and gathers. It builds on M1's movement and chat and on perception (TKT-01M62BMD1), and it needs memory (TKT-01M62EP0P) to know where the base, the chests and the work sites are.

### The two scenes

1. **Hauling.** The owner chops at a spot. The bot stays near them, picks up the wood, resin and seeds that drop, and when it is full (or the pile is gone and the owner is done) it carries them home, puts them in a chest, and comes back.
2. **Gathering on its own.** Told "go chop wood", the bot picks where to go: the place it saw the owner working, or else the nearest trees it knows of. It equips an axe, fells, splits and breaks logs, collects the wood, and hauls it home as in scene 1.

### Out of scope here

No-chop zones (places the owner wants left alone) come later, as do combat (M3) and crafting. The bot must still survive the trips: drowning on straight-line paths (TKT-01M62G71Q) is fixed in M2, because every haul is a trip home.

## Acceptance criteria

- [ ] While the owner chops at a spot, the bot picks up the drops near them, carries them home when full and stores them in a chest, then comes back
- [ ] Told "go chop wood", the bot goes to where it saw the owner working, or else to the nearest trees it knows, and fells, splits and collects
- [ ] The bot keeps hauling across trips without drowning or getting lost, and says when it cannot (no chest, chest full, no axe)

## Implementation plan

Order of work, set 2026-10-29 when the owner asked for M2 and M3 to be carried through autonomously:

1. TKT-01M62G71 drowning on straight-line paths: every haul is a trip.
2. A panel command box that speaks as an owner, so each step can be checked in the game from the panel.
3. Plugin tasks: interact (TKT-01M62PRDR), deposit/withdraw (TKT-01M62N12C), equip (TKT-01M62N12D), harvest (TKT-01M62N12F).
4. Memory: chests and what they hold (TKT-01M62VXYW), work sites and drops (TKT-01M62N12G), on the per-world store (TKT-01M62EP0).
5. Behaviour engine (TKT-01M62VXYS) on the settings store.
6. Templates: loot and deliver-when-full (TKT-01M62N12H), stow (TKT-01M62VXYY).
7. Go chop wood (TKT-01M62N12J).
8. Chat for instances (TKT-01M62VXYV); the panel's instance half of TKT-01M628X13X.

Each: ci, vuoro review until clean, deploy, check on the VM.

## Notes

**agent:claude-code/renki-groom** at 2026-10-29T03:40:56Z

2026-10-28: hauling and gathering are now instances on the behaviour engine. The behaviours epic TKT-01M62VXYQB6 ('Behaviours: templates and instances the bot runs on its own') carries the engine, settings store, chest memory and stow. Base upkeep (TKT-01M62VXZ2FWD) and the agent's tools and events (TKT-01M62VXZA33E) are filed for after M2.
