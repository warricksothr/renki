---
schema: 4
id: TKT-01M6CAHE9YS9TNSJ6H2H8S862G
title: Migrate Tas to the host Valheim Plus release with defaults
type: task
status: blocked
status_reason: Deployment and smoke checks passed; only exact host game/BepInEx build inventory remains unavailable without server-side version evidence (public query timed out).
priority: high
due_on: null
labels:
  - vm
  - plugin
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-01T19:49:16Z
updated_at: 2026-11-02T16:15:34Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:claude-code/renki-plan
  name: ""
extensions: {}
---

## Description

The host is migrating to Grantapher's maintained Valheim Plus fork (Nexus mod 2323). The user confirmed fresh default settings, likely retained long term. The exact host release remains unknown. Tas was stopped with `just vm stop` at the user's request and must remain offline during the server migration.

### Verified baseline
Blight's last startup reported Valheim 1.0.21 (network 41), BepInExPack Valheim 5.4.2360 and underlying BepInEx 5.4.23.6. The read-only DLL inventory found only Renki.Core and Renki.Plugin under BepInEx/plugins. FileVersion on valheim.exe is the Unity version, not the Valheim release; use the startup's Valheim Version line.

The official latest release inspected on 2026-11-01 was Valheim Plus 10.4.0.0 (Thunderstore 10.4.0), published 2026-10-25. Its documented supported combination is Valheim 1.0.20 (n-41) and BepInExPack_Valheim 5.4.2351; other combinations are explicitly unsupported. This differs from Blight. It is not evidence of a runtime failure, but compatibility cannot be claimed from the network version alone. Do not silently downgrade the working installation or install an assumed host version.

### Default settings audit
At tag 10.4.0.0, Inventory, Items, Player, CraftFromChest, AutoStack, Pickable, Smelter, Kiln and FireSource sections default disabled. Machine autoDeposit and autoFuel also default false. Vanilla inventory sizes, stack sizes, carrying capacity and manual item transfers therefore remain the intended baseline. Server defaults enabled with enforceMod=true and serverSyncsConfig=true: a matching V+ client is required and server settings are authoritative. Confirm these defaults against the actual selected release/config, rather than copying assumptions across versions.

Renki already reads inventory dimensions, maximum stacks, durability, capacity and transfer weights from the running game. No behavior change is justified solely by the confirmed defaults. Source review and client/server smoke checks remain necessary for the chosen runtime; defaults do not prove patch coexistence.

### Migration work remaining
Obtain the exact host V+ and Valheim versions. Preserve Renki DLLs, configs, controller memory, markers and saves; install the matching V+ client into the actual game folder because current VM launch scripts bypass r2modman profiles. Back up any existing V+ config; a bundled or reused non-default config is not a fresh default. Release 10.4.0.0 generates its config on first startup. Inspect effective server-synced settings and startup patch errors after joining. Then check state/inventory reporting, movement/home routing, a measured chest transfer and cooking/harvest receipts using existing `just vm` tooling. Coordinate restart after the host finishes migrating. Leave acceptance boxes open until observed.

### Sources and retained evidence
https://github.com/Grantapher/ValheimPlus/releases/tag/10.4.0.0
https://github.com/Grantapher/ValheimPlus/blob/10.4.0.0/INSTALL.md
https://github.com/Grantapher/ValheimPlus/tree/10.4.0.0/ValheimPlus/Configurations/Sections
Ignored local evidence: .checks/modpack-2323/release.json, source/Grantapher-ValheimPlus-1c3a7cf and blight-inventory.json. The latter contains PowerShell provider metadata and should be reduced to explicit scalar fields before display; do not dump the full file.

## Acceptance criteria

- [ ] Record the exact host game, V+ and BepInEx versions and effective default configuration.
- [x] Install the matching client while preserving Renki configuration, memory and saves.
- [x] Verify join, telemetry and patch coexistence on the selected runtime.
- [x] Pass measured movement, chest transfer and cooking or harvest smoke checks.

## Implementation plan

The user confirmed the server migration is complete and the intended V+ release is latest 10.4.0.0 with default settings. They explicitly requested backups next. Add a reusable `just vm migration-inspect` / `just vm migration-backup` workflow, inventory the actual paths, and make an ACL-restricted verified backup on Blight while both game and local controller are stopped. Include BepInEx and loader files, Renki executable/local settings/dataDir (including atlas/markers/goals), local Valheim saves and Steam Cloud saves when present. Write source-to-backup paths and SHA256 checksums on the VM without printing secret contents. Leave the client offline and do not install or restart during this backup step. Record the confirmed release, backup location and verification on this ticket; installation and live smoke acceptance remain open.

## Notes

**agent:codex/renki-milestones** at 2026-11-01T20:20:09Z

The host has switched the server; the user confirmed Valheim Plus 10.4.0.0 as the intended release with default settings. The latest request was specifically to do the backups, so installation and restart were not performed.

Verified offline backup: C:\renki\backups\vplus-20261101T201857721Z on Blight. 102 files, 87587416 bytes. Every copied file SHA256 matched the source before and after copying; the backup has manifest.json with source-to-backup paths and hashes plus complete.json. ACLs restrict the backup to the current account and SYSTEM. Both Valheim and local controller were stopped before and after copying.

Included current BepInEx installation and Doorstop loader files; controller executable, local settings and secrets; configured controller data directory (including SQLite atlas); local character/world save folders; Valheim Steam userdata from both detected accounts. No base game or remote server world backup is claimed.

Reusable tooling: just vm migration-inspect and just check vm migration-backup. Both ran successfully on Blight; git diff --check passed. Evidence: .checks/modpack-2323/backup-complete.json; .checks/20261101T201842Z-od2dMA.log; .checks/20261101T201859Z-shUFhs.log. Tools/docs are working-tree changes; no commit, push or mod installation in this backup step.

All migration acceptance boxes remain open: the backups satisfy the preservation prerequisite, not installation or live compatibility. Next inspect the actual server game/runtime requirements, stage the confirmed V+ 10.4.0.0 DLL without blindly overwriting Blight's newer BepInEx runtime, and coordinate startup and smoke checks.

**agent:codex/renki-milestones** at 2026-11-01T20:31:50Z

Installed official Valheim Plus 10.4.0.0 DLL (SHA256 9f6070177a0c58a2edb598b78a7e444dab533d87855c5ecc476e2be244aaae6b) beside Renki. The installer reverified all backup hashes and retained the Renki DLL hashes, game and BepInEx runtime. Fresh default config generated; there was no previous V+ modern or legacy config in use. Two joins succeeded into DizzysHellheim as Tasleroch. Observed client: Valheim 1.0.21, BepInExPack 5.4.2360, loader 5.4.23.6. V+ registered 462 settings, received server configuration, reapplied patches and logged zero settings changed. Network handshake matched server/client 41. No V+/Renki patch-loading failure was observed; two initial Unity matchmaking fallback errors preceded a successful join.

Measured cooking check passed: withdrew CookedDeerMeat1 and RawMeat1 from live chest 1:73565; cooked RawMeat at station 1:73615 and collected CookedMeat1 with the native receipt. Evidence .checks/cooking-20261101T202512Z/summary.json. Startup recovery refused an early movement attempt and subsequently healed safely; the retry and one home-storage round trip then passed. Route evidence /home/sothr/.t3/worktrees/renki/t3-b9d91179/.checks/routes-20261101T202959970535Z/summary.json; observed travel 7.14m, minimum HP 88.3. Four idle definitions were restored. Final observed phase in_world, position [-36.48, 36.09, 6.58], HP 87.5/87.5, injurySeq 0.

Validation: just check ci passed in 157s (.checks/20261101T202525Z-0rZChk.log); verified download, installation, read-only inspection and live smoke recipes passed. Tools and rollback procedure are in scripts/vm and docs/VM.md; observations in docs/TESTING.md.

AC1 remains unticked: the target V+ release, client runtime, network compatibility and effective default sync are verified, but the exact host game and BepInEx build strings were not independently exposed. Public server query ports timed out. No server access or version claims are invented. Operational migration is working; host-runtime inventory is the remaining audit item. The current client runtime differs from the upstream documented supported combination (Valheim 1.0.20 / BepInExPack 5.4.2351), so smoke success is not a general support guarantee.

**agent:codex/renki-milestones** at 2026-11-01T20:32:06Z

in-progress to blocked: Deployment and smoke checks passed; only exact host game/BepInEx build inventory remains unavailable without server-side version evidence (public query timed out).

**agent:claude-code/renki-plan-m4** at 2026-11-02T15:57:46Z

### Remaining for M4 goal 4.6

Only criterion 1 is open: the exact host game, Valheim Plus and BepInEx versions and the effective server-synced configuration. The blocking reason says the public query timed out. This needs the owner or the server host to report the versions; no agent can get them from the client side. Nothing else in M4 depends on this ticket.

Related: TKT-01M6DDBX7K (VM trials: combat setting, cleanup stop race, stale V+ backups) hardens `scripts/vm/vplus.ps1`'s backup choice.

### How to verify

A note with the host's versions and config, compared against `just vm vplus-inspect` (read-only) and the client's startup log.

**agent:claude-code/renki-plan** at 2026-11-02T16:15:34Z

Applied on 2026-11-02 by the planning pass: milestone cleared, so this ticket is unscheduled. No exit goal in docs/ROADMAP.md needs it. This supersedes any earlier note that only proposed the move. The owner can schedule it at a weekly review.
