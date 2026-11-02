---
schema: 4
id: TKT-01M6DDBX7KGCV2SNNCMMKXF068
title: "VM trials: combat setting, cleanup stop race, stale V+ backups"
type: bug
status: draft
status_reason: null
priority: low
due_on: null
labels:
  - vm
  - tooling
  - safety
assignees: []
milestone: M4
parent: TKT-01M6DCF2ABDDNG3QF369NDPBT5
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T05:57:55Z
updated_at: 2026-11-02T06:10:53Z
created_by:
  id: agent:claude-code/renki-review
  name: ""
updated_by:
  id: agent:claude-code/renki-review
  name: ""
extensions: {}
---

## Description

Three gaps in the VM trial and Valheim Plus scripts added by #152.

### 1. Trials no longer set the combat permission

`scripts/vm/combat-trial.py:277-280`: the stop, `combat <bool>`, start sequence now runs only with `--restart-client`, and cleanup no longer restores combat to off. A `harvest-trial` (whose help at :218 still says "with combat off") can therefore run with combat left on by an earlier combat trial. Tas may then fight mid-harvest, and benchmark results are no longer comparable.

Fix: read the effective combat setting from status and refuse a mismatch unless `--restart-client` is given. Correct the help text either way.

### 2. Cleanup can stop a human's order

`scripts/vm/combat-trial.py:425-437` reads `/api/status`, then sends a separate `task stop`. A human order that takes the body between the two calls is cancelled, which breaks the rule in the comment that "a later human order is never cleanup authority". Fix: a stop that names the task ID (add one to the panel API if it is missing), or re-check after stopping and report.

### 3. The V+ install accepts any old backup

`scripts/vm/vplus.ps1:14-26` picks the newest complete `vplus-*` backup and checks it only against its own manifest. A backup weeks old lets the install go ahead, and rolling back to it would restore stale memory, goals and character save.

Fix: require the backup to be newer than the last game or controller run (or set an age limit), and print its age.

### Related

V+ server-synced settings that change transfer, pickup or fuel rules are the open first criterion on TKT-01M6CAHE (Migrate Tas to the host Valheim Plus release with defaults).

## Acceptance criteria

- [ ] Trials refuse to run with an unexpected combat setting, and help text matches
- [ ] Trial cleanup cannot cancel a human order issued after its status read
- [ ] vplus install refuses or warns on a backup older than the last game or controller run
