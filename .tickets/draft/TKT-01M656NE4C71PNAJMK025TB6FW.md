---
schema: 4
id: TKT-01M656NE4C71PNAJMK025TB6FW
title: Repeatable isolated combat trials with automatic reset
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - tooling
  - vm
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
created_at: 2026-10-30T01:26:52Z
updated_at: 2026-11-02T17:47:06Z
created_by:
  id: agent:codex/renki-resume
  name: ""
updated_by:
  id: agent:claude-code/renki-plan
  name: ""
extensions: {}
---

## Description

The live combat trial on 2026-10-30 depended on owner staging and clearance. The owner explicitly asked that recurring manual prerequisites become tools, behaviors and automated test loops. The initial just vm combat-trial runner selects a safe naturally observed ordinary Greyling, records the encounter, bounds travel/health/stamina and always stops/restores combat off; it cannot guarantee a positive encounter or reset Tas after an aborted chase.

Develop a disposable local test profile/world with repeatable encounter setup and reset. Keep production Tas/world inventory and buildings outside the reset boundary. Run the actual plugin against installed Valheim controls and physics, arranging one ordinary Greyling and supplied test equipment/food inside that disposable world. Automate startup, encounter, evidence collection, failure capture, cleanup and reset without asking the owner to stage, clear, feed or reposition. Report actual contact/block/dodge/escape observations separately; policy passes alone do not prove a game interaction. Establish a supported local hosting/startup method before choosing how to create fixtures. Do not expose a general-purpose production-world spawn/teleport endpoint as a shortcut.

## Acceptance criteria

- [ ] One recipe creates or restores a disposable local test profile and world
- [ ] The real plugin completes an ordinary Greyling trial without owner staging
- [ ] Failures retain bounded evidence and reset without changing production saves
- [ ] Reports distinguish strike, block, dodge and escape observations from policy checks

## Notes

**agent:codex/renki-resume** at 2026-10-30T01:44:10Z

Owner challenged isolated-world staging as the next priority and instead selected finding an enemy and kiting it back to suitable fighting ground as native C# behavior. This draft remains optional future tooling; it is not a dependency for combat progress. Prioritize a reusable C# encounter routine with bounded local acquisition, observed pursuit, stamina-aware distance pacing, checked fighting ground, direct melee handoff and emergency escape under one body task. The controller/trial harness supplies goals and records outcomes rather than puppeting each movement. The existing runner's Python-side target selection is an interim tool, not the intended production architecture. See the revised plan on TKT-01M62N25826TRP761PYVVC7RKY (Plugin: attack task with blocking and dodging).

**agent:claude-code/renki-plan-m3** at 2026-11-02T15:58:21Z

### Milestone review: optional; owner decision

M3 grooming, 2026-11-02: the roadmap keeps this optional, and the owner chose native encounters over isolated staging on 2026-10-30. It is the only way listed to make goals 3.1 and 3.2 schedulable on demand: today a low-HP retreat (3.1) and a suitable Greyling (3.2) depend on what the production world offers. If the 2026-11-09 weekly review shows fewer than three qualifying episodes for 3.1, the owner should decide between this ticket and counting natural episodes over a longer window.

Not ready for an agent: it needs a supported local hosting/startup method chosen first (a spike), and it is larger than one pull request. Live work needs the VM lease (TKT-01M6EF5DTG). Files to start in: `scripts/vm/vm.just`, `scripts/vm/start-bot.ps1`, `scripts/vm/combat-trial.py`, `docs/VM.md`.

**agent:claude-code/renki-plan** at 2026-11-02T17:47:06Z

Supersedes the "2026-11-09 weekly review" trigger in the M3 grooming note. The owner decides on this isolated world at the first review (docs/ROADMAP.md, Cadence) where too few qualifying episodes for goal 3.1 have occurred while M3's other goals are met.
