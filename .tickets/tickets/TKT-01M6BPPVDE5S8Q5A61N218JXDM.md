---
schema: 4
id: TKT-01M6BPPVDE5S8Q5A61N218JXDM
title: Peel isolated enemies from groups with native pursuit checks
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - plugin
  - safety
  - movement
assignees: []
milestone: null
parent: TKT-01M62N25826TRP761PYVVC7RKY
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-milestones
  branch: t3/pantry-maintenance
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 2d6c3da73f244752302d0011ae2ebfdce9fe21aa
  session: null
  claimed_at: 2026-11-01T15:52:55Z
  expires_at: null
archive: null
created_at: 2026-11-01T14:02:42Z
updated_at: 2026-11-01T16:18:36Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

Build one bounded native group-peel episode following docs/WORLD-EVENTS.md: prepare equipment/food/escape, disengage, select a peripheral target, measure pursuit for the entire group, confirm isolation, fight one and reassess. A closer enemy is not necessarily isolated. Owner-authoritative targeting, event provenance and effective player-hunt mode are available in perception; remote/lost members remain uncertain. Groups that stay together need bounded escape/reposition instead of infinite lure or a larger melee crowd limit. Support Greydwarf-specific preparation/combat before testing the captured six-Greydwarf/three-Greyling group. Keep movement and combat in C#, persistent goal orchestration in Go, and diagnosis visible to agent tools and panel.

## Acceptance criteria

- [x] Preparation checks enemy support, gear, food, stamina, fighting ground and a checked escape.
- [x] Isolation uses temporal whole-group pursuit evidence and never treats remote or lost targets as disengaged.
- [ ] A second pursuer, fresh spawn, lost observation or blocked escape aborts safely under bounded work and retry budgets.
- [ ] One verified kill is followed by readiness and loot reassessment; checkpoints never replay uncertain attacks.
- [ ] Deterministic and natural live checks progress from one supported enemy to two before larger mixed groups.

## Implementation plan

Implement an explicit opt-in peel mode on native kite attacks, routed through the existing combat behavior and persistent combat goal. Reuse native scouting for discovery, then hand off one selected supported enemy to a bounded C# peel episode. Keep ordinary attack behavior and all melee bystander/footing gates intact.

Add a pure whole-group pursuit tracker with stable identities, fresh owner-authoritative target/hunt observations, movement evidence, permanent uncertainty for missing members, a continuous three-second isolation interval, and reset/abort on new or converging pursuers. Unknown targeting cannot certify disengagement. Expose tracking counts, phase, quiet evidence and planned escape in native combat status and traces.

Require active food, armour, weapon/armour durability, health/stamina and physically checked fighting/escape ground before drawing. Survey bounded routes away from the group; validate dry support, scenery, nearby characters and hostile crossing along navmesh paths. A native episode draws, leads, confirms isolation, hands off one melee fight, and monitors the whole group through combat. Group convergence, lost knowledge, unsafe routes or exhausted budgets cancels swings and uses native bounded escape. Observed death is the only kill proof; no task replays across restart.

Test deterministic single separation, a pack that stays together, late pursuers/new members, remote ownership, unloading, active hunt mode, blocked escape and readiness budgets. Add shared protocol and controller behavior/retreat/goal checks, then full CI/review and offline deployment. Natural one/two-enemy field checks remain conditional on actual preparation and a safe starting position. Greydwarf-specific combat and a larger mixed-group live pass remain follow-up work; do not log into the known crowd to claim them.

## Notes

**agent:codex/renki-milestones** at 2026-11-01T16:15:03Z

First native peeling increment implemented on stacked branch t3/native-peeling over event-awareness parent t3/pantry-maintenance. Explicit peel=true requires kite mode; the combat behavior, durable combat_trial goal, panel checkbox and just vm goal-trial --peel propagate it without controller steering. The goal remains one observed death and a separately verified home return; uncertain native intent is never replayed.

PursuitGroup tracks stable IDs, owner-authoritative target/hunt state, movement evidence, missing members and late arrivals. Fresh single-pursuer evidence must remain continuous for 3s, with nonselected living members known non-pursuing/non-hunting and >=24m away. Hunt mode alone cannot identify the selected enemy as pursuing Tas. The native episode checks food, armour, health, stamina, gear durability, supported enemy types, dry supported paths, grade continuity, bystanders and a checked escape before drawing. It revalidates escape from the current position and monitors the entire group through melee. Work is capped at60s/40m displacement, fallback escape at30s. Escape reassesses other threats when the selected threat disappears.

Focused core/native build, controller protocol/behavior/goal/retreat, VM supervisor and desktop/mobile peel-status checks passed. Full just check ci is running. Native field proof remains open: no login into the prior nine-attacker base capture, no group kill, no pantry field completion and no Greydwarf support claimed. Preparation now refuses unsupported/unready work; automatic acquisition/repair of missing gear and larger mixed-group combat remain additional work. Keep field ACs open until one- and two-enemy natural checks earn actual kill/escape/return evidence.

**agent:codex/renki-milestones** at 2026-11-01T16:15:33Z

Full just check ci passed (.checks/20261101T161219Z-bR2Uve.log): 52 desktop/mobile panel scenarios, controller race checks and 880 C# tests; native plugin build against game assemblies passed. Field ACs remain open; no native one-/two-enemy pass or Greydwarf readiness is claimed.

**agent:codex/renki-milestones** at 2026-11-01T16:18:36Z

First increment is committed as559a261 and available in draft PR148, stacked over PR147. Full CI and committed review passed; reviewed code reported no findings. Installed on BLIGHT with verified package checksums (.checks/20261101T161654Z-7qUckG.log), restarted independent panel and confirmed API version559a261 connected=false. AC1/2 have implementation and deterministic coverage; AC3/4/5 remain open for actual safe escape/kill/return, post-kill loot/readiness integration and progressive natural field trials. Tas remains offline after the prior nine-attacker base capture. No one-/two-enemy native pass, Greydwarf combat or pantry completion claimed.
