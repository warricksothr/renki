---
schema: 4
id: TKT-01M6CFDS8G4KSXAZ349R1PE3H0
title: Teach Tas archery and tactical ranged weapon selection
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - controller
assignees: []
milestone: null
parent: TKT-01M62N25ADM0Z968D974G0CHP2
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-01T21:14:39Z
updated_at: 2026-11-02T16:01:43Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:claude-code/renki-plan
  name: ""
extensions: {}
---

## Description

User requested archery as a core combat skill, particularly suitable for Tas. Extend native combat with ordinary-game bow inputs, observed target motion and aim elevation, clear trajectories, draw timing, ammunition type/count, and hit/miss feedback. Select ranged versus melee based on range, line of fire, threat to protected players, resources and terrain; avoid wasting shots into cover or firing through vulnerable players. Start with a bounded explicit target encounter before integrating escort protection, hunting and kiting. Include diagnostic traces and an autonomous repeatable acquisition/test workflow that keeps the client online after failures by default. No spawned gear, aim cheats or invented remote target knowledge.

## Acceptance criteria

- [ ] Uses normal bow draw/release, stamina and owned ammunition with measured hit evidence
- [ ] Chooses clear shots with protected players and vulnerable bystanders accounted for
- [ ] Repositions and switches to melee when range, cover or ammunition require it
- [ ] Reports aim, ammunition, target choice and failure reasons for repeatable testing

## Notes

**agent:claude-code/renki-plan-m3** at 2026-11-02T15:58:15Z

### Milestone review: not an M3 goal; too big for one PR

M3 grooming, 2026-11-02: archery serves none of goals 3.1-3.5 (3.2 and 3.3 are melee). Proposed: M7 or unscheduled, after the melee goals pass. Before an agent picks it up, split it into: native bow draw/release and owned ammunition with hit evidence against an explicit target; line-of-fire and protected-player checks; ranged/melee selection and repositioning; then escort and hunting integration. Start in `plugin/Renki.Plugin/MeleeTask.cs` and `plugin/Renki.Core/Tasks/Melee.cs` (weapon choice and inputs), `plugin/Renki.Plugin/Driver.cs` (inputs), and the `attack` task `style` in docs/BRIDGE-PROTOCOL.md. Live checks need the VM lease (TKT-01M6EF5DTG).
