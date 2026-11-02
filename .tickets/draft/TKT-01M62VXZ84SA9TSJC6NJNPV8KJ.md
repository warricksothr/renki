---
schema: 4
id: TKT-01M62VXZ84SA9TSJC6NJNPV8KJ
title: "Controller: assign every item to chests, with a loose-items chest"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - chat
assignees: []
milestone: M5
parent: TKT-01M62VXZ2FWDX3D4Z6XX1MB5A6
origin: null
dependencies:
  - TKT-01M62VXYY6F094WXQS3FBQ57RF
  - TKT-01M62VXYV33R1WXRHMRZ6GZMRH
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

A table per bot, item or group → an ordered list of chests, which generates
the `stow` instances. A chest marked loose takes anything without an
assignment. With none marked, unassigned items are reported instead.
Owners edit it from chat and the panel: `<name> wood goes in chest:wood-1`.

Sorting is moving items already in chests to where they are assigned. It
runs only while the bot is idle at home and nobody is using the chests, and
any order interrupts it. A chest marked "don't touch" is never moved from.

This is the core of M5 goal 5.3 (docs/ROADMAP.md): assigned items reach their chests. To keep it to one pull request, this ticket covers the table, the chat command, and using the table when Tas unloads. Two follow-ups carry the rest: TKT-01M6EFKDSV (Controller: sort misplaced items into assigned chests while idle) and TKT-01M6EFKDQQ (Panel: edit chest assignments, loose and don't-touch chests).

### What exists

- `stow` moves all of some items to an ordered list of up to 16 chests, with selectors `wood`, `ore`, `food` and `*`, `keep`, matched receipts and a storage-needed shortfall (internal/behaviour/stow.go, docs/BEHAVIOURS.md "Bounded collection and stowing").
- `marker_cargo` already unloads surplus cargo at a storage marker, preferring chests that hold the item, then chests near where it was picked up (internal/behaviour/marker_cargo.go). It has no owner table.
- The settings store keeps behaviour instances with who changed them (internal/settings/instances.go, internal/settings/settings.go).
- Remembered chest contents and positions persist per world (internal/home/containers.go).

### What to build

- An assignment table in the settings store, per bot and per world: entries of item or selector → ordered chest IDs; one optional loose chest; a set of don't-touch chests. Validation: chest IDs must be remembered containers in that world; at most 64 entries and 16 chests per entry.
- `GET` and `POST /api/assignments` (panel auth, `X-Renki-Act` on writes, source recorded) for the panel ticket to use.
- Chat, owners only, one-line replies: `<name> wood goes in <chest>` (the chest named by a waypoint or by the chest Tas or the speaker stands nearest within 3 m), `<name> loose chest is <chest>`, `<name> don't touch <chest>`, `<name> assignments`, `<name> forget wood`. Non-owners are refused. Share the command grammar with TKT-01M62VXYV3 (Chat: behaviour instances).
- `marker_cargo` and `stow` with `to` omitted consult the table first: an assigned item goes only to its chests (then is reported as a shortfall if they are full), an unassigned one goes to the loose chest, or is reported when there is none. Existing behaviour is unchanged when the table is empty.

### Files

- `internal/settings/settings.go` and `internal/settings/instances.go`: the store
- `internal/behaviour/stow.go` and `internal/behaviour/marker_cargo.go`: where chests are chosen
- `internal/app/behaviours.go`: an example of an authenticated endpoint with `X-Renki-Act`
- `internal/chat/brain.go`: `parse` and `run`
- `internal/home/containers.go` and `internal/home/waypoints.go`: chests and named places
- `docs/BEHAVIOURS.md`: a section on assignments

### Verification

`just check controller test-packages ./internal/settings ./internal/behaviour ./internal/chat ./internal/app` with tests for: an assigned item going only to its chests; a full assigned chest reported with item and count, not spilled into another chest; an unassigned item to the loose chest; no loose chest reported; a non-owner refused; a chest ID from another world rejected; an empty table leaving `marker_cargo` as it is. `just check ci`.

Live evidence is the assignments section of `just vm upkeep-report` in TKT-01M6EFM4G7 (VM: two 2 h base upkeep soaks).

## Acceptance criteria

- [ ] Unassigned items go to the loose chest, or are reported when there is none
- [ ] Owners assign items or groups to chests, set the loose chest and mark don't-touch chests from chat; non-owners are refused
- [ ] The table is kept per bot and world in the settings store and served by GET and POST /api/assignments with the source recorded
- [ ] An assigned item goes only to its chests; a full assigned chest is reported with item and count rather than spilled elsewhere
- [ ] With an empty table, marker_cargo and stow behave as before, in a test
