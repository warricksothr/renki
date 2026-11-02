---
schema: 4
id: TKT-01M6F8780VQAZ938F3MY56GS8V
title: "Agent: supply the passive list, with validation and built-in fallback"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - controller
assignees: []
milestone: M5
parent: TKT-01M62VXYQB6ZDBQE4E6ERGN83C
origin: null
dependencies:
  - TKT-01M6EWN4WH7R17QC51VKH4NDYW
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T23:06:28Z
updated_at: 2026-11-02T23:06:37Z
created_by:
  id: agent:claude-code/renki-lead
  name: ""
updated_by:
  id: agent:claude-code/renki-lead
  name: ""
extensions: {}
---

## Description

Part of the idle/passive decision (note on TKT-01M62VXYQB, 2026-11-02). The agent side (internal/voice) should be able to supply the passive list, so passive behaviour can vary without new built-in code.

### What to build

- A tool or API for the agent to propose a passive list: ordered instances of existing templates, with parameters, written as data.
- Validation before activation: known templates, valid parameters, passive mode, and no permission, area or priority escalation. The list is versioned, and the previous version is kept for rollback.
- **Fallback.** The built-in list runs when no list has been supplied, when the list is invalid, or when it is stale (older than a bound, or from before a world change).
- The agent revises the list on events: stash counts or the needs ledger changing, food reserves, or an owner request. It never revises it per tick, and it is never called when Tas goes passive.
- The panel shows the active list, where it came from (agent or built-in), and its version.
- Later, list entries may name procedure packages (docs/PROCEDURES.md, docs/PROCEDURE-TOOLS.md).

### Verification

- Tests: an agent list replaces the built-in one; invalid and stale lists fall back; escalation is refused; rollback works.
- `just check ci`.

## Acceptance criteria

- [ ] The agent side can propose a passive list of template instances, validated with no escalation, versioned with rollback
- [ ] The built-in list runs when none is supplied, or the list is invalid or stale, in tests
- [ ] The panel shows the active list, its source and version
