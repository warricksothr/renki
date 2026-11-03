---
schema: 4
id: TKT-01M6FT1ZVWC237RH45JMCR8Z9J
title: "Follow: go through the portal the leader just used"
type: task
status: review
status_reason: null
priority: high
due_on: null
labels:
  - plugin
  - movement
  - live
assignees: []
milestone: null
parent: TKT-01M6864352N8E28YCGHA19AN90
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-03T04:18:10Z
updated_at: 2026-11-03T17:31:52Z
created_by:
  id: agent:claude-code/passive-deploy
  name: ""
updated_by:
  id: agent:codex/portal-follow
  name: ""
extensions: {}
---

## Description

Live, 2026-11-02 (build a892cfa): Nimah went through a portal while Tas was following. Tas stopped, the follow failed with `target_lost` ("Nimah is gone") after FollowTask's 5 s LoseAfter, and the home keeper walked Tas home. Follow knows nothing about portals, and ordinary movement is deliberately refused at portal triggers: the route planner (WorldNavigation, `portal_trigger_*`) and the last movement guard (`WorldNavigation.PortalClear`, called from Driver.cs:64).

This is a narrower case than TKT-01M6AZ7D (general portal traversal): the owner chose the portal, and Tas copies them.

### Proposed approach

1. **Notice the portal.** FollowTask remembers the leader's last-seen position. If the leader vanishes within about 2 m of a connected portal (TeleportWorld with a target), treat it as "went through", not "lost".
2. **Allow only that portal.** Walk into that one portal. A scoped allowance in the guard admits only its collider, for the rest of this follow, and clears on finish or cancel. Every other portal stays refused.
3. **Respect cargo rules.** The game refuses a player carrying non-teleportable items (TeleportWorld.Teleport, `$msg_noteleport`). If Tas stands in the trigger and isn't teleported within a few seconds, fail with a distinct reason and detail ("can't take my cargo through the portal"), not `target_lost`, and don't send Tas home silently.
4. **Wait out the trip.** The lose timer doesn't run while Player.IsTeleporting(). On arrival, drop the local obstacle caches, look for the leader with a fresh and somewhat longer timer, then continue following.
5. **Cover a pickup in progress.** A passive run cut short when the leader leaves restarts the follow after the leader has already gone. Keep each player's last-seen position plugin-side for a short while, so a follow started just after they vanished still finds their portal.

### Conflicts

TKT-01M6AYP8 (Cache building collisions and remember portal links) owns the portal guard, and agent:codex/renki-milestones has claimed it. Coordinate on WorldNavigation.cs and Driver.cs.

## Acceptance criteria

- [x] A leader vanishing at a connected portal sends Tas through that portal only; other portals stay refused
- [x] Cargo the game refuses fails the follow with a distinct reason, and Tas says so rather than walking home
- [x] The lose timer pauses while teleporting; after arrival Tas finds the leader and keeps following
- [x] A follow restarted just after the leader vanished (a passive run cut short) still takes the portal
- [ ] Live, under the VM lease: Tas follows Nimah through a portal pair and back

## Implementation plan

Implement portal following as a bounded native transition on the existing follow task. Keep short-lived per-player departure observations while any body task runs, including passive pickup; accept a restarted follow only with fresh evidence at a connected source. Detect disappearance and remote position jumps without inferring links from tags.

Add pure core departure/timing policy with tests for exact source/link admission, expiry, cargo refusal, teleport waits, arrival reacquisition, cancellation and return trips. The plugin adapter walks into the selected trigger through normal navigation and Valheim's own teleport logic. Revalidate the exact source/destination link at each step. Scope the trigger exception to the active physics step and clear it on cancellation, failure, teleport and task replacement. Keep all other collision, support and portal checks.

Clear local geometry after arrival and resume the same follow with a longer sight timer. Add distinct portal cargo/failure reasons and owner explanations that do not trigger the lost-target trip home. Document behavior and remaining live evidence. Existing collision-cache ticket TKT-01M6AYP8 (Cache building collisions and remember portal links) stays claimed by its current actor; changes here only thread the scoped allowance through its existing guards.

Run focused plugin and controller recipes, then just check ci, commit, and just check review. Leave the live portal-pair crossing and return criterion unticked until a supervised VM-lease session with the owner proves it.

## Notes

**agent:codex/portal-follow** at 2026-11-03T17:31:51Z

### Implementation and offline evidence

The owner selected this draft on 2026-11-03. Implemented on t3/continue-renki-bot without taking over TKT-01M6AYP8 (Cache building collisions and remember portal links) or touching the VM.

PortalLeaderTrail remembers a nearby connected source for eight seconds while any body task runs. A disappearance or remote position jump supplies departure evidence. GameTaskHost accepts a returned follow with that evidence, and FollowTask uses a bounded native crossing. Source/destination identity is revalidated; cached bounds, planning probes and the final input guard admit that source only. Planning admission is scoped to the walker call because Walker.Stand clears inputs during route validation; input admission resets every physics step. Cancellation, terminal failure and teleport revoke admission.

Valheim's trigger and teleport methods remain authoritative. Native cargo restrictions produce portal_cargo; entry/link/teleport/arrival failures produce portal_failed. Chat explains both without dispatching the lost-target trip home. Arrival clears geometry and resets follow movement. Approach, trigger refusal, teleport and arrival lookup have independent limits of 45, 4, 60 and 15 seconds.

Focused checks passed: just check plugin test-core PortalFollowTests; just check controller test-packages ./internal/chat ./internal/behaviour ./internal/protocol; just check plugin build (including the native plugin against installed assemblies). Shared fixtures cover both new reason strings. The full gate also covers the final source and fixtures; its result belongs in the PR validation.

A controller regression confirms that a leader disappearing during passive pickup returns the follow on the next fresh-frame tick, without waiting for the 30-second settle gap. Pure core tests exercise departure expiry, a retained source after a remote jump, exact source/link admission, cargo refusal, independent time limits, cancellation, a trip already underway, and return-trip policy. These earn the offline implementation criteria; they do not establish native trigger entry or actual arrival.

### Remaining game checks

Criterion 5 remains unchecked. Nothing was deployed in this session; read-only VM status found Valheim stopped, controller running and the lease free. With Nimah available, take just vm lease agent:tool/session and use just check vm deploy after review/merge. Follow through one connected portal pair and back; repeat the departure during passive pickup; verify native non-teleportable cargo is explained with no trip home; verify a stop/new order during approach wins and other portals remain refused. Retain audit/log events portal departure, portal arrived, portal failed and follow task outcomes. Release the lease after cleanup. docs/HANDOFF.md carries the session steps and prior passive-loot/lease checks still owed.

## Summary

Implemented scoped native portal following, recent departure memory for returned passive follows, bounded teleport/arrival handling and explicit cargo/failure messages. Offline policy, controller and shared-wire coverage is in place; native criterion 5 remains unchecked. No deployment in this session. See docs/PORTALS.md and docs/HANDOFF.md for the remaining supervised VM checks.
