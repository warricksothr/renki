---
schema: 4
id: TKT-01M6EHP58NR3825QRA6PP0ZHD1
title: "Escort: report whether a defence was for a charge or for Tas himself"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - plugin
  - protocol
  - safety
  - live
assignees: []
milestone: M3
parent: TKT-01M62N2557R401S3E6JECSAPN4
origin: null
dependencies:
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T16:32:40Z
updated_at: 2026-11-02T16:32:40Z
created_by:
  id: agent:claude-code/renki-plan
  name: ""
updated_by:
  id: agent:claude-code/renki-plan
  name: ""
extensions: {}
---

## Description

Goal 3.3 in docs/ROADMAP.md counts only defences against a *confirmed attacker of a protected player*. Today `state.escort` cannot show that. In defend mode `EscortTask.Defend` always sets the reason "confirmed attacker targets a protected player" (plugin/Renki.Plugin/EscortTask.cs, `Defend`). That includes threats confirmed only because they target Tas himself or are `MeleeTask.RecentAttacker` (`Threats`, the `victimKnown` lines). So the escort report (TKT-01M6EFHEHM, escort-check: report defences, owner distance and stop latency) has to record every defence as `unknown`, and goal 3.3 cannot pass.

### What to build

- `Threats` already ranks priority 0 for a creature whose AI target is a protected player other than Tas, 1 for a creature targeting Tas or his recent attacker, and 2 for proximity only. Keep that ranking. Also return which kind won.
- Add an optional `trigger` field to `EscortStatus`, with the values `protected_target` (priority 0), `self_hit` (priority 1) or `proximity` (priority 2). Set it whenever mode is `defend`, and add the protected player's name as `victim` for `protected_target`.
- Make the defend `reason` text match the trigger, so a self-hit no longer reads as "targets a protected player".
- Mirror the fields in `internal/protocol/messages.go`, and document them in docs/BRIDGE-PROTOCOL.md (state, escort) and docs/ESCORT.md ("Diagnostics and validation").

### Files

- `plugin/Renki.Plugin/EscortTask.cs`: `Threats`, `Defend`, `Reset`
- `plugin/Renki.Core/Protocol/Messages.cs`: `EscortStatus`
- `internal/protocol/messages.go`: `EscortStatus`
- `docs/BRIDGE-PROTOCOL.md`, `docs/ESCORT.md`

### Verification

- If the ranking is moved into Renki.Core, `just check plugin test-core` with a test per trigger kind. Otherwise a protocol round-trip test for the new fields: `just check controller test-packages ./internal/protocol`.
- `just check ci`. Renki.Plugin needs `VALHEIM_DIR` to compile; without it, `just vm deploy` (holding the lease, TKT-01M6EF5DTG) compiles and installs it.
- Live: one escort run under `just vm escort-check` whose audit window shows `trigger` on each defend state.

## Acceptance criteria

- [ ] state.escort carries trigger (protected_target, self_hit or proximity) whenever mode is defend, and victim for protected_target
- [ ] The defend reason text matches the trigger
- [ ] The Go protocol type, BRIDGE-PROTOCOL.md and ESCORT.md include the new fields
- [ ] One live escort run's audit window shows trigger on each defend state
