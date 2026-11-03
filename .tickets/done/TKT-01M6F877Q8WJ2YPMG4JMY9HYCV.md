---
schema: 4
id: TKT-01M6F877Q8WJ2YPMG4JMY9HYCV
title: "Behaviours: infer the anchor's activity from changes around them"
type: task
status: done
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
updated_at: 2026-11-03T18:09:33Z
created_by:
  id: agent:claude-code/renki-lead
  name: ""
updated_by:
  id: agent:codex/portal-deploy
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

- [x] A pure tracker turns breaks and vanished pickables near the anchor into an activity (felling, mining, picking) with its prefab, in tests
- [x] The activity decays when nothing happens and is exposed in the View and behaviour status

## Implementation plan

Add a pure bounded activity tracker in internal/behaviour, reset on session/world changes, fed only ordered fresh perception frames. Infer recent felling, mining and picking from disappeared resources near a live authorized anchor, retain prefab/position/time, resolve mixed evidence deterministically and decay old observations. Expose immutable activity in View and behaviour status without dispatching work. Cover frame sequences, decay, duplicates, stale/gapped frames, anchor changes and scope resets with just check controller test-packages, then just check ci.

## Notes

**agent:codex/portal-deploy** at 2026-11-03T18:05:28Z

### Activity tracking implemented

The owner selected this draft along with preflight and the Loot panel. Implemented on t3/renki-runtime-loot-batch without enabling gathering or sending new work. ActivityTracker turns fresh ordered frame changes near a live authorized anchor into felling/mining/picking with player, target ID, prefab, position, observation time and age. It decays after 30 seconds. Simultaneous changes choose nearest, then kind/ID. Session/world/UID/anchor changes, stale/old/duplicate/gapped scans, jumps, capped resource kinds and scan-edge disappearance cannot create new evidence. These remain inferences: the current protocol cannot prove a complete scan or identify who caused a disappearance.

View.Activity feeds templates, and behaviour status exposes an immutable activity snapshot, including the passive list. Sequence tests cover beech trees/logs/stumps, copper and raspberries, mixed activity, duplicates, scope changes and decay. An engine integration check proves templates and disabled Loot status receive the activity, callers cannot mutate retained evidence, and disabled Loot does not run. Focused and race checks passed through just check controller test-packages; .checks/20261103T180323Z-1ZsXHp.log retains the race suite. Full just check ci and committed review remain the delivery gate. docs/BEHAVIOURS.md documents the inference limits.

**agent:codex/portal-deploy** at 2026-11-03T18:09:32Z

The complete just check ci gate passed (225 s), retained in .checks/20261103T180530Z-ShHgUG.log. This covers the final activity/diagnostics source, panel unit/UI suites, controller race tests/native builds, plugin/interop and deployment entry-point checks. Committing the batch before committed review; production remains unchanged and the game offline.

## Summary

Implemented and documented bounded inferred anchor activity in View and behaviour status; sequence, scope, decay, integration/race and full CI checks passed. No passive gathering was enabled.
