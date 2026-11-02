---
schema: 4
id: TKT-01M6EFHE80EACX255261HK14HN
title: "VM: wood-request trial for go chop wood and its refusals"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - vm
  - tooling
  - controller
  - live
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies:
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
  - TKT-01M62N12G98VW2CYW94FG5RCTB
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:55:08Z
updated_at: 2026-11-02T15:55:22Z
created_by:
  id: agent:claude-code/renki-plan-m2
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m2
  name: ""
extensions: {}
---

## Description

Goals 2.3 and 2.4 in docs/ROADMAP.md need evidence that "go chop wood" picks a sensible site and that it refuses honestly. `just vm gather-trial` (scripts/vm/gather_trial.py) cannot give that evidence. It picks its own tree (`select_site`) and saves its own behaviour definition, so it never runs the owner's request path, `Engine.StartWood` and `planWood` in `internal/behaviour/wood.go`. The only way into `StartWood` today is in-game chat (`internal/chat/gather.go`), and a supervised trial cannot type in the game.

### What to build

- A panel action on `POST /api/behaviours` (`internal/app/behaviours.go`), for example `{"action":"wood"}`, that calls `Behaviours.StartWood` with the panel as the requester and returns its reply line. It needs the same act header as the other actions. Chat stays the owner's path; this is the same request for a supervisor.
- A `--wood-request` mode for the trial supervisor (or a new `just vm wood-trial` recipe beside `gather-trial` in `scripts/vm/vm.just`). It sends the action, records the reply and the planned site, waits for the gather run to finish, and reuses `gather_trial.verify` for fell, split, collect, deposit and return. The summary records: the chosen site, whether it was a remembered work site or the nearest known tree, its distance from home, the reply text, and the duration.
- A refusal mode for goal 2.4's no-axe case: the supervisor first deposits the axe into a home chest with the existing `deposit` task, sends the request, checks that the refusal names the axe within 10 s, then withdraws the axe. Restore the axe in cleanup whatever happens.
- Offline tests in a `wood-trial-test.py`, wired into `just ci` beside `vm::test-gather-trial` in the root `justfile`.

### Files

- `internal/app/behaviours.go`, `internal/behaviour/wood.go`, `internal/chat/gather.go`
- `scripts/vm/gather_trial.py`, `scripts/vm/combat-trial.py` (`--gather-tree` mode), `scripts/vm/gather-trial-test.py`, `scripts/vm/vm.just`, `justfile` (`ci`)
- `docs/TESTING.md`

### Verification

- Offline: `just check controller test-packages ./internal/app ./internal/behaviour` and `just check vm test-gather-trial` (plus the new test recipe), then `just check ci`.
- Live (lease TKT-01M6EF5DTG): three wood-request runs whose summary.json shows outcome `succeeded` and the site source, and one no-axe run whose summary shows the refusal text and its delay under 10 s.

## Acceptance criteria

- [ ] POST /api/behaviours with action wood calls StartWood and returns its reply, behind the act header, with a test
- [ ] A just vm recipe runs the owner request path and its summary.json records site, site source, distance from home, reply and duration
- [ ] A no-axe mode records the refusal text, its delay and `bodyTasksAfterRefusal` (body tasks sent between the request and the supervisor's own withdraw; goal 2.4 needs 0), and always restores the axe
- [ ] Offline supervisor tests run in just ci
- [ ] docs/TESTING.md describes the recipe
