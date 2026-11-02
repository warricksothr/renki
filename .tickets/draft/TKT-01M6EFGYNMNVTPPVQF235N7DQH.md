---
schema: 4
id: TKT-01M6EFGYNMNVTPPVQF235N7DQH
title: "VM: pause the controller 20 s and time the plugin's reconnect"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - vm
  - live
  - protocol
assignees: []
milestone: M4
parent: TKT-01M6DCF2ABDDNG3QF369NDPBT5
origin: null
dependencies:
  - TKT-01M6DCFS8Y4W0BW6KE3HYMZBCP
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:54:52Z
updated_at: 2026-11-02T15:58:03Z
created_by:
  id: agent:claude-code/renki-plan-m4
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m4
  name: ""
extensions: {}
---

## Description

Goal 4.5 says that after a 20 s controller pause the plugin reconnects and accepts tasks again within 10 s, shown by a bridge test plus one live pause. TKT-01M6DCFS8Y (Bridge: reconnect after a dead-man stop, back off on 4409) delivers the fix and the offline test. This ticket is the live pause on the VM.

### What to do

- Take the VM lease (TKT-01M6EF5DTGGAB15GFMRMDT8KTA, VM: exclusive lease for deploys and live trials).
- Deploy the build containing TKT-01M6DCFS8Y with `just vm deploy`, with Tas idle at home and no trial running.
- Suspend the controller process for 20 s and resume it. Windows has no built-in suspend command, so add a small recipe (for example `just vm pause-controller SECONDS`) that calls `NtSuspendProcess`/`NtResumeProcess` from PowerShell, in `scripts/vm/`. A recipe, because this check is repeated by the soak.
- Record from the plugin log and the controller audit: when the dead-man stop fired, when the new connection's `welcome` arrived, and when a task sent after resume (for example a `stop` or a short move from the panel) was accepted.

### Files

- `scripts/vm/vm.just` (the new recipe), a new `scripts/vm/*.ps1` helper
- `docs/VM.md` (document the recipe)

### Verification

- `just vm pause-controller 20` (or the recipe's final name) on the VM.
- A note on this ticket with the three timestamps; the gap from resume to accepted task is 10 s or less.
- `just vm audit-window SINCE UNTIL` covering the pause is linked in the note.

## Acceptance criteria

- [ ] A just vm recipe suspends and resumes the controller for a given number of seconds, documented in docs/VM.md
- [ ] After a 20 s pause on the VM the plugin reconnects and accepts a task within 10 s of resume
- [ ] A note records the dead-man, welcome and accepted-task times and links the audit window
