---
schema: 4
id: TKT-01M62N256MPAJ6YJC11XJS09RS
title: "Plugin: flee task, and retreating when health is low"
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - plugin
  - movement
  - safety
  - live
assignees: []
milestone: M3
parent: TKT-01M62N2557R401S3E6JECSAPN4
origin: null
dependencies:
  - TKT-01M6EFHED7E7YV0VWND0F2CYMJ
  - TKT-01M6EFHEENT3MJX2G80JMD60TA
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
  - TKT-01M638V1VYY93J6343W7CJAD2X
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-resume
  branch: t3/retreat-hauling
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: b75db8a8da95c954ca7dcef7e8b7a8db7ea02658
  session: null
  claimed_at: 2026-10-29T17:30:39Z
  expires_at: null
archive: null
created_at: 2026-10-29T01:40:45Z
updated_at: 2026-11-04T16:25:59Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:codex/portal-deploy
  name: ""
extensions: {}
---

## Description

`flee` (7.2): move away from an entity or a point to a distance. The controller uses it on low_health and when a creature targets the bot (perception's targetingMe), heading toward the owner or home rather than blindly away. Works without AllowCombat: running is always allowed.

## Acceptance criteria

- [ ] flee moves the bot away from a threat to a distance
- [ ] On low health with a threat near, the bot retreats toward its owner or home and says so
- [ ] Goal 3.1: three retreat-evidence episodes pass (flee within 1.5 s of HP below 0.4 with a threat within 15 m, owner or home reached alive), one starting on the raised base; paths linked in a note

## Implementation plan

Retain the existing fresh-attacker, cooldown, active-flee and explicit-combat arbitration. Code inspection found that threat selection, unlike active-combat arbitration, does not exclude zero-health creatures. A recently killed attacker can therefore start another flee while its corpse is still perceived. Require positive perceived creature health before selection, including when a fresh remembered hit exists. Exercise both AI-target and actual-hit evidence at zero versus positive health, retaining emergency retreat for a living attacker. Run focused retreat checks, CI and committed review, land and deploy after the active harvest trial cleans up. This deterministic correction does not establish live fleeing/blocking/dodging success; those checks remain open.

## Notes

**agent:claude-code/renki-groom** at 2026-10-29T15:34:30Z

Landed on t3/m2 and deployed on 2026-10-29. `flee` is now in the VM's allowedTasks, added through the panel settings.

- **Plugin:** FleeTask, which aims with Core Flee.Goal. It runs to the owner or home unless that lies past the threat, carries on to safety however far it is, stops at the owner's side, and runs on from home if the threat followed. Otherwise it runs straight away. It needs only AllowMovement.
- **Controller:** `internal/retreat`. Below 40% HP, with a hostile targeting the bot within 15 m, it flees toward an owner in sight (60 m) or home (150 m, and clear of the threat by 15 m), with 60 s per flee. It stays put at an owner's side, and says where it is going.

**Not seen in the game yet:** after the deploy the bot walked home unharmed, and the Greylings that killed it this morning had gone. Both criteria are in the code and tested (CoreTests and retreat_test), but stay unticked until a fight shows them.

**Why this was moved ahead of M2:** at 10:05–10:08 the bot died three times at home to a Greyling. I stopped it from the panel, and turned home.idleMinutes to 0 until the retreat was in, then back to 10.

**agent:codex/renki-resume** at 2026-10-29T17:30:39Z

claim taken from agent:claude-code/renki-groom by agent:codex/renki-resume

**agent:codex/renki-resume** at 2026-10-29T17:34:37Z

Fixed the missing-target trigger: Valheim's AI target is local to the creature's simulation peer, and BaseAI's replicated target information only says whether some target exists. The controller now also recognizes damage from a still-perceived hostile for 30 seconds, scoped to the world and cleared on death/respawn. Unrelated creatures, players, environmental damage, expired evidence and out-of-range/gone attackers do not trigger it. An owner nearby no longer suppresses retreat; reaching the owner with the threat still close continues the escape.

Verification: the regression reproduces the fatal 5.4/25 HP scene with the two real Greyling IDs and no targetingMe flags. A bridge integration test confirms that an actual damage frame reaches the retreat watcher and dispatches flee. just check controller test-packages ./internal/retreat ./internal/app, just check plugin build and just check ci passed. Live criteria remain open pending a controlled fight on the new build.

**agent:codex/renki-resume** at 2026-10-29T23:18:25Z

Live harvesting interruption 2026-10-29 23:09–23:10Z exposed the low-health-only trigger and retry cooldown: confirmed Greyling damage arrived while HP was still above 40%; automatic escape waited until health became critical. Escapes physically ran away but did not bring Tas into shelter; another Greyling on return reduced him to 2.7/25. Logged out and stopped both processes without an observed death; all seeds, axe and two wood retained. Owner subsequently cleared the route. Evidence .checks/audit-window-a69Odi.jsonl.

Changed trigger: a recent, currently perceived hostile attacker within 15 m now causes escape at any health. Targeting alone still requires the existing low-health threshold. A genuinely fresh hit can bypass the ten-second retry cooldown after a completed escape, while an active flee is never replaced by another flee. Added regression for healthy remote attacker without targetingMe, nearest unrelated AI targeting, stored-hit deduplication, new-hit retry, and in-flight escape preservation. Focused just check controller test-packages ./internal/retreat passed. Live post-fix safety/arrival still open; do not close this ticket from policy tests. Straight retreat's eventual return to shelter remains unproven.

**agent:codex/renki-resume** at 2026-10-30T00:39:51Z

Self-defense planning after the completed raised-home stump delivery: current retreat-on-confirmed-hit at any health must be coordinated with future attack selection, otherwise defense and flee will cancel each other. Do not remove fresh-hit fallback or active-flee protection while adding combat. One shared choice should permit only a well-understood single-attacker melee encounter and prefer escape for low health/stamina, multiple or unknown threats, failing equipment or unsafe geometry. Owner stay-out-of-it suppresses attacks without suppressing the existing escape routine. Owner-guard evidence is separate: current damaged events report Tas, not arbitrary owners. Concrete staged plan and verification are in docs/SELF_DEFENSE.md; no runtime combat changes or live combat tests in this planning pass.

**agent:codex/renki-resume** at 2026-10-30T01:09:38Z

PR #43 adds coordination for the explicit melee primitive: fresh state.combat must match the running attack task id before retreat defers to a healthy single-Greyling encounter or the plugin's active fallback flee. Unsafe health/stamina, additional attackers, unknown mode or mismatched metadata do not suppress ordinary retreat. Regression tests cover healthy confirmed hit versus worsening condition, extra attacker, stale id, stop and preservation of fallback flee. Food upkeep waits through the attack; a reconnect after interrupted combat returns home without resuming it. Deployed dd3edda with plugin combat disabled and verified the permission refusal; positive live combat/escape tests remain pending owner staging.

**agent:codex/renki-resume** at 2026-10-30T01:22:25Z

Owner-staged live attempt on 2026-10-30 UTC exposed an initiation gap, not a successful melee encounter. Tas rejoined fully fed at 80.5/80.5 HP with an equipped 92/100 flint axe. The ordinary Greyling was observed near home, then moved northeast. Normal move_to approaches completed, but confirmed hits triggered ordinary retreat before an explicit attack could begin. t-10 attack was rejected outside the 12 m encounter leash (.checks/task-zbbs7ljn.json). No combat traces, swings, blocks, dodges or kill were observed.

Exact evidence: .checks/audit-window-7K058z.jsonl and .checks/melee-range-watch.jsonl. Repeated ordinary escapes drained stamina to zero and moved Tas from home to around [174.18,84.73,-246.79], where another Boar hit for 9. Minimum observed HP was 68.1; final observed HP was 76.1/76.7. Logged out and just vm stop completed. Plugin combat config reset false for next restart. Owner was asked to clear threats, then explicitly chose to iterate on combat using this evidence. Tas remains offline; do not rejoin automatically into that encounter during development.

**agent:codex/renki-resume** at 2026-10-30T01:30:44Z

PR #45 landed as 6ba8e99. just check ci passed with 245 plugin tests, Go checks/builds and automatic-trial selection/orchestration/cleanup checks (.checks/20261030T012652Z-dugfxm.log). Committed just check review passed with no findings (.checks/20261030T012713Z-Sfe8sW.log). just check vm install verified the installed package (.checks/20261030T012748Z-dXEi4d.log).

Ran just check vm combat-trial --wait-s 0 without owner staging or clearance (.checks/20261030T012816Z-5MpagM.log). The runner correctly reported aborted/no fed, rested, safe single-Greyling encounter available, exit 1; this is not a passing combat test. Live preflight saw active meat meals, 76.1/76.6 HP, only 36.5 stamina (below its 40 minimum), a Greyling just outside 30 m at a lower elevation, and a Boar near that target. No attack command was sent. It automatically logged out, stopped the game and restored combat config false, with no cleanup errors. Evidence and full audit: .checks/combat-trial-20261030T012816512053Z/summary.json and .checks/audit-window-bU7lMF.jsonl. No swing/block/dodge result or successful live closing is claimed; acceptance criteria remain open.

Owner-independent live trial selection/recording/backstops now exist, but guaranteeing a positive repeatable encounter still needs a disposable local arena and reset. Filed draft TKT-01M656NE4C71PNAJMK025TB6FW (Repeatable isolated combat trials with automatic reset) for that remaining setup, preserving production saves. Tas remains offline at the northeast saved position; do not request owner clearance as the default next step.

**agent:codex/renki-resume** at 2026-10-30T02:19:02Z

PR #48 (684504b) adds native draw/kite/hold phases and matching-task retreat arbitration without suppressing low-health, stale-ID, extra-attacker or exhausted-stamina emergencies. Pure policy and integration checks passed under just check ci; committed just check review had no findings. Installed checksum verification passed.

The automated native trial (.checks/combat-trial-20261030T021553504234Z/summary.json; audit .checks/audit-window-qI99Fo.jsonl) failed safely during fighting-ground survey before acquiring an enemy or moving materially. Minimum HP was 76.1 with no observed damage, automatic shutdown/reset had no cleanup errors, and Tas is offline with combat false. This does not supply positive live flee/pursuit/defense evidence; acceptance criteria remain open. No owner staging or clearance was requested.

**agent:codex/renki-resume** at 2026-10-30T06:25:03Z

Require positive perceived creature HP during retreat threat selection. Previously a dead recently confirmed attacker or stale targeting flag could start another flee before its corpse vanished. Regression exercises zero versus positive HP under both targeting-only and recent-hit evidence. Existing living-attacker emergency, native-combat ownership and fallback-flee cases remain passing. Focused just check controller test-packages ./internal/retreat passed (.checks/20261030T062313Z-4HzY85.log); full just check ci passed (.checks/20261030T062338Z-PagRsJ.log). No new live flee/block/dodge evidence is claimed. Review/landing and deployment follow after the active harvesting runner finishes cleanup.

**agent:codex/renki-resume** at 2026-10-30T06:45:16Z

PR #63 merged23248bb; its dead-attacker threat correction is installed in cd08433. CI and committed review passed (.checks/20261030T062338Z-PagRsJ.log, .checks/20261030T062503Z-Z6z3Hp.log). Living actual attackers still trigger retreat; zero-health former attackers do not. No positive live flee/block/dodge is newly claimed. Latest harvested-area perception contained only Deer, with no eligible ordinary Greyling, so another unchanged combat trial was not run. Combat remains off by default; owner staging/clearance is not a development prerequisite.

**agent:codex/renki-milestones** at 2026-10-30T17:34:54Z

Unstaged return-home trial on 41110b0 supplied partial retreat evidence: Greyling 1:43163 hit Tas for 2.98 at 17:31:25Z near [-40.56,31.99,1.83]. At 17:31:26Z the controller replaced move_to t-1 with flee t-2 and said it was getting away. Native states show movement west/south to [-48.73,33.75,17.06], HP22/25, before the resupply supervisor logged out. This proves hit attribution, survival preemption and some movement; it does not prove a completed escape or low-health fallback. Evidence .checks/audit-window-V6dJ5V.jsonl. Existing retreat AC remains open.

**agent:codex/renki-milestones** at 2026-10-31T20:42:44Z

New live raised-base escape failure after successful food/cooking trials; no live follow-up is authorized by a safe route yet. Audit .checks/audit-window-SNFFE5.jsonl retains Greydwarf845473195:381 damage7.97 at2026-10-31T20:39:43Z. Automatic flee taskt-20 was dispatched/accepted/running within0.3s, goal[-20.85,36.09,-2.57] from[-33.9,36.09,6.68], with no safety destination. Tas remained nearly stationary inside the raised base. Greydwarf_Elite845473195:448 then inflicted27.73 at20:39:58Z; user emergency logout at20:40:00Z ended the attempt. Last native observation43.7/67.1HP at[-33.13,36.09,5.99], no death observed. Game stopped and independent panel restored; RecoveryGuard remains true. Do not rejoin blindly into the known base attack or request manual enemy clearing.

Next implementation should distinguish a physically blocked escape from an active successful retreat, retain actual egress/door/floor geometry, and react to a second stronger attacker rather than waiting for the original60s flee task. A bounded native emergency logout backstop on worsening injury/no progress can remove the human from this testing loop; it must preserve ordinary successful movement, controller stop, permissions and exact native diagnostics. This failure does not erase the separately earned successful startup return83.4m to the original sheltered home floor (.checks/recovery-20261031T203218Z).

**agent:claude-code/renki-plan-m3** at 2026-11-02T15:57:39Z

### Remaining for M3 goal 3.1

Goal 3.1 (docs/ROADMAP.md): in 3 of 3 runs where HP falls below the retreat threshold (`retreat.Low`, 0.4) with a threat within 15 m, `flee` is dispatched within 1 s and the bot reaches its owner or home alive. One run starts on the raised base that failed on 2026-10-31.

What is left:

- The raised-base failure in the previous note is not fixed. TKT-01M6EFHED7 (Flee: fail fast on a blocked escape and re-aim on a new attacker) owns the code change; this ticket now depends on it. Commit 9263f4b (checked escape bearings) landed afterwards and covers hostile-avoiding escape children only.
- TKT-01M638V1VY (Stuck recovery: don't strafe off a ledge) stops a recovery dropping the bot off the deck under the floor; also a dependency.
- Scoring: TKT-01M6EFHEEN (VM: retreat-evidence report for goal 3.1 from an audit window) turns an audit window into pass/fail per episode. Retreats happen inside other trials, so run it over the audit of every live trial this week and link the passing episodes here.
- Policy change since this ticket's ACs: TKT-01M6CFDS6R (Protect charges without survival vetoes) makes a native escort keep fighting at low HP, and `retreat.Tick` skips while a matching escort or native travel owns the body. Goal 3.1 therefore applies to runs that are not escorting.
- Live work needs the VM lease, TKT-01M6EF5DTG (VM: exclusive lease for deploys and live trials).

Files to start in: `internal/retreat/retreat.go`, `plugin/Renki.Plugin/FleeTask.cs`, `plugin/Renki.Core/Movement/FleeBearings.cs`.

Verify: `just check controller test-packages ./internal/retreat`, `just check plugin test-core FleeBearings`, `just check ci`; live, `just vm retreat-evidence AUDIT` over each trial's audit window, three passing episodes, one starting within the raised base (around [-33.9,36.09,6.68]).

**agent:claude-code/renki-plan** at 2026-11-02T16:28:21Z

Supersedes the goal wording in the "Remaining for M3 goal 3.1" note: goal 3.1 in docs/ROADMAP.md now counts natural low-HP episodes across all live trials from 11-03 to 11-15, scored by TKT-01M6EFHEEN, with flee dispatched within 1.5 s rather than 1 s, because the retreat check runs once a second. The rest of that note stands.

**agent:claude-code/renki-plan** at 2026-11-02T17:47:06Z

Supersedes the window in the previous goal 3.1 note: goal 3.1 no longer uses a calendar window (11-03 to 11-15). It counts natural low-HP episodes across all live trials run after the flee fix TKT-01M6EFHED7 is deployed. Milestone dates are now guideline targets, not deadlines (docs/ROADMAP.md).

**agent:claude-code/renki-plan** at 2026-11-02T17:49:45Z

Also supersedes "every live trial this week" in the earlier "Remaining for M3 goal 3.1" note: run the retreat-evidence report over every live trial since the flee fix TKT-01M6EFHED7 was deployed, with no calendar window.

**agent:codex/home-survey-delivery** at 2026-11-03T07:13:01Z

Incident observed during TKT-01M6FV6E40 (Home survey: familiarize with an area and update its places) deployment/admission validation.

Later live state supersedes the earlier water-position handoff. The audit records Greydwarf damage at 07:04:51.507Z (13.27) and 07:04:53.536Z (13.56) at [-167.59,28.74,-454.4]. Existing retreat dispatched plain flee t-1 within 0.2 s, toward saved home; it failed died at 07:04:53.561Z. Tas respawned at 07:05:11.674Z at [-31.08,36.55,15.19]. Existing after-death home handling then dispatched dry move_to t-4 toward [-204.39,32.97,-467.43]; it failed unreachable at 07:05:33.739Z after support/local-route refusals. No familiarize goal was created or dispatched. This was a two-hit low-supply water encounter, not a four-second blocked-flee proof or a survey-leg failure. Full evidence: .checks/audit-window-NgxmRf.jsonl; bounded report: .checks/20261103T071106Z-WhOy9o.log. No survival acceptance is earned. Record this under the existing flee/survival work; no claim takeover or implicit combat/supply permissions were made.

**agent:codex/home-survey-delivery** at 2026-11-03T15:34:57Z

New-home continuation, without taking this ticket's claim: the operator opened storage entrance gates for access and left them open while searching chests for food. Greydwarf2600298615:830 entered and dealt11.75 at15:28:09.117Z and12.78 at15:28:11.282Z, leaving0.5/25HP. Native audit .checks/audit-window-OzbhE9.jsonl retains the exact window. Automatic flee t-25 was dispatched at15:28:09.826Z. A subsequent direct task-check chest probe t-27 replaced that already running flee at15:28:11.609Z, after both hits. Automatic retreat resumed as t-28 at15:28:11.822Z, cancelled t-27, then failed dry-bank distance_limit at15:28:21.345Z near[-184.9,30.97,-469.7]. This is not a death or a wading-task injury. The direct probe's0.22s interruption and leaving gates open are operator/tooling problems to preserve honestly, not evidence that retreat itself caused either hit.

Checked home travel refused hostile corridors and stair support (.checks/task-e306cp1b.json). Ordinary native t-31 then reached saved home; t-34 returned to its interior point. Final grounded/sheltered state[-203.58,32.97,-466.37],0.5HP, no food, idle, house entrance1:14767 confirmed closed. .checks/new-home-final-observation.json and .checks/new-home-final-map.json. Do not send another exposed trial at this health. No survival acceptance is claimed; retain the existing assignee/claim.

**agent:codex/home-survey-delivery** at 2026-11-03T15:52:42Z

Later runtime supersedes the earlier sheltered-house0.5HP snapshot. The owner supplied a Raspberry at15:33:47Z; automatic use_item t-36 restored health. Existing passive timer Loot left the house for Wood t-37, then chased Resin t-38. It remained active after repeated final portal guard refusals, escalated Jump/Strafe and entered darkforest at15:36:32Z. Audit .checks/audit-window-FfbfFj.jsonl. This was not a survey or an explicit relocation order. Loot was disabled persistently through the behavior API, preserving its definition.

TKT-01M6H1A2XPHCAR4F016AE4F8K0 (End the body task when the final portal guard refuses movement) is now fixed/deployed in cc5f323, with focused lifecycle/full gate/review passing and an actual native same-step unreachable receipt at the first guard refusal (.checks/task-gcwg1_fw.json). Post-settling state .checks/new-home-portal-fix-after.json is grounded/idle on structure at[414.42,31.65,-1176.15],25/25HP, no food, injurySeq0 in the restarted client. The earlier storage hits are historical; no new injury occurred in this probe. No portal entry allowance was introduced. The observed linked return portal is1:161078 ->1:161081 near home, but controlled traversal/return remains draft TKT-01M6AZ7DJYGTWDR9G9ZP0MA20M (Traverse known portals with verified arrivals).

Earlier ordinary native arrival at new home and the saved/bound new home storage west approach remain earned. Tas is currently940m away on the remote portal platform. All four observed beds near the new base were owned, none mine; no bed was claimed. No survey goal was admitted, no full mapping or checked round trip was earned, and native survey criteria5/7 remain unticked. Preserve other agents' claims. Current blockers are return capability, meal readiness and a free Tas bed/verified sleeping approach. Do not re-enable Loot or reverse newer owner orders without checking live state and intent.

**agent:codex/home-survey-delivery** at 2026-11-03T16:00:45Z

Supersedes the healthy remote-platform snapshot in the preceding note. On deployed cc5f323, Greydwarf 1629975892:102 hit at 15:54:52.701Z (11.99 damage), 15:54:54.821Z (12.28) and 15:55:04.879Z (11.44). Automatic flee t-2 started within 0.28s of the first hit, failed unreachable at 15:54:55.186Z; replacement flee t-5 remained active until death at 15:55:04.915Z near [413.36,31.65,-1177.66]. No direct probe interrupted these retreats. Respawn at 15:55:22.962Z was at the old base [-31.08,36.55,15.19], followed by automatic strict-dry return t-7, which failed unreachable at 15:55:44.553Z. Latest resumed observation at 15:56:28Z: idle 25/25HP near [-46.32,33.79,10.13], no active/carried food, empty pack and no equipped gear; fresh local perception had no characters. Loot remains disabled. Exact retained events/states/inventory: .checks/audit-window-M3BtxI.jsonl; fresh state/map .checks/new-home-resume-runtime.json and .checks/new-home-resume-map.json. This is a death/failed escape, not a safe portal arrival or return, and does not earn survey, bed or survival acceptance.

**agent:codex/home-survey-delivery** at 2026-11-03T16:27:38Z

Current emergency supersedes prior healthy/online snapshots. Fresh observation at about16:12Z showed Tas at[150.65,70.48,-33.81],15.8/25HP, no food, stamina34.9, injurySeq8679, with flee t-20 running and two known targeting Greylings5–9m away (2494842417:2243 and1:100792). The operator explicitly ran just vm stop to prevent another death, then just vm panel-start to restore the independent panel without rejoining. No new died event was observed; menu HP0 is not a death receipt. Evidence .checks/new-home-defense-current.json, .checks/audit-window-VVadhq.jsonl, and read-only offline verification .checks/new-home-offline-verification.json.

TKT-01M6H30FZE2PQC71JBDVG9CSTF (React to hostile pursuit before the first hit) implements fresh known targeting protection before damage. TKT-01M6EFHED7E7YV0VWND0F2CYMJ (Flee: fail fast on a blocked escape and re-aim on a new attacker) adds physical progress, checked ordinary escape bearings, one different-destination retry and new-attacker re-aim with owner-order/episode bounds. Regression checks are passing; full gate/review/install follow. No native survival acceptance is earned, and this ticket's existing claim is preserved. Do not blindly rejoin the saved pursuit or interrupt active recovery with a probe.

**agent:codex/portal-deploy** at 2026-11-04T16:25:59Z

On November 4, the owner restored the signed-in console and Tas restarted on deployed 501ca8d (just check vm start passed, .checks/20261104T155916Z-waASAP.log). Empty inventory, no active/carried food. Fresh normal read-only searches of outdoor chests 1:75525, 1:75526 and 1:77215 and house pantry 1:75501 found no ready food. Pantry now holds fish; historical mushrooms were absent. Northern berries were refused due Elite clearance. A southern mushroom attempt ended in death: Greydwarf 2675073813:273 approached, bank retreats t-18/t-20 and t-23/t-25 failed bearings_exhausted, damage at 16:12:35.238Z and 16:12:41.296Z, death at 16:12:41.322Z near [-11.08,30.8,44.77]. No food was collected. Audit .checks/audit-window-CReSWs.jsonl retains exact events. This supersedes earlier healthy/staged snapshots; neither forage nor retreat succeeded. The owner was informed; the agent should have abandoned the source as soon as the Greydwarf appeared.

Respawn was 25/25HP at old bed. Stopped automatic home travel; persisted home.idleMinutes=0 and home.afterDeath=stay (afterOutage=false retained) to prevent another unsupplied 500m trip. RecoveryGuard/AutoLogout remain false; combat/gather permissions retained; Loot, Relocation mushrooms and Relocation food search disabled. Two bounded shelter returns failed without injury, leaving Tas at [-43.39,33.73,8.53], idle, grounded, 25/25HP with empty pack. Latest fresh perception at 16:24Z showed a non-targeting Boar ~30m away. Old double gates are open after these failed returns; closed-state observations from the earlier pantry inspection do not establish final door restoration.

The owner says nearby SpawnCamp portal 1:166576 leads to the new base and cannot supply old-base food. Fresh native link is connected to 1:166579 at source [-50.26,34.09,13.84], without remote coordinates. Explicit ordered crossing is being added under TKT-01M6AZ7DJYGTWDR9G9ZP0MA20M (Traverse known portals with verified arrivals). No crossing, free bed, new-home food or survey completion is claimed yet. Saved home remains [-204.39,32.97,-467.43]. Keep Tas online except required implementation deployment restart.
