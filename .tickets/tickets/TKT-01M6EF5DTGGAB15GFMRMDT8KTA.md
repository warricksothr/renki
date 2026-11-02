---
schema: 4
id: TKT-01M6EF5DTGGAB15GFMRMDT8KTA
title: "VM: exclusive lease for deploys and live trials"
type: task
status: review
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
claim:
  actor: agent:claude-code/renki-m2-lease
  branch: t3/vm-lease
  worktree: /home/sothr/.t3/worktrees/renki/t3-6807db0c
  commit: e0ea1b0833b2d4f26205e097313156e24c5bf0e5
  session: null
  claimed_at: 2026-11-02T19:20:25Z
  expires_at: null
archive: null
created_at: 2026-11-02T15:48:34Z
updated_at: 2026-11-02T21:34:17Z
created_by:
  id: agent:claude-code/renki-plan
  name: ""
updated_by:
  id: agent:claude-code/renki-lead
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

- [ ] Every mutating vm recipe except the emergency stops (`stop`, `panel POST /api/act/stop`) refuses while another actor holds an unexpired lease, and names the holder and expiry
- [x] Read-only vm recipes keep working while a lease is held
- [x] Leases expire on their own and the holder can renew or release them
- [x] Trial supervisors release only after their cleanup
- [x] Offline guard tests run in just ci
- [x] docs/VM.md, docs/TESTING.md and AGENTS.md say how to take and release the lease

## Implementation plan

The lease lives on the VM as `C:/renki/lease.json` (id, actor, workstation host, worktree, branch, started, expires). `scripts/vm/lease.ps1` is a small VM-side helper sent with `-EncodedCommand` (no scp per call): `Read` returns the VM's UTC clock, the file's SHA-256 and its bytes; `Write` is a compare-and-swap that, under an exclusive lock file, replaces (temp file + File.Replace/Move) or deletes the lease only if its hash still matches what was read. Expiry is measured on the VM clock, so workstation clock skew does not matter.

`scripts/vm/lease.py` holds every decision, so it is testable offline: `status` (read-only), `take ACTOR [minutes]` (take or renew, default 60), `release [force]`, `guard` (pass when held by this worktree's lease id, take it when free or expired, otherwise refuse naming holder and expiry) and `run -- CMD` (take if needed, renew while the child runs, release only after the child, which does its own cleanup, has exited, and only if `run` took it). The holder is identified by a random lease id stored in the worktree's gitignored `.vm-lease`.

`vm.just`: `lease`, `release`, `lease-status` recipes; a private `guard` dependency on every mutating recipe (start/restart/reconnect/install/deploy/deploy-controller/panel-start/panel-stop/panel-setup/secret/migration-backup/vplus-install, the setters with a VALUE, `panel POST` except `/api/act/stop`); the emergency stops, `stop` and `panel POST /api/act/stop`, are left unguarded for a person by the owner's decision, but guard inside a `lease.py run` (RENKI_VM_LEASE_RUN), so a cleanup that lost its lease cannot stop the next holder; supervisors (combat-trial family, task-check, marker-check, camera-check, escort/sleep/cooking/food/marker-food checks, recover-home, route-benchmark) run through `lease.py run`. Read-only recipes are untouched.

Offline tests in `scripts/vm/lease-test.py` (other holder, expired, self, malformed, renew/release, run releases after child cleanup) use a local-file transport and are wired into `just ci` as `vm::test-lease`. Docs in VM.md, TESTING.md, one AGENTS.md line. The live refusal check is left to the owner with exact commands in a note.

## Notes

**agent:claude-code/renki-plan** at 2026-11-02T16:59:47Z

Not labelled live, although its last check runs on the VM. This ticket creates the lease, so it cannot wait for one, and it has to show in `git ticket ready --not-label live`, the parallel queue, because every live ticket depends on it. Agree its one live check with the owner, so that no other session is using the VM at the time.

**agent:claude-code/renki-m2-lease** at 2026-11-02T19:29:43Z

Implemented on branch t3/vm-lease. Offline: `just vm test-lease` (in `just ci`) covers another holder (refusal names actor and expiry, file untouched), expiry (VM clock, next guard takes it), held by self (guard passes without a write, renew keeps id and start, release forgets the local id), malformed files (refused until `release --force`), forced release, `run` holding the lease through the child's own cleanup and releasing after it, `run` leaving a lease taken with `take`, renewal while running, and the compare-and-swap refusing a stale write. recover-home-test now also shows a mutating recipe (`recovery-return`) takes the lease.

Live, only the read path was exercised: `just vm lease-status` against the real VM printed "the VM lease is free", so lease.ps1's encoded Read works. Nothing was written to the VM. Criterion 1 is left unticked until the owner proves the write path and the refusal on the VM, with no other session using it:

1. Worktree A: `just vm lease agent:owner/a 15`, then `just vm lease-status` (shows the holder and expiry; C:\renki\lease.json exists).
2. Worktree B (a second checkout, its own `.vm-lease`): `just vm deploy` must exit 1 before packaging, printing "refused: the VM is leased to agent:owner/a ... until <expiry>". Also `just vm panel POST /api/act/stop` refused, while `just vm status` and `just vm panel GET /api/status` still work.
3. Worktree A: `just vm release`; in B, `just vm lease-status` says free.
4. Optional trial check: in A with no lease, start any supervised trial (e.g. `just vm marker-check`); during it, B's `just vm lease-status` names A's actor; after it finishes, the lease is free again.

Unverified on Windows until then: the Write action's exclusive lock (`[IO.File]::Open(..., 'None')`) and `[IO.File]::Replace` with `[NullString]::Value`.

**agent:claude-code/renki-m2-lease** at 2026-11-02T21:17:54Z

`just review` took eighteen rounds before it came back clean. The fixes hardened `lease.py run` and the local id record:
- renewal errors only produce a warning;
- a command that fails to start, or a signal that arrives during the take, gives the lease back;
- SIGINT, SIGTERM, SIGHUP and SIGQUIT go to the trial's own process group once, as SIGINT;
- a trial killed by any other signal keeps the lease until it expires;
- `.vm-lease` is locked, replaced atomically and carries a generation counter.

lease.ps1 Read hashes the bytes it returns, opens the file with delete sharing, treats a file deleted mid-read as absent, and formats the VM clock in the invariant culture. Write fails loudly if the delete fails. All of this is covered offline by `just vm test-lease` (18 cases), except the Windows-only parts: the lock, File.Replace and delete-pending access. The owner's live check in the previous note covers those. Criterion 1 stays unticked until it has been run.

**agent:claude-code/renki-lead** at 2026-11-02T21:34:17Z

Supersedes step 2 of the live check in the 19:29 note. The owner chose to exempt the emergency stops, so `just vm stop` and `just vm panel POST /api/act/stop` now run under anyone's lease and do not take it, and `restart` guards before it stops. In step 2, use `just vm panel POST /api/act/home` as the refused POST. Do not run /api/act/stop from B, because it would now stop A's session. The plan has been updated to match.
