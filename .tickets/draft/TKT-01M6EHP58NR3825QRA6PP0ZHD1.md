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

- `Threats` already ranks priority 0 for a creature whose AI target is a protected player other than Tas, 1 for a creature whose AI target is Tas or that is `MeleeTask.RecentAttacker`, and 2 for proximity only. Only confirmed threats (priority 0 or 1) start a defence, so proximity never appears in defend mode. Keep that ranking, but do not derive the trigger from the priority number: priority 0 has no check that the charge is within 12 m of the leader, so a `RecentAttacker` aimed at a distant charge ranks 0 on the strength of the hit alone. Compute the trigger from which clause of `victimKnown` matched, in this order: the protected-ally clause (charge within 12 m of the leader), then `target == p`, then `MeleeTask.RecentAttacker`.
- Add an optional `trigger` field to `EscortStatus`, with the values `protected_target` (the protected-ally clause matched), `self_target` (the AI target is Tas and no ally clause matched) or `self_hit` (only the `RecentAttacker` clause matched). Set it whenever mode is `defend`, and add the protected player's name as `victim` for `protected_target`.
- Make the defend `reason` text match the trigger, so a self-hit no longer reads as "targets a protected player".
- Mirror the fields in `internal/protocol/messages.go`, and document them in docs/BRIDGE-PROTOCOL.md (state, escort) and docs/ESCORT.md ("Diagnostics and validation").

### Files

- `plugin/Renki.Plugin/EscortTask.cs`: `Threats`, `Defend`, `Reset`
- `plugin/Renki.Core/Protocol/Messages.cs`: `EscortStatus`
- `internal/protocol/messages.go`: `EscortStatus`
- `docs/BRIDGE-PROTOCOL.md`, `docs/ESCORT.md`

### Verification

- If the ranking is moved into Renki.Core, `just check plugin test-core` with a test per trigger kind, including a recent attacker whose AI target is a charge more than 12 m from the leader (`self_hit`, not `protected_target`). Otherwise a protocol round-trip test for the new fields: `just check controller test-packages ./internal/protocol`.
- `just check ci`. Renki.Plugin needs `VALHEIM_DIR` to compile; without it, `just vm deploy` (holding the lease, TKT-01M6EF5DTG) compiles and installs it.
- Live: one escort run under `just vm escort-check` whose audit window shows `trigger` on each defend state.

## Acceptance criteria

- [ ] state.escort carries trigger (protected_target, self_target or self_hit) whenever mode is defend, and victim for protected_target
- [ ] The defend reason text matches the trigger
- [ ] The Go protocol type, BRIDGE-PROTOCOL.md and ESCORT.md include the new fields
- [ ] One live escort run's audit window shows trigger on each defend state
