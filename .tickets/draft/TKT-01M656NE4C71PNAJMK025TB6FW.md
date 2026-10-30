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
assignees: []
milestone: M3
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-30T01:26:52Z
updated_at: 2026-10-30T01:26:52Z
created_by:
  id: agent:codex/renki-resume
  name: ""
updated_by:
  id: agent:codex/renki-resume
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
