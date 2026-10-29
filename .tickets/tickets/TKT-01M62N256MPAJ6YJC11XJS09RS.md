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
  actor: agent:claude-code/renki-groom
  branch: t3/m2
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 0726cab49205ebf014eefe292d668e7c793c8c95
  session: null
  claimed_at: 2026-10-29T15:09:07Z
  expires_at: null
archive: null
created_at: 2026-10-29T01:40:45Z
updated_at: 2026-10-29T15:34:30Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

`flee` (7.2): move away from an entity or a point to a distance. The controller uses it on low_health and when a creature targets the bot (perception's targetingMe), heading toward the owner or home rather than blindly away. Works without AllowCombat: running is always allowed.

## Acceptance criteria

- [ ] flee moves the bot away from a threat to a distance
- [ ] On low health with a threat near, the bot retreats toward its owner or home and says so

## Implementation plan

Moved ahead of the M2 order: on 2026-10-29 a Greyling at home killed the bot over and over (respawn, home after death, killed again).

### Plugin
- `flee` takes `from` (an entity or a position), `distance` (default 20) and an optional `toward` (an entity or a position).
- Core `Flee.Goal` picks where to run. That is `toward`, unless it lies past the threat (within 60° of the threat's bearing), in which case the bot runs straight away to `distance` + 5.
- The task runs there and re-aims every 0.5 s as the threat moves. It succeeds once the threat is `distance` away. A lost entity counts as escaped, and it fails `stuck` or `deep_water` like the walk does.
- Safety: flee needs AllowMovement, not AllowCombat.

### Controller
- New unit `internal/retreat`. Every second: if HP is below 40% and a hostile is targeting the bot within 15 m, it sends `flee` from the nearest such hostile.
- `toward` is the nearest owner in sight within 60 m, else home, unless home lies within 15 m of the threat. It says which: "Falling back to you, X", "Heading home", or "Getting away". It waits 10 s between retreats and does not supersede a flee already running.
- `TaskFlee` goes into KnownTasks, and home's body().

Docs: 7.2 and 9.

## Notes

**agent:claude-code/renki-groom** at 2026-10-29T15:34:30Z

Landed on t3/m2 and deployed on 2026-10-29. `flee` is now in the VM's allowedTasks, added through the panel settings.

- **Plugin:** FleeTask, which aims with Core Flee.Goal. It runs to the owner or home unless that lies past the threat, carries on to safety however far it is, stops at the owner's side, and runs on from home if the threat followed. Otherwise it runs straight away. It needs only AllowMovement.
- **Controller:** `internal/retreat`. Below 40% HP, with a hostile targeting the bot within 15 m, it flees toward an owner in sight (60 m) or home (150 m, and clear of the threat by 15 m), with 60 s per flee. It stays put at an owner's side, and says where it is going.

**Not seen in the game yet:** after the deploy the bot walked home unharmed, and the Greylings that killed it this morning had gone. Both criteria are in the code and tested (CoreTests and retreat_test), but stay unticked until a fight shows them.

**Why this was moved ahead of M2:** at 10:05–10:08 the bot died three times at home to a Greyling. I stopped it from the panel, and turned home.idleMinutes to 0 until the retreat was in, then back to 10.
