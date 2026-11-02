---
schema: 4
id: TKT-01M6EFHEBCYEFSTZNJJPFGAJCP
title: "Retreat: an owner stop must not let a remembered hit restart combat"
type: bug
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - safety
assignees: []
milestone: M3
parent: TKT-01M62N2557R401S3E6JECSAPN4
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:55:08Z
updated_at: 2026-11-02T15:55:08Z
created_by:
  id: agent:claude-code/renki-plan-m3
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m3
  name: ""
extensions: {}
---

## Description

An owner's `stop` or "stay out of it" can be followed within a second by a new `attack` that the owner did not ask for. Goals 3.2 (chat stop ends combat within 2 s) and 3.3 (stop on "stay out of it") in docs/ROADMAP.md fail if that happens, and docs/SELF_DEFENSE.md ("One fight-or-retreat decision") promises that a remembered hit cannot restart an attack after an owner stop.

### What the code does now

`internal/retreat/retreat.go` remembers each actual attacker for `HurtFor` (30 s) in `r.hurt`. On every `Tick`, if the current threat is in `r.hurt`, `defend` may dispatch `attack` with `Defensive: true` (unstarred Greyling within 6 m, at least 65 % HP, 35 stamina, food, no structures within 8 m). `r.defended` limits that to once a minute per creature, but it is only set by `defend` itself. Nothing tells the retreat that an owner stopped. Both chat (`cmdStop` in `internal/chat/brain.go`, which also covers "stay out of it" and "halt") and the panel (`PanelAct` "stop" in `internal/app/panel.go`) dispatch `protocol.TaskStop` and nothing else.

So in a `just vm combat-trial` encounter or an escort defence against a Greyling, an owner stop ends the native attack, and the next retreat tick can start a defensive attack on the same Greyling.

### Outcome

After an owner stop, the retreat never dispatches `attack` for an injury that happened before the stop. Flee stays available: running is always allowed (SELF_DEFENSE.md: "Stopping attacks must preserve the existing escape behavior").

### Files

- `internal/retreat/retreat.go`: add a method such as `Stood(now)` that marks every remembered attacker as not defendable (or clears `hurt` for defence while keeping flee), and check it in `defend`.
- `internal/chat/brain.go` (`cmdStop`, `cmdWait`) and `internal/app/panel.go` (`PanelAct` "stop"): call it. `internal/app/app.go` builds the retreat (`retreat.New`) and the brain, so wire it there.
- `internal/retreat/retreat_test.go`: the existing table at line 137 already has a "stop" case for controlled attacks; add the remembered-hit case.

### Verification

- `just check controller test-packages ./internal/retreat ./internal/chat ./internal/app`
- `just check ci`

No live check is needed: the live stop latency is measured under goals 3.2 and 3.3.

## Acceptance criteria

- [ ] After an owner stop (chat stop, wait here, stay out of it, or panel stop), the retreat sends no attack for any injury that happened before the stop; a test covers chat and panel
- [ ] Flee after a stop still works: a test shows a confirmed hit after the stop can still start a flee
- [ ] A new hit after the stop is treated as new evidence, as today; a test covers it
- [ ] just check ci passes
