---
schema: 4
id: TKT-01M62N2557R401S3E6JECSAPN4
title: "M3: the bot survives danger and stands with its owner"
type: epic
status: in-progress
status_reason: null
priority: normal
due_on: "2026-11-15"
labels:
  - plugin
  - controller
assignees: []
milestone: M3
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-resume
  branch: t3/milestone-evidence
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 552c491e6b0ec2cac344cbb99a4e38a529ff19ef
  session: null
  claimed_at: 2026-10-30T13:45:00Z
  expires_at: null
archive: null
created_at: 2026-10-29T01:40:45Z
updated_at: 2026-11-02T16:02:27Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-plan
  name: ""
extensions: {}
---

## Description

M3 makes the bot safe to take out and useful in a fight. On 2026-10-28 it was killed twice by creatures while standing about and drowned once on a trip, and it never fought back or ran. M2 makes it work; M3 keeps it alive while it does, and lets it help.

The plugin's safety switch for this (Safety.AllowCombat) is off by default and stays that way until an owner turns it on.

## Acceptance criteria

- [ ] Hurt and outmatched, the bot retreats toward its owner or home instead of standing still
- [ ] With combat allowed, it fights what attacks it or its owner, and stops when told
- [ ] It remembers where it was hurt and avoids those places when it can

## Definition of done

- [ ] 3.1 Across all live trials from 11-03 to 11-15, just vm retreat-evidence shows >=3 consecutive low-HP episodes with a threat within 15 m where flee is dispatched within 1.5 s and the bot reaches its owner or home alive, and none that fails; one starts on the raised base (TKT-01M6EFHEEN, TKT-01M6EFHED7)
- [ ] 3.2 With AllowCombat on, 3/3 combat-trial runs that find a Greyling end in its observed death, and an owner stop (chat or panel) ends combat within 2 s in at least one (TKT-01M6EFHEG5); with it off, no attack is sent (MeleeTests)
- [ ] 3.3 Two of two escort or guard runs engage a confirmed attacker of a protected player within 10 m, and stop on 'stay out of it' with no further attack (escort-check, TKT-01M6EFHEHM)
- [ ] 3.4 A replay test shows routes home detouring around remembered danger zones (TKT-01M6EFHEK3), and one live trip detours around a zone still scoring >=1 (TKT-01M6EFHS5E)
- [ ] 3.5 Review tickets TKT-01M6DCFSEG and TKT-01M6DCFSG0 (with its three children) are done

## Implementation plan

Continue the owner-requested M3 completion. Danger memory/site preference/map layer is implemented and deployed. Native retreat, ground survey, scouting, kiting and gated melee exist. Obtain positive live melee/block/dodge and retreat evidence using bounded automatic supervisors; resolve actual failures without requiring owner staging or clearance. Then implement conservative self/owner defense selection with explicit targeting/damage evidence, species/equipment/health/stamina limits and immediate stop; preserve AllowCombat off by default. Do not infer winnability from hpFrac/stars alone or turn passive wildlife hostility into a fight. Route avoidance beyond target preference and broad milestone acceptance remain open.

## Notes

**agent:claude-code/renki-plan-m3** at 2026-11-02T15:58:34Z

### M3 grooming, 2026-11-02: goals to tickets

Each goal in docs/ROADMAP.md and the tickets that must finish for it. Live tickets depend on TKT-01M6EF5DTG (VM: exclusive lease for deploys and live trials).

- 3.1 retreat: TKT-01M62N256M (Plugin: flee task, and retreating when health is low), which depends on TKT-01M6EFHED7 (Flee: fail fast on a blocked escape and re-aim on a new attacker), TKT-01M6EFHEEN (VM: retreat-evidence report for goal 3.1 from an audit window) and TKT-01M638V1VY (Stuck recovery: don't strafe off a ledge).
- 3.2 fight and stop: TKT-01M62N2582 (Plugin: attack task with blocking and dodging), which depends on TKT-01M6EFHEBC (Retreat: an owner stop must not let a remembered hit restart combat) and TKT-01M6EFHEG5 (combat-trial: time an owner stop sent mid-melee).
- 3.3 defend the owner: TKT-01M62N25AD (Controller: guard the owner and defend itself), which depends on TKT-01M6EFHEBC, TKT-01M6EFHEHM (escort-check: report defences, owner distance and stop latency) and TKT-01M6CFDS6R (Protect charges without survival vetoes; opt in to auto logout). TKT-01M638V1XC (Follow: stuck outside a wall) supports it. Needs two owner sessions.
- 3.4 avoid where it was hurt: TKT-01M6EFHEK3 (Routes home detour around remembered danger zones) for the replay test, then TKT-01M6EFHS5E (Live: a trip home detours around a zone recorded that day). The roadmap names TKT-01M62EP0PD as owner; that M2 ticket is about landmark memory and does not cover routing.
- 3.5 containment: TKT-01M6DCFSEG (Behaviours: stow walks avoid hostiles; marker health baseline) and TKT-01M6DCFSG0 (Goals: lifecycle so stuck records never block new goals), now split into TKT-01M6EFHS9D, TKT-01M6EFHSCK and TKT-01M6EFHSFV.

Serving no M3 goal, proposed elsewhere (notes on each): TKT-01M6A62GWA, TKT-01M6A9EVRB, TKT-01M6AT4EFW (M5 household); TKT-01M6AYP8TQ (M5 or M7); TKT-01M6A8R2PA (possibly done, else M4); TKT-01M6BPPVDE (M7 after the in-flight PR); TKT-01M6CFDS8G, TKT-01M62VXZHF, TKT-01M62VXZJH (unscheduled or M7); TKT-01M656NE4C (optional, owner decision).
