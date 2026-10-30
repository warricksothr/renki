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
  branch: t3/retreat-hauling
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: b75db8a8da95c954ca7dcef7e8b7a8db7ea02658
  session: null
  claimed_at: 2026-10-29T17:30:39Z
  expires_at: null
archive: null
created_at: 2026-10-29T01:40:45Z
updated_at: 2026-10-30T17:34:54Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

`flee` (7.2): move away from an entity or a point to a distance. The controller uses it on low_health and when a creature targets the bot (perception's targetingMe), heading toward the owner or home rather than blindly away. Works without AllowCombat: running is always allowed.

## Acceptance criteria

- [ ] flee moves the bot away from a threat to a distance
- [ ] On low health with a threat near, the bot retreats toward its owner or home and says so

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
