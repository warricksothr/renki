---
schema: 4
id: TKT-01M6EFHS5ES3WRCJA0D0QTF6SM
title: "Live: a trip home detours around a remembered danger zone"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - vm
  - live
  - movement
assignees: []
milestone: M3
parent: TKT-01M62N2557R401S3E6JECSAPN4
origin: null
dependencies:
  - TKT-01M6EFHEK3PB5CSHWBHBW0P7AX
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:55:19Z
updated_at: 2026-11-02T15:55:19Z
created_by:
  id: agent:claude-code/renki-plan-m3
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m3
  name: ""
extensions: {}
---

## Description

The live half of goal 3.4 in docs/ROADMAP.md: one trip home detours around a danger zone recorded earlier the same day.

### Outcome

On the deployed build that includes the danger detour for trips home, one supervised trip home whose straight line crosses a danger zone still scoring at least 1 takes the detour and arrives without injury. Do not injure the bot on purpose to create a zone: use a zone that a natural injury created (a combat trial, a gather trial or a work trip), on any day, as long as it still scores at least 1. If there is none, record that and try again after the next natural injury.

### Steps

1. Take the VM lease (see the VM lease ticket this depends on).
2. Read current zones with `just vm panel GET /api/map` (field `danger`) and pick one between a reachable place and home.
3. Move the bot to the far side with an ordinary order, then send it home from the panel (`just vm panel POST /api/act/home`).
4. Retain the audit window: `just vm audit-window SINCE UNTIL`. It must show the `danger detour` log line, the leg tasks and their success, and no `damaged` event inside the zone.
5. `just vm route-evidence` can summarise the path, if its input fits.

### Files

- `scripts/vm/vm.just` (`panel`, `audit-window`, `route-evidence`)
- `docs/MEMORY.md` for the zone fields

### Verification

Evidence: the audit window path and a map snapshot (`just vm map-snapshot`) showing the zone and the path, linked in a note here. No new code unless the live trip finds a defect, which gets its own ticket.

## Acceptance criteria

- [ ] One supervised trip home, on a build with the detour, crosses a zone created by a natural injury and still scoring at least 1, logs danger detour and arrives without a damaged event inside the zone
- [ ] The audit window and a map snapshot are linked in a note
