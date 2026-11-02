---
schema: 4
id: TKT-01M6EFKDNH2TZ04SB1N8KZP06A
title: "Live check: owner clones and edits a behaviour from chat and panel"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - chat
  - live
assignees: []
milestone: M5
parent: TKT-01M62VXYQB6ZDBQE4E6ERGN83C
origin: null
dependencies:
  - TKT-01M62VXYV33R1WXRHMRZ6GZMRH
  - TKT-01M6EFK472WBD151TR1K7JE4N1
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:56:13Z
updated_at: 2026-11-02T15:56:13Z
created_by:
  id: agent:claude-code/renki-plan-m5
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m5
  name: ""
extensions: {}
---

## Description

M5 goals 5.4 and 5.5 (docs/ROADMAP.md) need live evidence that an owner can clone and edit a behaviour from chat and from the panel, and that the edited instance does what the edit says. The code lives in TKT-01M62VXYV3 (Chat: list, start, stop, clone and edit behaviour instances) and the panel ticket filed with this one. This ticket is the live check only.

### What to do

On the VM, holding the lease (TKT-01M6EF5DTG), with Tas idle at home and a saved disabled `stow` instance named `stow wood` whose chests are known home chests:

1. From game chat as an owner, three times with fresh names: clone `stow wood` as `stow stone N`, set its items to `Stone`, set its chests, and start it, with Stone in the pack. Each command's reply must arrive within 2 s, and the run must end with a deposit whose receipt matches the named chest.
2. A player who is not an owner sends the same clone. It is refused, and `GET /api/behaviours` shows no new instance.
3. From the panel, clone `stow wood`, change its items in the editor, save and start it. `GET /api/behaviours` shows the instance changed by `panel`, and its run deposits with a matched receipt.

Save an audit window per run with `just vm audit-window` and summarise it with `just vm audit-summary`. Combat stays off. Remove the test clones afterwards.

### Files

- `scripts/vm/vm.just`: `audit-window`, `audit-summary`, `panel`
- `docs/BEHAVIOURS.md`: the control API and stow parameters
- `docs/TESTING.md`: record the procedure if it becomes a recipe

### Verification

Notes on this ticket link the three chat audit windows, the refusal and the panel window, with reply latencies and the matched deposit receipt for each. If the sequence is repeated more than once, turn it into a `just vm` recipe with an offline self-test rather than repeating it by hand.

## Acceptance criteria

- [ ] Three chat sequences (clone, set items, set chests, start) each reply within 2 s and end in a deposit with a matched receipt, with audit windows linked
- [ ] A non-owner's clone is refused and GET /api/behaviours shows no new instance
- [ ] One panel clone and edit appears in GET /api/behaviours as changed by panel and runs once with a matched deposit receipt
- [ ] Test clones are removed and combat stayed off throughout
