---
schema: 4
id: TKT-01M6EYEC1M3CWJR3562ZXHJGYW
title: "Loot: anchor on the followed or commanding player, not any owner"
type: task
status: done
status_reason: null
priority: high
due_on: null
labels:
  - controller
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T20:15:36Z
updated_at: 2026-11-02T20:48:22Z
created_by:
  id: agent:claude-code/renki-wave1
  name: ""
updated_by:
  id: agent:claude-code/renki-loot-anchor
  name: ""
extensions: {}
---

## Description

The `Loot` instance (TKT-01M6EFHE2N, shipped in #165) is on by default and picks up drops within 30 m of *any* live owner it can see (`liveOwners` in `internal/behaviour/loot.go`). The owner wants it tied to one person: the player Tas is with.

### What to build

- Loot anchors on one player: the one Tas is following, or failing that the last player who gave Tas an order. That player must be an owner, or hold the `command` permission (an `operator` profile, `internal/access/access.go`). Companions and guests never anchor loot.
- Drops near other owners, outside the anchor's area, are left alone.
- With no anchor in sight, loot does nothing.
- The trace and the panel name the anchor player.

### Files

- `internal/behaviour/loot.go` (`liveOwners`, the eligibility walk), `internal/behaviour/engine.go` (`View`: who is being followed, who commands), `internal/access/access.go` (`Has`)
- `internal/session` for the current follow target
- `docs/BEHAVIOURS.md` (the loot paragraph)

### Verification

- Tests: two owners in sight, Tas following one, so only drops near the followed one are taken. An operator anchors; a companion does not. No anchor, no run.
- `just check controller test-packages ./internal/behaviour`, then `just check ci`.

## Acceptance criteria

- [x] Loot takes only drops in the anchor player's area, the anchor being the followed player or the last commander, and an owner or operator
- [x] Companions and guests never anchor loot, and with no anchor in sight loot does not run, covered by tests
- [x] docs/BEHAVIOURS.md describes the anchor; just check ci passes

## Implementation plan

The chat brain records the anchor when it sends an order (follow target, else the speaker; Brain.Anchor/Follows); the panel's follow and escort task buttons set it too. The engine passes it to View.Anchor. Loot's pick measures radius from the anchor, who must be in View.Owners (owners plus operators), alive and in fresh perception; with no anchor yet, a single commander in sight stands in. The deliberate-drop rule still uses every owner. A Summary hook lets loot record 'picked up N near X' as the run's last result, which the panel card shows.

## Notes

**agent:claude-code/renki-loot-anchor** at 2026-11-02T20:48:21Z

Done at 2852a8d. The chat brain records the anchor on each body order it sends: the follow or escort target, else the speaker if Tas can see them (panel chat from outside the world leaves it). Go home counts; food, gather and waypoint commands don't. The panel's follow and escort task buttons set it too. Loot measures its radius from the anchor, who must be in the commanders list (owners plus operators), alive and in fresh perception. Before any order, one commander in sight stands in. Summary hook: a run's last result reads 'picked up N near X'. Tests: TestLootWorksAroundItsAnchorOnly (7 cases), TestLootSummaryNamesItsAnchorAndCountsConfirmedPickups, TestAnchorIsThePlayerTheLastOrderServed, TestAVoiceEscortAnchorsOnTheEscortedPlayer, TestGoalCommandsAndUnseenSpeakersLeaveTheAnchor, the anchor column in TestOnlyACarriedOutOrderEndsTheHomeShun, and a panel follow in the app panel test. just check ci passed; just review clean after three rounds (voice escort, go home, unseen speakers). Not deployed.

## Summary

Loot works around one anchor player: whoever Tas last followed or escorted, or the seen player who gave its last body order, while they still command Tas. Companions, guests and absent anchors never anchor it. The panel card names the anchor after each run.
