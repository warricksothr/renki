---
schema: 4
id: TKT-01M62N12C9EWXCD54WMBH08T4E
title: "Plugin: deposit into a chest (and withdraw)"
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - plugin
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies:
  - TKT-01M62N129K1RGB8VSJZ3SDAC7N
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-resume
  branch: t3/hauling-loop
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 33d5c9ee55c3da7ea78be1f4d0bacbee756d9fd6
  session: null
  claimed_at: 2026-10-29T17:37:29Z
  expires_at: null
archive: null
created_at: 2026-10-29T01:40:10Z
updated_at: 2026-10-29T18:07:33Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:codex/renki-resume
  name: ""
extensions: {}
---

## Description

`deposit` (7.2): walk to a container by entity id, open it as a player would, move the named items (by prefab, a count or "all") into it, and close it. `withdraw` is the same the other way. A full chest stores what fits and reports the rest; a chest in use by someone else or behind a ward fails with a reason the controller can say.

## Acceptance criteria

- [ ] deposit moves the named items into a chest by id and reports what it moved and what did not fit
- [ ] withdraw takes named items out
- [ ] A chest that is full, in use or warded fails with a reason, not silently

## Implementation plan

### Chest transfers for the hauling loop
Use the game's normal chest access request and response, preserving wards, privacy and use locks; wait for ownership and current contents before moving anything. Approach with line of sight, fail promptly for denied access, close/release on every completion, cancellation or death. Move stacks through Inventory.MoveItemToThis to preserve metadata and save notifications, topping up matching stacks first. Exclude equipped gear and enforce carry weight on withdrawal.

Add deposit/withdraw protocol arguments with named counts or all, structured moved/remaining counts, and a container snapshot reporting position/size/contents. Record focused transfer-planning tests and shared protocol fixtures. Verify a round trip with only Tas's test seeds in game, then partial/full/access failure cases as available. just check ci and just check review gate each landed change; live acceptance boxes require evidence.

## Notes

**agent:codex/renki-resume** at 2026-10-29T18:07:33Z

Implemented deposit/withdraw with normal chest access/ownership RPCs, current inventory load, metadata-aware game stack moves, exact moved/remaining snapshots and immediate pack reports. Equipped items stay protected; withdrawal respects weight. Locks release on completion/cancel/death; weak pending requests consume late RPCs without opening UI, and a missing response quarantines that chest until its reply or unload. Added authenticated exact POST /api/task plus just vm task for reproducible live checks through normal permissions and direct-order semantics. Shared Go/C# fixtures, malformed-count validation, conservation/partial/protection tests, connection/world cache tests and API authorization checks pass; plugin builds against installed game. Gathering is already true on VM. Next: full CI/review, deploy, round-trip only Tas test seeds, partial/full/in-use/ward checks as available. AC boxes remain unchecked pending live evidence.
