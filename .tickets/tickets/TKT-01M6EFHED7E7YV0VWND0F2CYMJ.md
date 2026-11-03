---
schema: 4
id: TKT-01M6EFHED7E7YV0VWND0F2CYMJ
title: "Flee: fail fast on a blocked escape and re-aim on a new attacker"
type: bug
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - plugin
  - controller
  - movement
  - safety
assignees: []
milestone: M3
parent: TKT-01M62N2557R401S3E6JECSAPN4
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/home-survey-delivery
  branch: t3/map-home-location
  worktree: /home/sothr/.t3/worktrees/renki/t3-b02dd361
  commit: 671e8d6a2e8b275a497a77165277fdfa3bec3fd4
  session: null
  claimed_at: 2026-11-03T16:14:08Z
  expires_at: null
archive: null
created_at: 2026-11-02T15:55:08Z
updated_at: 2026-11-03T16:22:40Z
created_by:
  id: agent:claude-code/renki-plan-m3
  name: ""
updated_by:
  id: agent:codex/home-survey-delivery
  name: ""
extensions: {}
---

## Description

Goal 3.1 in docs/ROADMAP.md needs three of three low-HP runs where the bot flees and reaches its owner or home alive, one of them starting on the raised base. On 2026-10-31 that exact case failed: see the last note on TKT-01M62N256M (Plugin: flee task, and retreating when health is low). Flee task t-20 was accepted within 0.3 s with goal [-20.85,36.09,-2.57] from [-33.9,36.09,6.68] and no safety destination (home was within `HomeClear` of the threat). Tas stayed nearly still inside the raised base for 15 s until a Greydwarf_Elite hit him for 27.7, and the owner logged him out. Audit `.checks/audit-window-SNFFE5.jsonl`.

Two gaps caused this. The flee did not notice that it was making no progress, and the controller does not replace a running flee (`Tick` in `internal/retreat/retreat.go` returns early while a `flee` is running), so a second, stronger attacker did not change anything for the rest of the 60 s flee.

Commit 9263f4b (Try checked escape bearings) landed after that failure and added up to seven checked escape bearings for hostile-avoiding flee children (`plugin/Renki.Core/Movement/FleeBearings.cs`). Read it first: it may cover part of the first gap for some callers. The controller's plain `flee` does not set `avoidHostiles`.

### Outcome

- A flee that moves less than about 1 m in 4 s while the threat is within `Near` fails fast with a distinct reason (for example `blocked`) and a trace naming the refused goal, instead of running out its 60 s.
- The controller retries once with a different target: toward home's door or floor exit if one is remembered, else another bearing. It does not loop: at most two flee attempts per threat within `Again`.
- A hit from a different creature during a running flee lets the controller re-aim from the new attacker.
- Unchanged: a successful flee, owner stop, escort and native travel ownership, dry routing and all physical checks.

### Files

- `plugin/Renki.Plugin/FleeTask.cs`, `plugin/Renki.Core/Movement/Flee.cs`, `plugin/Renki.Core/Movement/FleeBearings.cs`, `plugin/Renki.Core/Movement/RouteProgress.cs`
- `internal/retreat/retreat.go` and `internal/retreat/retreat_test.go`
- `docs/BRIDGE-PROTOCOL.md` (flee reasons) if a reason is added; keep `internal/protocol` and `plugin/Renki.Core/Protocol` in step.
- Fixtures: build the captured raised-base positions above into a test, beside `plugin/Renki.Tests/FleeBearingsTests.cs`.

### Verification

- `just check plugin test-core FleeBearings` and `just check plugin test-core RouteProgress`
- `just check controller test-packages ./internal/retreat`
- `just check ci`

The live proof is goal 3.1's raised-base run, owned by TKT-01M62N256M.

## Acceptance criteria

- [x] A flee that moves less than about 1 m in 4 s with the threat within 15 m fails with a distinct reason and a trace naming the refused goal; a test covers it
- [x] The captured 2026-10-31 raised-base positions are a regression test that either finds a checked escape or fails fast with that reason
- [x] After a blocked flee the controller makes at most one more flee attempt per threat, with a different target; a test covers the bound
- [x] A hit from a different creature during a running flee re-aims the flee from that creature; a test covers it
- [ ] just check ci passes

## Implementation plan

Add a native physical-progress window that spans replans and fails within four seconds of less than one meter progress while a threat is close. Use checked escape bearings for ordinary flee tasks, retain finite bearing work, and allow one controller retry with a different destination after a blocked outcome. Re-aim a running flee once for a genuinely new confirmed attacker, preserving owner stop and matching escort/travel ownership. Record the captured raised-base and new remote-portal failures as regressions; live survival stays unearned.

## Notes

**agent:codex/home-survey-delivery** at 2026-11-03T16:22:40Z

Implemented native FleeProgress independent of replans: less than 1m flat displacement over 4s with threat within 15m fails blocked and traces actual position/threat/goal/gap. Ordinary flee now uses existing bounded checked bearings. Controller tracks its own task receipt and owner revision, permits one alternate tangent destination after blocked/unreachable/stuck, and can re-aim once from a genuinely new confirmed attacker. Newest damage takes priority over a closer older attacker. Two-attempt episode bound, native ownership and owner-stop/world dispatch races are covered. No remembered verified door/floor exit is available in the current retreat API; fallback is checked tangent guidance rather than guessed geometry.

Captured raised-base start [-33.9,36.09,6.68] and goal [-20.85,36.09,-2.57] are retained in FleeProgressTests with synthetic obstruction/jitter clearly labeled. just check plugin test-core Flee and focused controller regressions passed; later combined checks .checks/20261103T161946Z-AGIa9e.log and .checks/20261103T162143Z-4MDyZq.log. Full gate/review follow. Raised-base survival remains owned by the parent flee ticket, unearned.
