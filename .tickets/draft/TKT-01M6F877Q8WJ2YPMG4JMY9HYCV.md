---
schema: 4
id: TKT-01M6F877Q8WJ2YPMG4JMY9HYCV
title: "Behaviours: infer the anchor's activity from changes around them"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T23:06:28Z
updated_at: 2026-11-02T23:06:28Z
created_by:
  id: agent:claude-code/renki-lead
  name: ""
updated_by:
  id: agent:claude-code/renki-lead
  name: ""
extensions: {}
---

## Description

Part of the idle/passive decision (note on TKT-01M62VXYQB, 2026-11-02). Passive work should match what the anchor player is doing, so Tas needs to infer that from what changes around them. This is not yet copying their actions.

### What to build

- A pure tracker in `internal/behaviour`, fed every fresh frame, that turns changes near the anchor into an activity with a kind and a target:
  - trees, logs and stumps breaking near them (loot already records these breaks) mean felling;
  - rocks and ore deposits breaking mean mining;
  - pickables disappearing mean picking.
- Each activity carries the prefab and how recently it was seen. It decays once nothing has happened for a while.
- Expose it in the `View` for templates and the passive list, and in behaviour status for the panel view (TKT-01M6EYEMTX).

### Verification

- Tests with sequences of frames: felling beech, mining copper, picking raspberries, mixed activity, and decay.
- `just check ci`.

## Acceptance criteria

- [ ] A pure tracker turns breaks and vanished pickables near the anchor into an activity (felling, mining, picking) with its prefab, in tests
- [ ] The activity decays when nothing happens and is exposed in the View and behaviour status
