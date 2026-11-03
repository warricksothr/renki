---
schema: 4
id: TKT-01M6H80F6B8ABPBKZZHB805TZ7
title: "VM: detect runtime drift before installing older builds"
type: bug
status: done
status_reason: null
priority: high
due_on: null
labels:
  - vm
  - tooling
  - safety
  - live
assignees: []
milestone: null
parent: null
origin: null
dependencies:
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-03T17:41:15Z
updated_at: 2026-11-03T18:14:57Z
created_by:
  id: agent:codex/portal-deploy
  name: ""
updated_by:
  id: agent:codex/portal-deploy
  name: ""
extensions: {}
---

## Description

On 2026-11-03, just check vm deploy from portal-following commit 77a527c replaced the VM's newer unmerged runtime 2e8c6d1. Origin/main's handoff described b5cc019, while PR #176 had already installed atlas schema 2, inspect_container and newer protection fixes. Installation checksum verification passed, but the new controller exited with atlas schema 2 newer than supported schema 1. The saved allowlist was also newer than the deploying controller. The game started before this was diagnosed, despite the actual newer handoff retaining an emergency offline state after hostile pursuit.

### Desired behavior

Before replacing binaries or stopping a working controller, retain a deployment manifest with the source commit, supported data versions, installed checksums, current game/controller state and deployment actor. Detect unmerged/newer runtime drift and unsupported durable-data versions, and refuse the replacement with a concrete explanation while leaving the current runtime available. Preserve settings, atlas, game saves and explicit offline state; never reset data to make an older build start. A deliberate compatible build can then be installed through the normal lease-guarded recipes.

### Files and verification

Start in scripts/vm/vm.just, install.ps1, deploy-controller.ps1 and controller startup/version metadata. Add offline deployment-planning checks under just ci and a VM-lease live refusal check that does not alter production data. Cover both full and controller-only installation, missing/legacy manifests, newer schemas/permissions and unchanged/offline runtime state. Keep an operator's explicit start request separate from an install that retains offline state.

Evidence: portal ticket TKT-01M6FT1ZVW (Follow: go through the portal the leader just used), .checks/20261103T173701Z-bcB6T6.log and the deployment recovery notes. This is a new draft; it has not been promoted.

## Acceptance criteria

- [x] Full and controller-only install refuse an incompatible durable schema or newer runtime drift before replacing or stopping the installed controller, in offline tests.
- [x] The deployment manifest records source/version, checksums, actor and prior runtime state without credentials; settings, atlas and explicit offline state are preserved.
- [x] A lease-guarded live refusal check leaves the existing compatible runtime and production data unchanged.

## Implementation plan

Add a deployment manifest generated from the controller's real capabilities and Git ancestry, with source/version, artifact hashes and deployment actor. A staged controller performs read-only config/settings and SQLite schema inspection before either full or controller-only installation may stop the installed runtime or replace its files. Compare installed version/manifest/checksums against the candidate ancestry; safely adopt a recognizable legacy ancestor, refuse unknown/unmerged/newer builds and incompatible data without resetting anything. Retain prior game/controller state and preserve offline state; starting the game remains an explicit deploy/start action. Cover read-only data inspection, ancestry/refusal, manifests and both installation paths with focused offline checks included in just ci. Keep the lease-guarded live refusal criterion open while the owner requires the VM game offline.

## Notes

**agent:codex/portal-deploy** at 2026-11-03T18:05:28Z

### Offline implementation and native refusal staging

Implemented on t3/renki-runtime-loot-batch, stacked above the deployed portal branch. Full and controller-only installers stage a candidate and validate artifact hashes, committed source ancestry, current runtime receipt/version/checksums, saved task permissions and a read-only SQLite user_version before stopping/replacing the installed runtime. The package manifest records source/branch/ancestry, schema/task support, hashes and deployment actor; the installed receipt records verified hashes and previous game/controller state. Clean identifiable legacy ancestors may be adopted; unknown, dirty, unmerged/newer or incompatible candidates refuse. Secrets are inherited for config validation without entering metadata or output. No writable store or startup runs in preflight; local settings, atlas and saves are retained.

Focused compatibility/data tests passed for full/controller packages, unsupported schemas/permissions, mismatched artifacts/receipts/ancestry, dirty or unknown legacy versions, and unchanged atlas/settings bytes. just check controller test-packages ./internal/deployment ./internal/behaviour -race passed (.checks/20261103T180323Z-1ZsXHp.log). just check vm test-deployment passes and is in just ci. docs/VM.md describes the flow. The safe native verification recipe is just check vm deployment-refusal: with the game already offline and panel running, under the VM lease, first read-check the compatible candidate then exercise both real installers with only the staged ancestry altered. A pass must retain game-offline state, the controller PID and installed/config/data hashes. This improves the initial plan: criterion 3 can be earned without starting the game. Until that recipe actually passes, criterion 3 remains unchecked. Production stays on 871d818 with Loot disabled.

**agent:codex/portal-deploy** at 2026-11-03T18:09:32Z

The complete just check ci gate passed (225 s), retained in .checks/20261103T180530Z-ShHgUG.log. This covers the final activity/diagnostics source, panel unit/UI suites, controller race tests/native builds, plugin/interop and deployment entry-point checks. Committing the batch before committed review; production remains unchanged and the game offline.

**agent:codex/portal-deploy** at 2026-11-03T18:10:30Z

### Native read-only compatibility passed; refusal verifier corrected

The complete gate passed and implementation commit 3b56dff passed just check review origin/t3/continue-renki-bot...HEAD with no findings (.checks/20261103T180933Z-kXEjwI.log). Took the VM lease as agent:codex/portal-deploy and ran just check vm deployment-refusal. The compatible candidate passed native read-only preflight against the real configuration, saved permissions and atlas. The verification harness then failed hashing the live SQLite file because Get-FileHash does not share its handle with the controller's writer. Evidence: .checks/20261103T180946Z-LNPIK6.log. The incompatible-manifest installer calls had not run yet; criterion 3 stays unchecked.

Corrected the verifier to hash through a FileAccess.Read handle with FileShare.ReadWrite/Delete. This does not request file writes or stop the controller. Re-run after validation/commit. The game remained offline; no binaries or production configuration/data were replaced. The staged candidate is 3b56dff and production remains 871d818.

**agent:codex/portal-deploy** at 2026-11-03T18:14:57Z

### Native offline compatibility and both refusal paths verified

After the shared-read correction, just check ci passed again (180 s; .checks/20261103T181030Z-mkQMwC.log). Candidate bc64ae9 passed just check review origin/t3/continue-renki-bot...HEAD with no findings (.checks/20261103T181344Z-riCsUg.log). Under the VM lease held by agent:codex/portal-deploy, just check vm deployment-refusal passed in 11 s (.checks/20261103T181353Z-vMP7sd.log).

The unmodified candidate manifest passed native read-only compatibility against the installed configuration, saved permissions and atlas. After removing the installed source from that staging manifest only, both install.ps1 and deploy-controller.ps1 refused with runtime drift before process stop or binary replacement. The verifier retained the running controller PID and unchanged installed artifact/config/data hashes across both refusals. The game stayed offline. Final just vm status confirmed controller PID 39640, Valheim not running and bot disconnected. The VM lease was released after verification.

This earns criterion 3 without starting Valheim. Production remains 871d818; this new batch was not installed and no data was reset/downgraded. The initial sharing failure note remains as historical evidence; this note supersedes its pending-refusal result. The independent VM lease ticket and its A/B field criterion were not changed.

## Summary

Implemented committed manifest/ancestry, checksum, schema and permission preflight before both install paths stop or replace runtime. Full CI and committed review passed; lease-guarded native compatible inspection and both refusal paths passed with controller PID/data hashes preserved and game offline. Production remains 871d818.
