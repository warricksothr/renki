---
schema: 4
id: TKT-01M6EFK472WBD151TR1K7JE4N1
title: "Panel: list, start, stop, clone and edit behaviour instances"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - controller
assignees: []
milestone: M5
parent: TKT-01M62VXYQB6ZDBQE4E6ERGN83C
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:56:03Z
updated_at: 2026-11-02T15:56:03Z
created_by:
  id: agent:claude-code/renki-plan-m5
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m5
  name: ""
extensions: {}
---

## Description

M5 goal 5.5 (docs/ROADMAP.md): an owner clones a behaviour and edits it from the panel. The controller already has the whole API (`GET /api/behaviours`, `POST /api/behaviours` with `save`, `start`, `stop`, `enable`, `disable`, `delete` and `clone`, docs/BEHAVIOURS.md "Control API"), but the panel only drives two presets through it: idle food work on a place marker and household food. There is no list of instances and no way to clone or edit one. The first criterion of TKT-01M628X13X (Panel: behaviour switches that apply without a restart) asks for exactly this and has waited since 2026-10-29 on the engine, which is now done.

### What to build

A Behaviours section in the panel workspace:

- A list of every instance from `GET /api/behaviours`: name, template, enabled, running or queued, run count, last result, and any storage-needed note. It refreshes as the other sections do.
- Start, stop, enable, disable and delete buttons per row, posting the matching action with `X-Renki-Act`.
- Clone with a new name (clones start disabled, as the engine does).
- An editor for one instance's `params`, `trigger`, `repeat`, `priority` and `enabled`, as structured fields where the template is known (`stow`: items, to, keep, overflow) and raw JSON otherwise. Save posts `action: save` with the whole definition.
- A rejected save shows the controller's error, which names the field and the reason. Nothing is applied on a rejection.

### Files

- `panel/src/controller.jsx`: the existing `/api/behaviours` calls around the idle-food and household controls
- `panel/src/workspace.jsx`: where workspace sections are laid out
- `panel/ui/scenarios.mjs` and `panel/ui/fixtures/`: the browser scenarios and their fixtures
- `internal/app/behaviours.go`: the API the panel calls (change only if an error message does not name its field)
- `docs/PANEL.md` and `docs/BEHAVIOURS.md`: replace "The engine controls do not yet have dedicated page or chat controls" for the panel half

### Verification

`just check panel ci` with new UI scenarios: list renders from a fixture, clone posts the right request, an edit saves the changed field only, a rejected edit shows the field and reason and leaves the row unchanged, start and stop post their actions. `just check controller test-packages ./internal/app` if the API changes. `just check ci`. No game is needed; the live use is the check ticket filed with this one.

## Acceptance criteria

- [ ] The panel lists every instance from GET /api/behaviours with template, enabled, running or queued, run count and last result
- [ ] Start, stop, enable, disable, delete and clone each post the matching action with X-Renki-Act, covered by a UI scenario
- [ ] Editing one stow parameter and saving posts the whole definition with only that field changed, covered by a UI scenario
- [ ] A rejected save shows the controller's field and reason and leaves the instance unchanged, covered by a UI scenario
- [ ] docs/BEHAVIOURS.md and docs/PANEL.md no longer say the panel lacks behaviour controls
