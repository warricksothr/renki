---
schema: 4
id: TKT-01M6CFDS6R71NG6WBY7HBPWTGP
title: Protect charges without survival vetoes; opt in to auto logout
type: task
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - plugin
  - controller
  - vm
  - live
  - safety
assignees: []
milestone: M3
parent: TKT-01M62N2557R401S3E6JECSAPN4
origin: null
dependencies:
  - TKT-01M6EF5DTGGAB15GFMRMDT8KTA
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-milestones
  branch: t3/valheim-plus-migration
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 30cdf96251c2f53ece801c98fdb426d1efebe89b
  session: null
  claimed_at: 2026-11-01T21:15:12Z
  expires_at: null
archive: null
created_at: 2026-11-01T21:14:39Z
updated_at: 2026-11-02T15:57:39Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m3
  name: ""
extensions: {}
---

## Description

The user explicitly wants Tas to take personal risks, including death, to protect Nimah and configured owners/operators. Use normal stamina costs, blocking/dodging, target evidence, bounded pursuit and friendly-fire protection; courage must not mean indiscriminate aggression. Remove the fixed 30 HP combat floor. Automatic logout hides failures and prevents learning: make recovery-failure logout and test-tool client shutdown opt-in, preserving telemetry and explicit manual stop/deployment restarts.

## Acceptance criteria

- [ ] Escort fights confirmed hostile attackers at low health and against groups without fleeing solely to preserve Tas
- [x] Automatic native recovery logout and test shutdown default off; explicit stop and deployment restart still work
- [x] Combat enabled and automatic logout disabled on Blight with observed configuration evidence

## Implementation plan

Introduce an internal escort-protection combat policy: confirmed hostile creatures including groups/stars, no personal health/food/armour veto, normal affordable attacks/blocks/dodges, bounded movement and bystander/structure safeguards. Prefer confirmed threats to charges over Tas-only threats and unknown attackers. Preserve native escort ownership against controller retreat, with short bounded retries instead of fleeing after a child refusal. Remove the absolute 30 HP floor from conservative self/travel defence too. Gate recovery logout with a separate default-off Safety.AutoLogout option, aborting failed recovery and releasing its input hold when disabled. Add explicit lifecycle flags to every live test harness, retain failed outcomes and combat permissions without closing the client by default. Add policy and failure-cleanup regressions, update current operator docs, run focused recipes and complete CI, deploy once with combat enabled and recovery/auto logout disabled, verify live state, commit and review. Record live combat evidence honestly and retain pending encounter checks.

## Notes

**agent:codex/renki-milestones** at 2026-11-01T21:24:58Z

Implemented internal escort protection with confirmed hostile targets across species/stars/groups, prioritizing threats to charges over Tas's own attackers, positive-life/weapon admission, actual affordable combat costs and no personal survival flee. Vulnerable bystanders and scenery hold fire; blocked routes/strikes and bounded encounter failures return to escort for reassessment, with no escape child. Exact escort task identity suppresses competing controller retreat even at 1 HP and against an additional Troll. Removed absolute 30 HP floors in conservative self/travel defence; those non-protection encounters retain relative reserves.

Native Safety.AutoLogout defaults false independently of RecoveryGuard. Failed startup healing/return emits the actual reason, aborts recovery and releases its input hold without ending the bridge or game; explicit/manual logout and deployment restarts are preserved. Combat/harvest/gather/goal/resupply, escort, cooking, food, marker-food, route, recovery and camera harnesses require --allow-logout for cleanup shutdown. Shared combat harness also requires --restart-client for setup restart/permission changes, and no longer restores combat off. Direct unfinished task cleanup checks exact ownership. Structured failure outcomes remain failures while the client is retained.

Validation: just check ci passed in 176 seconds (.checks/20261101T212127Z-waB8L2.log), including game-API plugin build, C#/Go policy/permissions/ownership tests, 46 desktop/mobile UI scenarios, all live harness simulations, and new injured cooking/food/camera lifecycle execution tests. Focused low-health/group protection, startup abort, route retention and combat cleanup tests passed. Initial focused travel test had the obsolete 30 HP expectation; updated to positively accept the 25/25 HP case before full CI compiled and tested it. Live deployment/configuration and actual owner-defence encounter evidence are still pending; leave AC1 and AC3 open.

**agent:codex/renki-milestones** at 2026-11-01T21:29:31Z

Committed implementation at 135c356. just check review HEAD~1..HEAD passed with no findings (.checks/20261101T212459Z-qAe54j.log). just check vm install passed with verified controller/plugin checksums (.checks/20261101T212530Z-4js4ge.log), followed by one explicit required game restart. Blight now runs controller/plugin source 135c356 with AllowCombat=true and Safety.AutoLogout=false; no cleanup restores combat off.

Live negative-path check: armed a bounded startup return while Tas was at home with 25/25 HP and zero active foods. Native trace at 2026-11-01T21:27:23Z recorded startup return refused (unsafe health, food, water or threat clearance), then startup recovery aborted; staying online with autoLogout=false. A fresh state remained connected/in_world at the same home position with no recovering hold, and inventory still held his equipped flint axe (76/100 durability) and rag armour. Evidence .checks/protection-policy/after-status.json, pack.json and audit-path.txt (native traces in the referenced audit). RecoveryReturn is cleared and RecoveryGuard=false is persisted for future launches; the current guard already completed its abort.

just check vm escort-check Nimah --seconds 15 --leave-running passed (.checks/20261101T212827Z-rqx7Yc.log): five observations, minimum25HP, mode wait, body t-1. This exercises actual native escort admission/permissions with no food and without the old 30 HP floor. Tas remains online, waiting for Nimah to enter loaded range; the tool did not log out or disable combat. This is not evidence of a fought encounter. Leave AC1 open for live low-health/group protection; field positioning around buildings and ranged combat remain future checks/work. Archery draft TKT-01M6CFDS8G4KSXAZ349R1PE3H0 remains queued for design/implementation.

**agent:claude-code/renki-plan-m3** at 2026-11-02T15:57:39Z

### Remaining for M3 goal 3.3

This ticket's open AC (escort fights confirmed attackers at low health and against groups) is live evidence of the same kind goal 3.3 needs, so it now sits under the M3 epic and TKT-01M62N25AD (Controller: guard the owner and defend itself) depends on it. Nothing else is left in code that I can find.

Close it on an escort-check run, scored by TKT-01M6EFHEHM (escort-check: report defences, owner distance and stop latency), that shows a defence started below the old 30 HP floor or with two attackers, with the charge within 10 m. A run that only shows a single healthy defence satisfies goal 3.3 but not this AC.

Files: `plugin/Renki.Plugin/EscortTask.cs`, `plugin/Renki.Core/Tasks/Escort.cs`, `scripts/vm/escort-check.py`.

Verify: `just check plugin test-core EscortTests`; live, `just vm escort-check <owner> --leave-running` with the VM lease (TKT-01M6EF5DTG).
