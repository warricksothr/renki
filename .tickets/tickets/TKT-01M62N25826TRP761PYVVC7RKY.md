---
schema: 4
id: TKT-01M62N25826TRP761PYVVC7RKY
title: "Plugin: attack task with blocking and dodging"
type: task
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
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
  actor: agent:codex/renki-resume
  branch: t3/melee-defense
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 459d71d029cf36af4895765c63bd76b1c5869507
  session: null
  claimed_at: 2026-10-30T00:56:00Z
  expires_at: null
archive: null
created_at: 2026-10-29T01:40:45Z
updated_at: 2026-10-30T01:10:58Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:codex/renki-resume
  name: ""
extensions: {}
---

## Description

`attack` (7.2): melee or ranged against an entity, with blocking and dodging, behind Safety.AllowCombat. Fails with target_lost when the target is gone, and stops on the dead-man rule like every body task.

## Acceptance criteria

- [ ] attack fights an entity until it dies or the task is stopped
- [ ] It blocks and dodges, and is refused unless AllowCombat is on

## Implementation plan

Implement the first explicit bounded melee attack against one unstarred Greyling (melee/auto style, ranged deferred), behind combat plus movement permissions. Add Go/C# args/fixtures, a testable core encounter policy, normal game aiming/weapon selection from combat damage, stamina reserves, attack-window block/dodge with supported dodge paths, complete input/queue cleanup, live target death versus loss, encounter leash/progress/time bounds, and plugin-owned flee fallback. Coordinate the existing retreat watcher with fresh world-scoped attack metadata in state so healthy accepted attacks are not canceled on every hit but new threats/unsafe state still escape. No autonomous combat or owner guarding in this slice. Test policy transitions, protocol parity, permissions, cancellation/dead-man, and retreat/attack arbitration. Run focused just check recipes, just check ci, commit and just check review before landing. Deploy with combat off for smoke checks, retain live staged Greyling criteria as pending until an owner is present to clear the encounter.

## Notes

**agent:codex/renki-resume** at 2026-10-30T01:03:57Z

Implemented explicit bounded melee attack (melee/auto styles) for one ordinary unstarred hostile Greyling behind combat and movement gates. Added Go/C# protocol support and shared attack/combat-state fixtures. Weapon selection uses character damage; normal aim/attack/block/dodge uses installed game stamina methods, supported movement and a validated 5m dodge corridor. Stops/changes clear held controls and queued attacks/dodges; existing animations finish normally. Bounds are 12m pursuit,45s work,12s without observed target-health progress; deteriorating conditions invoke a bounded normal flee whose outcome remains a failed encounter. Observed death is required; unloading is target_lost. Added task-ID-scoped combat metadata so controller retreat permits healthy explicit melee and an active fallback flee, while unsafe/stale state still permits escape. No autonomous target selection or owner guarding. Panel shows stamina/mode/reason; just vm combat reads/sets the separate plugin gate and requires restart for changes.

Focused protocol, permission, runner cancellation/dead-man/death/reconnect, stamina, eligibility, progress/leash and retreat-arbitration regressions pass. Plugin compiles against installed Valheim API. just check ci passed (.checks/20261030T010309Z-fV6JkT.log;235 plugin tests). A constructor argument-order build failure was diagnosed and corrected before this gate. Committed review and combat-disabled deployment smoke follow; live contact/block/dodge, staged single-Greyling and safe-return checks remain pending, so broad AC stay open.

**agent:codex/renki-resume** at 2026-10-30T01:09:38Z

PR #43 landed as dd3edda after the final just check ci pass (.checks/20261030T010444Z-Z7dLjj.log) and committed just check review with no findings (.checks/20261030T010519Z-mDc1nz.log). The final integration also defers food upkeep throughout attack, including fallback flee, and records combat interruption so reconnect returns home without restarting attacks. just check vm deploy passed with verified package hashes (.checks/20261030T010607Z-11nWh6.log).

Live permission smoke passed (.checks/melee-smoke-evidence.json): AllowCombat=false confirmed; added attack to controller allowedTasks and reconnected; t-1 attack reached the plugin and was rejected not_allowed with the plugin safety-config reason (.checks/task-240dx2tv.json). Used the nonliving stump id1:39765 for this disabled-gate check, so this is not a positive combat test. Tas stayed sheltered and idle. Controller permits explicit attack; plugin combat remains false. Native browser preview passed stamina, block-mode display, retreat reason and clearing ended combat, with zero runtime errors.

Tas rejoined without active meals (25/25 health), so food preparation is underway through normal chest withdrawal/automatic consumption, indoors. No encounter is being staged by the agent. Asked the owner for availability for one ordinary Greyling in open ground with immediate clearance available. Actual swing contact, blocking, dodge displacement, worsening-condition escape and post-encounter return remain unverified; broad acceptance criteria stay open. Autonomous self-defense, guard-me and ranged/other-creature profiles remain future slices.

**agent:codex/renki-resume** at 2026-10-30T01:10:58Z

Food preparation completed through normal withdrawals: Tas has active CookedDeerMeat and NeckTailGrilled plus one carried spare of each. Fresh panel status (.checks/melee-handoff-status.json) shows dd3edda, sheltered on structure inside home at [-37.59,36.09,6.11], 73/83.2 health and full 69.4 stamina. No combat task is active; plugin combat remains disabled while the owner staging question is pending. The next live check must refresh health, target identity and surroundings before enabling combat.
