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
updated_at: 2026-10-30T01:26:52Z
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

Add a bounded closing phase to the existing explicit single-Greyling attack, accepting a named target up to 30 m away. Walk using supported navigation, keep the initial area bounded, require distance progress and cap closing at 12 seconds. At first entry to 12 m, anchor the existing encounter leash and start its combat progress/time budgets. Continue health, stamina, weapon, bystander and additional-threat checks throughout; no automatic target selection. Matching task-ID combat metadata distinguishes closing, allowing retreat to defer only to this healthy explicit target within the closing bound while preserving emergency escape for other threats or worsening conditions. Add deterministic closing/engagement boundary and retreat handoff regressions, compile against the installed game, run just check ci and committed just check review, land/deploy with combat disabled, then retry only with a controlled live staging opportunity. Record the separate repeated-flee stamina/roaming limitation without weakening escape to conceal it.

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

**agent:codex/renki-resume** at 2026-10-30T01:22:25Z

Owner-staged live attempt on 2026-10-30 UTC exposed an initiation gap, not a successful melee encounter. Tas rejoined fully fed at 80.5/80.5 HP with an equipped 92/100 flint axe. The ordinary Greyling was observed near home, then moved northeast. Normal move_to approaches completed, but confirmed hits triggered ordinary retreat before an explicit attack could begin. t-10 attack was rejected outside the 12 m encounter leash (.checks/task-zbbs7ljn.json). No combat traces, swings, blocks, dodges or kill were observed.

Exact evidence: .checks/audit-window-7K058z.jsonl and .checks/melee-range-watch.jsonl. Repeated ordinary escapes drained stamina to zero and moved Tas from home to around [174.18,84.73,-246.79], where another Boar hit for 9. Minimum observed HP was 68.1; final observed HP was 76.1/76.7. Logged out and just vm stop completed. Plugin combat config reset false for next restart. Owner was asked to clear threats, then explicitly chose to iterate on combat using this evidence. Tas remains offline; do not rejoin automatically into that encounter during development.

**agent:codex/renki-resume** at 2026-10-30T01:26:52Z

Implemented an explicit closing phase up to 30 m, bounded by 12 seconds, the initial 30 m area and 0.5 m distance progress every 4 seconds. At first entry to 12 m, anchor the encounter and reset combat budgets; all health/stamina/equipment/geometry checks remain active. Matching closing/task metadata coordinates retreat, including regression coverage for known damage, stale IDs, stronger/additional threats and worsening conditions. No autonomous combat selection was enabled.

The owner then requested replacing recurring manual test prerequisites with tools and loops. Added just vm combat-trial: restart/temporarily enable combat, check fed/rested state, select an observed safe single ordinary Greyling, send normal attack, retain task/status/perception/audit evidence, and always logout/stop/restore combat off. It aborts on another creature, task takeover, 40 m travel, low health or exhausted escape rather than requesting owner clearance. Selection and orchestration/cleanup checks are part of just check ci. A naturally unavailable encounter is aborted, never a positive pass; isolated-world fixture/reset remains separate follow-up. Focused policy/arbitration checks and just check ci passed before the documentation/ticket follow-up; final gate/review and deployed automatic trial follow. Tas remains offline at the saved northeast position.
