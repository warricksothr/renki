---
schema: 4
id: TKT-01M6EFHED7E7YV0VWND0F2CYMJ
title: "Flee: fail fast on a blocked escape and re-aim on a new attacker"
type: bug
status: draft
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

- [ ] A flee that moves less than about 1 m in 4 s with the threat within 15 m fails with a distinct reason and a trace naming the refused goal; a test covers it
- [ ] The captured 2026-10-31 raised-base positions are a regression test that either finds a checked escape or fails fast with that reason
- [ ] After a blocked flee the controller makes at most one more flee attempt per threat, with a different target; a test covers the bound
- [ ] A hit from a different creature during a running flee re-aims the flee from that creature; a test covers it
- [ ] just check ci passes
