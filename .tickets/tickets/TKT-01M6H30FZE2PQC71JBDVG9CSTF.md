---
schema: 4
id: TKT-01M6H30FZE2PQC71JBDVG9CSTF
title: React to hostile pursuit before the first hit
type: bug
status: in-progress
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - safety
  - movement
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/home-survey-delivery
  branch: t3/map-home-location
  worktree: /home/sothr/.t3/worktrees/renki/t3-b02dd361
  commit: 671e8d6a2e8b275a497a77165277fdfa3bec3fd4
  session: null
  claimed_at: 2026-11-03T16:14:07Z
  expires_at: null
archive: null
created_at: 2026-11-03T16:13:53Z
updated_at: 2026-11-03T16:34:55Z
created_by:
  id: agent:codex/home-survey-delivery
  name: ""
updated_by:
  id: agent:codex/home-survey-delivery
  name: ""
extensions: {}
---

## Description

Healthy Tas currently ignores a known hostile target until it deals damage. The owner observed approaching Greylings swinging while Tas stood idle. Begin bounded defense or retreat on a fresh known pursuit before contact; retain prepared single-Greyling native defense checks, active escort/travel ownership, real-attacker fallback and owner orders.

## Acceptance criteria

- [x] Fresh known targeting triggers protection at healthy HP before damage; existing native defense readiness and permissions remain authoritative.
- [x] Stale or unrelated hostile observations do not start protection; active native work and successful escape ownership are preserved.
- [ ] Focused regression checks, full CI and committed review pass; live pre-contact and survival evidence remain distinct.

## Implementation plan

React at any health to fresh known hostile targeting; permit the existing prepared, single ordinary Greyling native defense before damage and otherwise flee. Shorten watcher latency, validate real-session freshness, and preserve matching escort/travel/combat ownership and actual-hit fallback. Regress healthy pursuit, stale observations, readiness and explicit owner control. Native success is separate; the client was emergency-stopped during actual pursuit.

## Notes

**agent:codex/home-survey-delivery** at 2026-11-03T16:22:39Z

Implemented pre-contact protection for fresh known targeting within 15m at healthy HP. Prepared isolated ordinary Greylings can request native defensive melee before damage; permission/equipment checks remain authoritative. Otherwise retreat begins. Real-session state/perception freshness and world/connection scope are required, and observed dispatch checks owner revision atomically. Pending defense is protected from stale idle telemetry but cannot hide worsening health or another close attacker. Native escort/travel/combat and unrelated/manual flee retain ownership.

Focused regression checks passed in .checks/20261103T162143Z-4MDyZq.log. Additional pending-defense regression and final gate/review follow. This is policy evidence, not live survival evidence. Tas was emergency-stopped at 15.8/25 HP with no food and two known targeting Greylings roughly 5–9m away; .checks/new-home-defense-current.json and .checks/audit-window-VVadhq.jsonl. No new death was observed. Do not blindly rejoin that saved pursuit.

**agent:codex/home-survey-delivery** at 2026-11-03T16:26:58Z

Final review added explicit episode invalidation across death/respawn and connection changes, so an old escape/cooldown cannot mask fresh pursuit after recovery. Pending defensive admission also yields to another confirmed attacker when that AI target is unavailable. New regressions cover both cases.

**agent:codex/home-survey-delivery** at 2026-11-03T16:31:29Z

Bridge regressions now exercise healthy known pursuit before damage, remote damage with unknown AI targeting, and preservation of a pending manual flee before streamed task state arrives. just check controller test-packages ./internal/app ./internal/retreat ./internal/session ./internal/behaviour ./internal/chat ./internal/protocol passed12s in .checks/20261103T163044Z-wEhVgj.log.

The full gate previously exposed TestControlAPI racing its updated character frame: phase was already in_world for the greeting Bjorn, so its phase-only wait could return before Tasleroch arrived. The test now waits for both expected phase and character before asserting logout state. This is a test synchronization fix, not a production name change. Full gate is rerunning sequentially; the earlier concurrent-format npm failure is tooling interference, not validation evidence.

**agent:codex/home-survey-delivery** at 2026-11-03T16:34:55Z

Native review found and corrected the remaining damage-only entry gate in MeleeTask: DefensiveReady now accepts either a recent actual hit or a locally owned AI currently targeting Tas. Unknown/remote AI targeting alone cannot grant entry. Species/range, health, food, armour, stamina, weapon, bystander and combat permissions retain their existing checks. New native policy cases cover true/false/unknown targeting and withheld preparation before the first hit. just check plugin build passed10s in .checks/20261103T163419Z-yGICn2.log. A full gate that overlapped this edit saw mismatched old/new Core/plugin signatures and is not passing evidence; the serial native rebuild passed and final gate follows. No live encounter was staged from the known emergency pursuit.
