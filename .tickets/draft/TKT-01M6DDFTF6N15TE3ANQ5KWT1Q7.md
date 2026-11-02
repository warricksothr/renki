---
schema: 4
id: TKT-01M6DDFTF6N15TE3ANQ5KWT1Q7
title: "VM: 24 h soak with crash, leak, growth and store checks"
type: task
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
milestone: M4
parent: TKT-01M6DCF2ABDDNG3QF369NDPBT5
origin: null
dependencies:
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T06:00:03Z
updated_at: 2026-11-02T15:57:05Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m4
  name: ""
extensions: {}
---

## Description

M4 (safe to leave running) needs a long unattended run that fails loudly. docs/ROADMAP.md goal 4.2 defines it.

### What to build

A `just vm soak HOURS` supervisor, with an offline self-test in `just ci`. It starts or keeps the bot idle-working at home (marker idle food work, bedside rest, cargo unloading), samples once a minute, and writes `.checks/soak-<ts>/summary.json` with a time series and a verdict.

It fails the soak on any of these:

- the controller process restarts or exits;
- more than 100 open TCP connections owned by the controller or obs-adapter, measured with `Get-NetTCPConnection` on the VM;
- audit directory growth projected over 2 GB a day;
- any store write error in the controller log (memory, goals, settings, atlas);
- any plugin task failure with reason `internal`;
- the plugin's dead-man stop firing with no reconnect within 10 s.

It must never print the panel token. See the VM checks memory in docs/VM.md for how to call the panel from the VM.

### Why

The review found an OBS adapter leak (one connection per poll), audit files with no size cap, and a store that blocks startup when damaged. All three show up only over hours. A soak turns "it ran overnight" into evidence.

### Scope

This ticket builds the supervisor and its offline self-test, and proves it with a short live run. The 24 h run that goal 4.2 needs is TKT-01M6EFGYV6 (VM: run the 24 h soak and link its summary from the M4 epic); the third criterion moved there.

### Files

- `scripts/vm/vm.just` (the `soak` recipe and a `test-soak` recipe), a new `scripts/vm/soak.py` and `scripts/vm/soak-test.py`, a VM-side sampler beside `scripts/vm/status.ps1` (uses `Get-NetTCPConnection`)
- `scripts/vm/panel-request.ps1` and `scripts/vm/trial-status.py` for reading panel status without printing the token
- `justfile` (add `vm::test-soak` to `ci`)
- `docs/VM.md`, `docs/TESTING.md`

### Verification

- `just check vm test-soak` covers every failure condition with recorded samples, and `just check ci` runs it.
- Live, under the VM lease (TKT-01M6EF5DTG): a 15-minute `just vm soak` writes `.checks/soak-<ts>/summary.json` with every sampled field filled and a verdict.

## Acceptance criteria

- [ ] just vm soak HOURS samples once a minute and writes a summary with a verdict
- [ ] Each failure condition in the description fails the soak, covered by an offline self-test in just ci
- [ ] A 15-minute just vm soak on the VM writes summary.json with every sampled field filled and a verdict
