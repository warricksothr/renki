---
schema: 4
id: TKT-01M62VXYV33R1WXRHMRZ6GZMRH
title: "Chat: list, start, stop, clone and edit behaviour instances"
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - chat
  - controller
assignees: []
milestone: M5
parent: TKT-01M62VXYQB6ZDBQE4E6ERGN83C
origin: null
dependencies:
  - TKT-01M62VXYSN4RQV75HTA3BSD162
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:48Z
updated_at: 2026-11-02T16:01:43Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-plan
  name: ""
extensions: {}
---

## Description

Owners control instances from game chat through the command grammar, so nothing here needs the LLM:

- `<name> behaviours`: what exists, what is on, what is running
- `<name> start stow-wood`, `stop stow-wood`, `turn off loot`, `turn on loot`
- `<name> clone stow-wood as stow-stone`
- `<name> set stow-stone items Stone`, `set stow-stone to chest:stone-1, chest:stone-2`

Replies are one line each. A bad edit says which field is wrong and why. A long list goes out in a few lines, kept short.

### Where it stands (2026-11-02)

The engine already has what the commands need: `Engine.List`, `Action` (start, stop, enable, disable), `Clone` and `Put` in `internal/behaviour/engine.go`. The panel uses them through `POST /api/behaviours` (`internal/app/behaviours.go`). Chat has only "go chop wood" (`internal/chat/gather.go`). This ticket adds the chat grammar on top of the same calls, with owner checks as `gatherCommand` does them.

docs/ROADMAP.md schedules editing behaviours from chat with M5 ("an owner clones and edits a behaviour from chat and from the panel"). No M2 goal needs it.

### Files

- `internal/chat/brain.go` (command dispatch), `internal/chat/gather.go` (an owner-checked command to copy)
- `internal/behaviour/engine.go` (`List`, `Action`, `Clone`, `Put`), `internal/behaviour/definition.go` (field validation errors)
- `internal/chat/brain_test.go`

### Verification

- `just check controller test-packages ./internal/chat ./internal/behaviour`, then `just check ci`.

## Acceptance criteria

- [ ] Owners can list, start, stop, enable and disable instances from chat
- [ ] Owners can clone an instance and set one of its parameters from chat
- [ ] A rejected edit names the field and the reason
- [ ] Players who are not owners are refused
