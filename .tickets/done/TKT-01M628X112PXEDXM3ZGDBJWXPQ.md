---
schema: 4
id: TKT-01M628X112PXEDXM3ZGDBJWXPQ
title: Set a home and go back to it when follow fails or the bot idles
type: task
status: done
status_reason: null
priority: high
due_on: null
labels:
  - chat
  - controller
  - protocol
  - movement
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-28T22:08:14Z
updated_at: 2026-10-29T05:26:35Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

The bot needs somewhere to go when it has nothing to do or loses the plot.

### Commands (owners only)

- `set home` (also `home is here`, `this is home`): home is where the
  speaker stands, kept per world.
- `go home` (also `head home`, `return home`): walk there.

### Going home by itself

Each can be switched off in the controller's `home` config, and later from
the panel:

- when follow loses its leader (`target_lost`);
- when stuck recovery gives up (`stuck`), unless it was already going home;
- after an outage: a body task was running when the controller or the
  connection dropped, so once the bot is back in the world it goes home;
- after `idleMinutes` with no task, when it is away from home.

Homes and the interrupted flag live in a data file the controller owns
(`dataDir`), not in the config. `state` gains the world name.

## Acceptance criteria

- [x] `set home` stores the speaker's position for the current world, and it survives a controller restart
- [x] `go home` walks the bot there, and says so when no home is set
- [x] The bot goes home when follow fails with target_lost or stuck, each switchable
- [x] The bot goes home after an outage interrupted a body task, switchable
- [x] The bot goes home after idleMinutes with no task, when away from home
- [x] Verified in the game

## Implementation plan

- internal/home: Store (homes per world and the interrupted flag in
  dataDir/memory.json, replaced atomically) and Keeper (set, go, homeward
  tasks, ReturnOn rules, and a one-second Tick for outages and idling).
- Outage: Tick sets the flag while a follow or move_to runs and clears it
  when the task ends on the same connection; still set on the first in-world
  tick of a new connection means an outage, so the bot goes home.
- Chat: `set home` (speaker's position) and `go home` in the grammar; after
  a follow or move_to fails with target_lost or stuck, the brain sends it
  home on the same channel, unless the failed task was itself the way home.
- state gains `world`; config gains dataDir and home rules.

## Notes

**agent:claude-code/renki-groom** at 2026-10-28T22:15:09Z

Deployed as 83c9e81; the VM's state now carries world DizzysHellheim. Criteria 1-5 are covered by tests in internal/home and internal/chat; criterion 6 waits on checks 17-20 in docs/TESTING.md.

**agent:claude-code/renki-groom** at 2026-10-28T23:56:57Z

Lost follow verified in the game on 2026-10-28. At 18:38:00 the plugin traced 'Nimah out of sight', then player_left. At 18:38:05 task t-3 failed target_lost, the controller logged 'going home' (why: after follow failed: target_lost), and the bot said 'I lost track of you, Nimah. Heading home.' Still to verify before ticking 'Verified in the game': the outage trip (kill the controller mid-follow), the stuck give-up, and idling.

**agent:claude-code/renki-groom** at 2026-10-28T23:59:24Z

Outage trip verified in the game on 2026-10-28: after the controller was killed mid-follow at 18:58:16 and restarted at 18:58:51, it sent the bot home on reconnect at 18:59:04 (why 'after an outage') and said 'Something cut me off. Heading home.' Still to verify: the stuck give-up, and idling.

**agent:claude-code/renki-groom** at 2026-10-29T00:16:18Z

Verified in the game on 2026-10-28: the stuck give-up trip (19:13:36: why 'after move_to failed: stuck', with 'I'm stuck. I'll try to make it home.'), and afterDeath=home twice (19:03:59 and 19:14:53, each with 'I'm back. Heading home.'; the trip at 19:14:53 reached home at 19:15:26). Of the triggers, only idling is still unverified. The trip home after the give-up drowned on the way; see TKT-01M62G71Q (Bot swims into deep water on a straight-line path and drowns).

**agent:claude-code/renki-groom** at 2026-10-29T05:26:34Z

2026-10-28: seen in the game. The idle rule fired twice in the audit log (22:58:37 and 23:09:08, why=idle, 'Nothing to do. Heading home.'), after the outage rule at 00:05:34 on the 29th, and go home by chat at 22:48:07. The walks themselves stuck on the slope below the base, which is the walker, not the home rules (stuck recovery now climbs).

## Summary

Homes per world in dataDir; set home and go home by chat; trips home after follow fails, after an outage, after a respawn and when idle, each switchable, all seen in the game.
