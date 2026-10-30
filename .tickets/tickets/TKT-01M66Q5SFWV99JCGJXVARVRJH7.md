---
schema: 4
id: TKT-01M66Q5SFWV99JCGJXVARVRJH7
title: Durable versioned world atlas and portable mapping data
type: epic
status: ready
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - plugin
  - protocol
  - docs
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: children
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-30T15:34:39Z
updated_at: 2026-10-30T16:29:09Z
created_by:
  id: agent:codex/renki-atlas
  name: ""
updated_by:
  id: agent:codex/renki-atlas
  name: ""
extensions: {}
---

## Description

Design durable mapping that survives controller/plugin runs, merges newer terrain observations into an accumulated atlas, keeps changed versions for a bounded period, and supports portable mapping exchange later. Current terrain is a 256-tile session cache; other memories persist separately. See docs/ATLAS.md for evidence, storage decisions and rollout.

Proposed canonical store: a separate local SQLite atlas with fixed Cartesian sample/tile keys, current materialized state and a bounded change journal. Track stable world identity/epoch, source/provenance, sample freshness, resolution and content hashes. New unknown samples never erase known measurements; unchanged fresh observations renew confirmation without creating history. Separate terrain/biomes, objects/resources, transient conditions, coverage and route evidence with different retention policies.

Sharing remains a future explicit export/import capability: prefer GeoPackage/local coordinate metadata for GIS exchange, numeric raster snapshots where useful, and read-only tile archives for publication. Do not present local game coordinates as WGS84 GeoJSON. No publishing or runtime storage change is part of this design ticket.

## Acceptance criteria

- [x] Mapping survives restarts under stable world/epoch identity and updates current state without losing unsampled data
- [x] Changed observations retain bounded history; freshness, logical expiry, quotas and physical storage maintenance are explicit and tested
- [x] Panel distinguishes current, remembered/stale, unknown and imported evidence without claiming exploration or navigability
- [ ] Portable mapping representation defines local coordinates, provenance, compatibility and selected shareable layers; exchange remains explicit

## Implementation plan

PR #79 merged as 1561eb5. Initial installation found Valheim stopped but the panel controller still held controller.exe open, so SCP could not replace it. Add reusable just vm panel-stop with bounded process wait, and make the full installer call it even when Valheim is already offline. Then land the tooling fix, install, collect a live measured atlas and verify it survives controller reconnect and offline panel viewing. Retain full API evidence in .checks with bounded summaries.

## Notes

**agent:codex/renki-atlas** at 2026-10-30T16:04:17Z

The first three children are implemented: identity/freshness, durable terrain storage and saved-map viewing. The explicit portable export/import child remains draft; no shared publication or imported evidence is enabled. docs/ATLAS.md now distinguishes implemented runtime from the broader format/layer proposal. Local CI, interop, Windows builds and browser fixture checks passed; live deployment proof follows.

**agent:codex/renki-atlas** at 2026-10-30T16:12:05Z

Supersedes the earlier panel-stop success claim in the deployment plan/PR #80 validation: the inline SSH PowerShell command returned exit 0 despite execution-policy errors, so it did not stop the controller. Moved the step into stop-panel.ps1, invoked with the standard -NoProfile -ExecutionPolicy Bypass path and terminating errors. A real controller-start/stop check now reports exit verified, followed by just vm status confirming both controller and game stopped. Also handle brief stale process records after Wait-Process with a bounded 10-second poll. Full installer and atlas live checks continue below.

**agent:codex/renki-atlas** at 2026-10-30T16:22:33Z

Deployed 5f94daf after PR #79 (persistence), #80/#81 (installer lock/error handling). Full package checksum verification passed. Live DizzysHellheim world UID 827096491, namespace local/epoch 1: captured 841 canonical samples across 81 derived viewport tiles. All 841 retained with unchanged height/biome values and no older measurement times after controller reconnect, game shutdown plus offline panel restart, and a new game process rejoin. Persistent observer source stayed stable; the producer stream stayed the same across controller restart and changed across game restart as intended. Fresh unchanged confirmations advanced all 841 measurement times. No nearby actors were reported at initial capture; Tas rejoined at full current health 59/59. No terrain edits or harvesting were performed; localized changed-value/border/expiry behavior is proven by replay/storage checks, not a live landscaping edit.

Added read-only just vm atlas-check, which captures full evidence under .checks, prints bounded counts/version/health and compares canonical samples, ages and border consistency against a baseline. The helper checks live identity when in-world, supports saved/offline reads and never sends tasks or changes settings. Added regression checks for missing samples, old times and conflicting border evidence; all are in just check ci.

Evidence: .checks/atlas-live-5eh3p64k.json (initial), .checks/atlas-live-y811eqrm.json (controller reconnect), .checks/atlas-live-nz2zjdd8.json (offline panel), .checks/atlas-live-c0qhp5x5.json (new game process). CI: .checks/20261030T161948Z-JuSjpP.log. Final shutdown/offline confirmation follows. Portable export/import remains the fourth draft child; no map publication is enabled.

**agent:codex/renki-atlas** at 2026-10-30T16:29:09Z

Final native shutdown/offline proof retained all 841 samples: .checks/atlas-live-yhj_ei_8.json. Tas is logged out, Valheim stopped, and the controller/panel is online. The user additionally requested automatic offline terrain/fog, so the page now defaults to Follow Tas: it preserves the last live atlas when he leaves, loads the newest saved atlas on a fresh offline page, dims known terrain and keeps unknown cells dark. Rejoining automatically restores live mode unless the user pinned a saved world. Live actors/objects remain cleared for saved views; request generations reject delayed live responses.

Browser checks using the recorded native offline atlas passed initial offline fallback (81 derived tiles), live return, explicit saved-world pinning and a delayed live response across logout, with zero browser errors. Added a rendering regression proving fog dims known cells and never fills unknown cells. just check ci passed with the new read-only VM evidence helper/oracle checks (log .checks/20261030T162752Z-ID5jSz.log). The final controller-only UI deployment follows this commit; portable sharing remains a separate draft child.
