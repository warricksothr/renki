---
schema: 4
id: TKT-01M66QF9EH8W2EG78AKKKF05PP
title: Report stable world identity and truthful terrain freshness
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - protocol
assignees: []
milestone: null
parent: TKT-01M66Q5SFWV99JCGJXVARVRJH7
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-30T15:39:51Z
updated_at: 2026-10-30T16:22:33Z
created_by:
  id: agent:codex/renki-atlas
  name: ""
updated_by:
  id: agent:codex/renki-atlas
  name: ""
extensions: {}
---

## Description

Extend the terrain bridge as proposed in docs/ATLAS.md. Report the game world UID as a decimal string, configured namespace/atlas epoch and compatibility metadata. Distinguish fresh per-sample measurements from cached carried values, and send bounded confirmations for genuinely remeasured unchanged values. Preserve the current sampling/frame budgets and older-producer session-only behavior. Do not generate unexplored world data from a seed.

## Acceptance criteria

- [x] Protocol fixtures preserve 64-bit world identity and distinguish worlds with the same display name and explicit epochs
- [x] Per-sample freshness/confirmation cannot be renewed by unloaded carried samples, unchanged resends or an adjacent changed corner
- [x] Focused checks cover partial tiles, identical fresh resampling, reconnect/reordering and bounded confirmation/sampling work

## Implementation plan

Add optional stable world UID to state plus a scoped terrain manifest (namespace/epoch, source/stream, game/generator/mod metadata). Emit per-corner monotonic measurement timestamps and producer capture time, including unchanged fresh observations while preserving carried-sample timestamps and original timestamps on resends. Session accepts only validated, ordered, live identity-matching batches and offers a bounded nonblocking observation drain for the atlas writer; legacy terrain remains session-only. Extend shared fixtures and core/session regressions before durable ingestion.

## Notes

**agent:codex/renki-atlas** at 2026-10-30T16:04:16Z

Implemented stable decimal-string world UID in state and a namespaced/epoch-scoped terrain manifest with source/stream and game/generator/plugin-version fingerprint. Added monotonic per-corner measurement times and capturedAt; unchanged fresh measurements confirm age, carried corners and cached resends keep original times. Legacy payloads remain session-only. Shared protocol fixture and core/session regressions passed within just check ci, including Windows builds and controller/plugin interop. Live deployment evidence follows after merge.

**agent:codex/renki-atlas** at 2026-10-30T16:22:33Z

Deployed 5f94daf after PR #79 (persistence), #80/#81 (installer lock/error handling). Full package checksum verification passed. Live DizzysHellheim world UID 827096491, namespace local/epoch 1: captured 841 canonical samples across 81 derived viewport tiles. All 841 retained with unchanged height/biome values and no older measurement times after controller reconnect, game shutdown plus offline panel restart, and a new game process rejoin. Persistent observer source stayed stable; the producer stream stayed the same across controller restart and changed across game restart as intended. Fresh unchanged confirmations advanced all 841 measurement times. No nearby actors were reported at initial capture; Tas rejoined at full current health 59/59. No terrain edits or harvesting were performed; localized changed-value/border/expiry behavior is proven by replay/storage checks, not a live landscaping edit.

Added read-only just vm atlas-check, which captures full evidence under .checks, prints bounded counts/version/health and compares canonical samples, ages and border consistency against a baseline. The helper checks live identity when in-world, supports saved/offline reads and never sends tasks or changes settings. Added regression checks for missing samples, old times and conflicting border evidence; all are in just check ci.

Evidence: .checks/atlas-live-5eh3p64k.json (initial), .checks/atlas-live-y811eqrm.json (controller reconnect), .checks/atlas-live-nz2zjdd8.json (offline panel), .checks/atlas-live-c0qhp5x5.json (new game process). CI: .checks/20261030T161948Z-JuSjpP.log. Final shutdown/offline confirmation follows. Portable export/import remains the fourth draft child; no map publication is enabled.

## Summary

Implemented stable decimal-string world UID in state and a namespaced/epoch-scoped terrain manifest with source/stream and game/generator/plugin-version fingerprint. Added monotonic per-corner measurement times and capturedAt; unchanged fresh measurements confirm age, carried corners and cached resends keep original times. Legacy payloads remain session-only. Shared protocol fixture and core/session regressions passed within just check ci, including Windows builds and controller/plugin interop.
