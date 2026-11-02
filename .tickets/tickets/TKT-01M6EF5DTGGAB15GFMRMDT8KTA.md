---
schema: 4
id: TKT-01M6EF5DTGGAB15GFMRMDT8KTA
title: "VM: exclusive lease for deploys and live trials"
type: task
status: ready
status_reason: null
priority: high
due_on: null
labels:
  - vm
  - tooling
  - safety
assignees: []
milestone: M2
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:48:34Z
updated_at: 2026-11-02T19:17:51Z
created_by:
  id: agent:claude-code/renki-plan
  name: ""
updated_by:
  id: agent:claude-code/renki-wave1
  name: ""
extensions: {}
---

## Description

There is one Windows VM and one Valheim client, and nothing stops two agents from using them at once. `just vm deploy`, `just vm install` and every supervised trial (`gather-trial`, `combat-trial`, `harvest-trial`, `goal-trial`, `resupply-food`, `escort-check`, `route-benchmark`, ...) in `scripts/vm/vm.just` replace binaries, restart the client or move the bot without checking whether another session is mid-trial. Two parallel agents would overwrite each other's plugin DLL or controller.exe, abort each other's trials, and produce `summary.json` evidence that describes neither run.

This blocks spreading roadmap work across parallel sub-agents (docs/ROADMAP.md, "Working in parallel").

### Suggested approach

- Add a lease recipe pair, for example `just vm lease ACTOR [minutes]` and `just vm release`, plus a guard every mutating `vm` recipe calls first.
- Keep the lease on the VM itself (a small JSON file next to the deployed controller, written atomically), holding the actor, the branch or worktree, the start time and an expiry. The VM is the shared resource. A lease on the workstation would not protect against a second workstation.
- A mutating recipe refuses with the current holder and expiry when someone else holds an unexpired lease. Read-only recipes (status, audit windows, map snapshots, panel requests that only read) stay available.
- Expiry bounds a crashed session; the default is 60 minutes, renewable by the holder.
- The trial supervisors keep their own cleanup (combat off, stop or logout) and release the lease only after that cleanup.

### Files

- `scripts/vm/vm.just`: the recipes and the guard
- `scripts/vm/common.ps1`: a shared helper if the guard runs VM-side
- `docs/VM.md` and `docs/TESTING.md`: how to take and release the lease
- `AGENTS.md`: one line saying live work takes the lease

### Verification

Offline tests for the guard (held by another actor, expired, held by self, malformed file) wired into `just ci` alongside the other `vm::test-*` recipes. One live check that a second `just vm deploy` is refused while a trial holds the lease.

## Acceptance criteria

- [ ] Every mutating vm recipe refuses while another actor holds an unexpired lease, and names the holder and expiry
- [ ] Read-only vm recipes keep working while a lease is held
- [ ] Leases expire on their own and the holder can renew or release them
- [ ] Trial supervisors release only after their cleanup
- [ ] Offline guard tests run in just ci
- [ ] docs/VM.md, docs/TESTING.md and AGENTS.md say how to take and release the lease

## Notes

**agent:claude-code/renki-plan** at 2026-11-02T16:59:47Z

Not labelled live, although its last check runs on the VM. This ticket creates the lease, so it cannot wait for one, and it has to show in `git ticket ready --not-label live`, the parallel queue, because every live ticket depends on it. Agree its one live check with the owner, so that no other session is using the VM at the time.
