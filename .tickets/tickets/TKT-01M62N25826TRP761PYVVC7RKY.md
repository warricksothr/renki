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
updated_at: 2026-10-30T02:31:36Z
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

Owner correction after the first native survey: ordinary terrain should be usable when its grade is reasonable and it has no sharp regions; flat ground is not required. The existing six-metre radial probes incorrectly apply the movement single-step height limit to the entire radius, and dodge clearance also caps total elevation change. Replace those aggregate height checks with half-metre supported corridor samples that bound local slope and adjacent slope changes. Use an initial 30-degree maximum inclination (below the installed player sliding threshold of 38 degrees) and 15-degree maximum change between adjacent samples. Continue validating physical body clearance and dry ground, and retain all existing encounter budgets and bystander gates. Reuse the profile validator for fighting-ground and dodge corridors, with regressions for long uphill/downhill slopes, smooth grade changes, steep ground, sharp ridges/dips, unsupported/blocked segments and nonfinite data. Report per-candidate ground rejection reasons for live diagnosis. Validate, review and land, install offline, then run another normal-controls automated attempt without owner staging. Keep live pursuit/strike/defense criteria open until observed.

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

**agent:codex/renki-resume** at 2026-10-30T01:30:44Z

PR #45 landed as 6ba8e99. just check ci passed with 245 plugin tests, Go checks/builds and automatic-trial selection/orchestration/cleanup checks (.checks/20261030T012652Z-dugfxm.log). Committed just check review passed with no findings (.checks/20261030T012713Z-Sfe8sW.log). just check vm install verified the installed package (.checks/20261030T012748Z-dXEi4d.log).

Ran just check vm combat-trial --wait-s 0 without owner staging or clearance (.checks/20261030T012816Z-5MpagM.log). The runner correctly reported aborted/no fed, rested, safe single-Greyling encounter available, exit 1; this is not a passing combat test. Live preflight saw active meat meals, 76.1/76.6 HP, only 36.5 stamina (below its 40 minimum), a Greyling just outside 30 m at a lower elevation, and a Boar near that target. No attack command was sent. It automatically logged out, stopped the game and restored combat config false, with no cleanup errors. Evidence and full audit: .checks/combat-trial-20261030T012816512053Z/summary.json and .checks/audit-window-bU7lMF.jsonl. No swing/block/dodge result or successful live closing is claimed; acceptance criteria remain open.

Owner-independent live trial selection/recording/backstops now exist, but guaranteeing a positive repeatable encounter still needs a disposable local arena and reset. Filed draft TKT-01M656NE4C71PNAJMK025TB6FW (Repeatable isolated combat trials with automatic reset) for that remaining setup, preserving production saves. Tas remains offline at the northeast saved position; do not request owner clearance as the default next step.

**agent:codex/renki-resume** at 2026-10-30T01:50:53Z

Start implementation of the approved native kiting plan. Extend attack with kite=true and optional preferred at point; target omission in this mode delegates bounded local Greyling acquisition to C#. Preserve existing explicit melee args. Add testable kiting phases, observed follower movement, stamina/gap pacing, bounds and failure reasons. Add native area validation, target acquisition, navigation and direct MeleeTask handoff under the same attack ID; report phases and coordinate retreat for the selected target. Update the trial harness to send one native goal rather than select enemies. Validate policy, protocol, cancellation, retreat integration and supervisor ownership/cleanup. Land after just check ci and committed just check review; install with combat disabled. Positive live kiting/strike/defense checks remain open until observed.

**agent:codex/renki-resume** at 2026-10-30T02:10:06Z

Implemented attack kite=true (optional target and preferred at point), a native C# survey/position/search/draw/kite/hold/engage routine, and testable pursuit/pacing policy. It validates dry supported clear terrain and escape corridors, moves to checked ground, acquires one ordinary Greyling locally, observes actual movement toward Tas, preserves separation/stamina, bounds preparation/search/luring and hands directly to MeleeTask with the same attack ID. It revalidates ground and characters and retains cancellation/escape paths. No forced enemy AI or frame-by-frame controller movement orders.

Added a world-scoped combat behavior template that submits one goal and waits; active goal edits use the existing cancel/stop acknowledgement before a replacement starts. Added protocol fixture/argument validation, native-phase retreat arbitration and lifecycle/policy/behavior regressions. Python trial runner now delegates acquisition and kiting to C# and only supervises/records/stops. Focused policy, behavior/protocol/retreat and supervisor checks pass, and the plugin builds against installed game APIs. An initial full CI build caught a C# local-variable shadow after the positioning step was added; renamed it and verified the plugin build. Final CI, committed review and installed trial follow. Live attraction, pursuit, pacing and melee handoff remain unverified; broad acceptance criteria stay open.

**agent:codex/renki-resume** at 2026-10-30T02:19:02Z

Native acquisition/kiting implementation landed in PR #48 as 684504b. just check ci passed (275 plugin tests plus controller race/check/build and trial-supervisor checks; .checks/20261030T021052Z-bAmoqJ.log). Committed just check review passed without findings (.checks/20261030T021427Z-i6YNUI.log). just check vm install verified deployed controller/plugin checksums (.checks/20261030T021537Z-mlweZs.log).

Ran just check vm combat-trial --wait-s 60 with no owner staging, clearing or repositioning (.checks/20261030T021553Z-8sQ8dt.log). C# accepted one attack kite=true goal, surveyed its bounded local candidates and failed unreachable after 8.6 seconds: no supported clear fighting ground with a checked escape corridor. Tas remained near [174.17,84.75,-246.72]; preflight stamina was 67.2, minimum HP 76.1, and no damage was observed. No enemy was acquired and no positive pursuit/strike/block/dodge evidence is claimed. The supervisor correctly returned failure, logged out/stopped, restored combat false and reported no cleanup errors. Summary: .checks/combat-trial-20261030T021553504234Z/summary.json; exact audit: .checks/audit-window-qI99Fo.jsonl.

Keep both broad acceptance criteria open. Next native iteration should expose per-candidate ground rejection reasons and use supported local approach/escape corridors to choose suitable nearby ground on ordinary sloped terrain. Preserve normal controls and bounded resource/time/travel exits; do not require owner clearance or pretend this survey refusal proves a successful encounter. Tas is offline with combat disabled.

**agent:codex/renki-resume** at 2026-10-30T02:26:48Z

Replaced the aggregate elevation restriction with a shared GroundCorridor validator: half-metre physical support/clearance samples, maximum 30-degree inclination and 15-degree adjacent inclination change. Four complete diameters cover the fighting area's center so a sharp crest/dip there is not missed. The melee dodge corridor now uses the same local profile limits instead of a 0.5m total elevation ceiling. Dry ground, structures, characters and navmesh approach checks remain, and failed candidates emit location/attempt/reason traces. The installed Character implementation starts player sliding at 38 degrees, so the 30-degree cap leaves margin.

just check plugin test-core GroundCorridorTests passed 12 meaningful cases for sustained uphill/downhill elevation, gradual grade changes, steep slopes, sharp crests/dips/ledges, unsupported/blocked intermediate ground, character-foot height offset and invalid/unbounded data (.checks/20261030T022447Z-8ZaF9n.log). just check ci passed with 287 plugin tests plus controller and supervisor suites (.checks/20261030T022558Z-OXvjlk.log). Committed review, installation and another automated live attempt follow. No positive combat acceptance criterion has been checked.

**agent:codex/renki-resume** at 2026-10-30T02:31:36Z

The grade/roughness correction landed in PR #49 as 33cd427. Committed just check review had no findings (.checks/20261030T022648Z-m8CAb6.log), and just check vm install verified package checksums (.checks/20261030T022743Z-xwMUT9.log).

Ran a second automated normal-controls attempt with just check vm combat-trial --wait-s 60 (.checks/20261030T022800Z-9GPeqv.log). One native attack surveyed 34 candidates and returned unreachable without acquiring an enemy. Rejection traces distinguish 16 grade, 8 sharp_change and 10 blocked_or_unsupported candidates; this run did not find an eligible fighting area. Those counts describe observed policy refusals, not proof every candidate is physically unusable. GroundCorridorTests separately establish that gradual slopes can accumulate several metres of elevation and that the flat-ground restriction is gone.

Tas remained around [174.15,84.73,-246.79]. Minimum observed HP was 75.8 with HP following the food-derived maximum and no observed damage; stamina stayed near 67. Automatic logout, stop and combat=false restoration completed with no cleanup errors. Summary: .checks/combat-trial-20261030T022800895969Z/summary.json. Audit with candidate reasons: .checks/audit-window-df13eN.jsonl. No positive pursuit, contact, block or dodge is claimed. Acceptance criteria remain open. Future ground-selection work can inspect rejected sample locations and actual collider/terrain causes rather than adjusting limits blindly; no owner staging or clearance is required by the harness.
