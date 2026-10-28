---
schema: 4
id: TKT-01M628X112PXEDXM3ZGDBJWXPQ
title: Set a home and go back to it when follow fails or the bot idles
type: task
status: in-progress
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
claim:
  actor: agent:claude-code/renki-groom
  branch: t3/groom-start-renki
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 1c80f224a850be6f5ad8b36431596e5a0c94be0c
  session: null
  claimed_at: 2026-10-28T22:08:15Z
  expires_at: null
archive: null
created_at: 2026-10-28T22:08:14Z
updated_at: 2026-10-28T22:12:45Z
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
- [ ] Verified in the game

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
