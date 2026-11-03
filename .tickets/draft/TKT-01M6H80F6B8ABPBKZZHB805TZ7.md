---
schema: 4
id: TKT-01M6H80F6B8ABPBKZZHB805TZ7
title: "VM: detect runtime drift before installing older builds"
type: bug
status: draft
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
updated_at: 2026-11-03T17:41:51Z
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

- [ ] Full and controller-only install refuse an incompatible durable schema or newer runtime drift before replacing or stopping the installed controller, in offline tests.
- [ ] The deployment manifest records source/version, checksums, actor and prior runtime state without credentials; settings, atlas and explicit offline state are preserved.
- [ ] A lease-guarded live refusal check leaves the existing compatible runtime and production data unchanged.
