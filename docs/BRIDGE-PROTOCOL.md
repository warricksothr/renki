# Bridge Protocol v1 (draft)

Status: draft. This is the contract between the two halves of this repository:
the C# plugin inside Valheim and the Go controller outside it. It follows the
architecture of [Dahuchsi/ValheimAI](https://github.com/Dahuchsi/ValheimAI)
without using its code; section 11 says what was taken and what differs.

## 1. Three layers, and what each one owns

| Layer | Runs | Rate | Owns |
|---|---|---|---|
| **Plugin** (body) | BepInEx, inside the bot's Valheim client on the Windows VM | every physics frame (~50 Hz) | Anything that needs per-frame feedback: steering, stuck recovery, swinging at a target until it breaks, blocking, picking things up. |
| **Controller** (brain) | Separate Go process, on the VM or elsewhere | decisions at ~5 Hz | Every *choice*: which task to run, which target, when to flee, the task queue, memory, chat routing. |
| **LLM** (voice) | Called by the controller | only when addressed in chat | Words, and choosing at most one task from the menu the controller offers. Never in the tick loop. |

Rule of thumb: if getting it wrong for 200 ms kills you, it belongs in the plugin.
If it's a decision, it belongs in the controller. If it's a sentence, it can go
to the LLM.

The plugin is useful on its own: with no controller it stands still, and with
the controller but no LLM key, the command grammar in section 9 still works.

## 2. Transport

- WebSocket with JSON text frames, UTF-8. One message per frame.
- **The plugin dials the controller**, as ValheimAI does. The default is
  `ws://127.0.0.1:8765/ws/`.
- To run the brain off the VM, the controller listens on a LAN address and the
  plugin config sets `controllerUrl` and `token`. The token goes in `hello`, and
  a bad token closes the socket with code `4401`. (If the brain runs in WSL2, it
  needs mirrored networking or a port-proxy on the WSL host.)
- Close codes beyond the standard ones: `4400` when the first message is not
  a usable `hello` (including an unknown major version, which first gets an
  `error` with code `unsupported_version`), `4401` for a bad token, and `4409`
  when a newer plugin connection replaces this one. The controller holds one
  plugin connection at a time.
- The controller sends `ping` every 3 s, well inside the plugin's 10 s
  dead-man limit, and gives up on a plugin that sends nothing for 15 s.
- On disconnect the plugin reconnects with backoff (1 s, then 2, 4 … up to 30 s).
  While disconnected, the dead-man rule in section 8 applies.
- **Threading:** the socket runs on a background thread. Inbound messages go
  through a `ConcurrentQueue` that is drained in `Update()` on Unity's main
  thread. Game APIs are only ever touched from the main thread.

## 3. Envelope

Every message has these fields:

```json
{ "v": 1, "type": "state", "seq": 1042, "t": 18234.52 }
```

| Field | Meaning |
|---|---|
| `v` | Protocol major version. A receiver rejects a major version it doesn't know. |
| `type` | Message type (sections 5–7). |
| `seq` | Per-sender, per-connection counter. Lets the receiver drop stale states. |
| `t` | Server-synced game time in seconds (`ZNet.GetTimeSeconds()`), so both sides and the logs agree on "when". |
| `id` | Present on controller requests that expect an answer. |
| `re` | On a reply: the `id` it answers. |

Unknown fields are ignored. An unknown `type` gets an `error` reply. New fields
are added without bumping `v`. Removing or redefining a field bumps `v`.

## 4. Conventions

- **Positions:** Unity world space, metres, y up, written as `[x, y, z]` arrays
  rounded to 0.01. (ValheimAI uses `{x,y,z}`; arrays keep perception messages
  small.)
- **Angles:** degrees. Yaw 0 = +Z (north), increasing clockwise.
- **Entity IDs:** the object's ZDOID written as `"<userId>:<id>"`. It's stable
  for as long as the object exists and is never reused within a session. An ID
  that stops showing up in perception means the object is gone or out of range.
- **Item IDs:** prefab names (`Wood`, `AxeBronze`, `CookedMeat`). Display names
  are optional extras and never used as keys.
- **Durations:** seconds, as floats, suffixed `S` in field names (`timeoutS`).

## 5. Session lifecycle

```
plugin                                controller
  | -- hello ---------------------------> |
  | <------------------------- welcome -- |
  | -- state / perception / inventory --> |   (streams start)
  | <---------------------------- task -- |
  | -- task (status) -------------------> |
  | ...                                   |
  | <-------------------------- logout -- |   or plugin sends bye
```

The plugin reports one of these `phase` values: `menu`, `connecting`,
`in_world`, `dead`, `logging_out`.

### `hello` (plugin → controller)

```json
{
  "v": 1, "type": "hello", "seq": 1, "t": 0,
  "token": "…",
  "plugin": "0.2.0",
  "game": "0.221.12",
  "phase": "in_world",
  "character": "Bjorn",
  "world": "Midgard",
  "capabilities": ["stop", "move_to", "follow", "look_at", "say", "emote"]
}
```

`capabilities` lists the tasks this plugin build implements. The controller must
not send anything outside it.

### `welcome` (controller → plugin)

```json
{
  "v": 1, "type": "welcome", "seq": 1, "t": 0,
  "agentName": "Bjorn",
  "owners": ["Sothr"],
  "rates": { "stateHz": 5, "perceptionHz": 1 },
  "perception": { "radius": 40, "maxPerKind": 24 },
  "allowedTasks": ["stop", "move_to", "follow", "look_at", "say", "emote"]
}
```

`agentName` is the name the bot answers to in chat. It is empty when the
controller follows the character's own name, the `character` in `state`.

`allowedTasks` can only *narrow* what the plugin's own safety config allows
(section 8). `owners` are the players whose commands the bot obeys. Everyone
else gets conversation but no tasks.

### Session control

- `{"type":"logout","id":"…"}`: save and return to the main menu
  (`Game.instance.Logout()`).
- `{"type":"bye","reason":"quit"}`: either side, before closing.
- `join` (connect to a server from the main menu) is deferred. Launch arguments
  `+connect host:port +password …` cover it for now, and the plugin only has to
  automate character select (see the open questions).

## 6. Plugin → controller

### 6.1 `state` (at `stateHz`, default 5)

The bot's own body. Small and sent often.

```json
{
  "v": 1, "type": "state", "seq": 1042, "t": 18234.52,
  "phase": "in_world", "character": "Tasleroch", "world": "Midgard",
  "pos": [123.45, 31.02, -88.10], "yaw": 271.5, "vel": [0.0, 0.0, 3.9],
  "hp": 47.0, "hpMax": 60.0, "stamina": 38.0, "staminaMax": 75.0,
  "eitr": 0.0, "eitrMax": 0.0,
  "flags": ["grounded", "running"],
  "biome": "BlackForest", "env": "Rain",
  "time": { "day": 14, "dayFraction": 0.81, "night": false },
  "equipped": { "right": "AxeBronze", "left": "ShieldWood", "chest": "ArmorBronzeChest" },
  "effects": ["Rested", "Wet"],
  "weight": 212.0, "weightMax": 300.0,
  "task": { "id": "t-17", "task": "follow", "status": "running" }
}
```

`character` is the bot's character name as other players see it, and
`world` the name of the world it is in. Both are empty until the bot is in
the world.

`flags` is drawn from: `grounded`, `running`, `crouching`, `swimming`,
`sitting`, `sleeping`, `attacking`, `blocking`, `encumbered`, `in_shelter`,
`near_fire`.

### 6.2 `perception` (at `perceptionHz`, default 1)

What's near the bot, within `radius`, nearest first, capped at `maxPerKind`
per kind. v1 sends a full snapshot every time. Deltas come later if size turns
into a problem.

```json
{
  "v": 1, "type": "perception", "seq": 210, "t": 18234.60,
  "characters": [
    { "id": "1234:88", "kind": "player", "name": "Sothr", "pos": [120.1, 31.0, -86.0], "hpFrac": 1.0 },
    { "id": "0:5521", "kind": "creature", "prefab": "Greydwarf", "pos": [140.2, 33.9, -70.4],
      "hpFrac": 0.6, "stars": 1, "hostile": true, "targetingMe": true }
  ],
  "resources": [
    { "id": "0:7001", "kind": "tree", "prefab": "Beech1", "pos": [130.0, 31.5, -90.0], "minTool": 0 },
    { "id": "0:7044", "kind": "rock", "prefab": "MineRock_Copper", "pos": [150.3, 30.2, -95.8], "minTool": 2 },
    { "id": "0:7101", "kind": "pickable", "prefab": "RaspberryBush", "pos": [118.0, 31.1, -84.9] },
    { "id": "0:7120", "kind": "drop", "prefab": "Wood", "stack": 3, "pos": [129.6, 31.6, -89.2] }
  ],
  "structures": [
    { "id": "1234:301", "kind": "container", "prefab": "piece_chest_wood", "pos": [110.0, 32.0, -80.0] },
    { "id": "1234:302", "kind": "station", "prefab": "piece_workbench", "pos": [112.0, 32.0, -80.0] },
    { "id": "1234:310", "kind": "bed", "prefab": "bed", "pos": [108.0, 32.0, -78.0], "mine": true },
    { "id": "1234:320", "kind": "portal", "tag": "home", "pos": [105.0, 32.0, -75.0] },
    { "id": "1234:330", "kind": "door", "prefab": "wood_door", "pos": [111.0, 32.0, -82.0], "open": false }
  ]
}
```

| Kind | Values |
|---|---|
| `characters.kind` | `player`, `creature`, `tamed` |
| `resources.kind` | `tree`, `log`, `stump`, `rock`, `pickable`, `drop` |
| `structures.kind` | `container`, `station`, `bed`, `portal`, `door`, `fire`, `smelter`, `other` |

- `characters`: the bot itself is left out. `stars` is the creature's
  level less one. `hostile` means the creature would fight the bot, and
  `targetingMe` that it has picked the bot as its target.
- `structures`: only pieces of the kinds above. Walls, floors and the like
  are left out, so a base cannot fill the cap with them. `mine` is set on
  the bot's own bed, `tag` is a portal's name, and `open` is set on doors
  only.
- `resources`: `minTool` is the lowest tool tier that can harvest it.
  Pickables already picked are left out. A tree felled with an axe becomes
  a `log` that falls, and a `stump` left where it stood; a log splits into
  halves, each a new `log` with a new id, and those break into wood drops.
  A tree falling on a log breaks it too. Each step is a new object, so
  following one through means matching place and time, not id.
- The v1 plugin does not send `other`.

### 6.3 `inventory` (on change, and once after `welcome`)

```json
{
  "v": 1, "type": "inventory", "seq": 33, "t": 18230.00,
  "size": [8, 4],
  "items": [
    { "slot": [0, 0], "prefab": "AxeBronze", "stack": 1, "quality": 2,
      "durability": 140.0, "durabilityMax": 250.0, "equipped": true },
    { "slot": [3, 1], "prefab": "Wood", "stack": 42, "stackMax": 50 }
  ]
}
```

### 6.4 `chat` (every chat line except the bot's own)

```json
{
  "v": 1, "type": "chat", "seq": 501, "t": 18235.10,
  "from": { "name": "Sothr", "playerId": "1234", "owner": true },
  "channel": "normal",
  "text": "bjorn, follow me",
  "pos": [120.1, 31.0, -86.0], "dist": 4.2
}
```

`channel` is one of `normal`, `shout`, `whisper`, `ping`.

### 6.5 `event` (as it happens)

```json
{ "v": 1, "type": "event", "seq": 777, "t": 18236.00,
  "event": "damaged", "by": "0:5521", "amount": 6.5, "damageType": "blunt" }
```

| `event` | Extra fields |
|---|---|
| `damaged` | `by`, `amount`, `damageType` |
| `died` | `by` |
| `respawned` | `pos` |
| `item_added` | `prefab`, `count` |
| `skill_raised` | `skill`, `level` |
| `player_joined`, `player_left` | `name`, `playerId` |
| `entered_biome` | `biome` |
| `low_health` | `hpFrac` (fires once on crossing 0.3) |

### 6.6 `task` status (reply to a `task`, and on every change)

```json
{ "v": 1, "type": "task", "seq": 812, "t": 18236.20,
  "re": "t-17", "status": "failed", "reason": "stuck", "detail": "no progress for 9s" }
```

`status` moves through `accepted` → `running` → one of `succeeded`, `failed`,
`cancelled`. `rejected` is the only reply when a task never starts.

| `reason` | When |
|---|---|
| `unsupported` | Not in `capabilities`. |
| `not_allowed` | Blocked by the safety config (section 8). |
| `bad_args` | Missing or malformed arguments. |
| `invalid_target` | ID unknown, or the wrong kind of thing. |
| `target_lost` | Target left range or was destroyed by someone else. |
| `unreachable` | No path. |
| `stuck` | Stuck recovery gave up. |
| `timeout` | `timeoutS` elapsed. |
| `no_tool` | Nothing in the inventory meets `minTool`. |
| `inventory_full`, `missing_materials` | — |
| `superseded` | A newer body task replaced it. |
| `stopped` | `stop`, the dead-man rule, or a logout. |
| `died` | — |

### 6.7 `trace` (when the plugin decides something itself)

The plugin's own decisions inside a task, which no other message shows: a
change of gait while following, a new path, a stuck recovery. The controller
writes them to its audit log (docs/AUDIT.md) and acts on none of them.

```json
{
  "v": 1, "type": "trace", "seq": 1043, "t": 18234.60,
  "task": "t-17", "source": "walker", "msg": "stuck, trying Strafe",
  "data": { "attempt": 2, "pos": [123.45, 31.02, -88.1], "center": [123.1, 31.0, -88.3] }
}
```

`task` is the running task, if any. `source` names what decided (`follow`,
`walker`). `data` holds the numbers behind the decision, and its keys depend
on the message.

### 6.8 Housekeeping

`pong` (reply to `ping`), and `error`, which looks like
`{"re":"…","code":"bad_message","detail":"…"}`.

## 7. Controller → plugin

### 7.1 Task model

```json
{ "v": 1, "type": "task", "id": "t-17", "seq": 90, "t": 18234.0,
  "task": "follow", "args": { "player": "Sothr", "distance": 3.5 }, "timeoutS": 0 }
```

- **Body tasks** move or occupy the character. Only one runs at a time. A new
  body task cancels the running one, which then reports `cancelled` with reason
  `superseded`.
- **Instant tasks** (`say`, `emote`, `look_at`, `equip`, `unequip`,
  `use_item`) run alongside a body task without replacing it.
- `timeoutS: 0` means no timeout. `stop` cancels everything.
- The queue lives in the controller. The plugin only ever holds one body task.

### 7.2 Task catalogue

| Task | Kind | Args | Milestone |
|---|---|---|---|
| `stop` | body | — | M1 |
| `move_to` | body | `to`: `[x,y,z]` or entity id; `stopDistance` (default 1.5); `run` (bool) | M1 |
| `follow` | body | `player` (name or id); `distance` (default 3.5); `run`: `"auto"` matches the leader | M1 |
| `look_at` | instant | `to`: position or entity | M1 |
| `say` | instant | `text`; `channel`: `normal`, `shout` or `whisper`; `to` (whisper target) | M1 |
| `emote` | instant | `name` (`wave`, `sit`, `point`, …) | M1 |
| `pickup` | body | `entity` (a drop) | M2 |
| `interact` | body | `entity`; `alt` (bool); `hold` (bool) | M2 |
| `harvest` | body | `entity` (tree, log, rock or pickable); `collect` (default true) | M2 |
| `equip` / `unequip` | instant | `item` (prefab) / `slot` | M2 |
| `use_item` | instant | `item` (eat, drink) | M2 |
| `deposit` | body | `container`; `items`: `[{prefab, count \| "all"}]` | M2 |
| `withdraw` | body | `container`; `items` | M2 |
| `craft` | body | `recipe` (prefab); `count`; walks to the needed station | M2 |
| `sleep` | body | `bed` (optional; defaults to the bot's own) | M2 |
| `repair_all` | body | `station` | M2 |
| `attack` | body | `entity`; `style`: `melee`, `ranged` or `auto` | M3 |
| `guard` | body | `around`: player or position; `radius` | M3 |
| `flee` | body | `from`: entity or position; `distance` | M3 |
| `controls` | body | raw `move` `[x,z]`, `run`, `jump`, `attack`, `block`, `dodge`, … ; `durationS` ≤ 1 | debug |

Notes:

- `harvest` covers approaching the target, equipping the best tool that meets
  `minTool`, swinging until the target breaks, then sweeping up drops within a
  few metres. Choosing *which* tree is the controller's job.
- `attack` includes blocking and dodging. Choosing *whether* to fight is the
  controller's job.
- `controls` exists for debugging and later RL experiments. It's gated by
  `allowRawControls` and expires on its own, so a lost connection can't leave
  a key held down.

### 7.3 Movement inside the plugin

- Get a path from the game's own navmesh
  (`Pathfinding.GetPath(..., AgentType.Humanoid)`). Fall back to a straight line
  when no path comes back.
- Steering sets the move direction and look yaw every frame.
- **Stuck detection:** the bot is stuck while it stays within 3 m of where it
  was. Every 3 s inside that circle triggers the next recovery: a jump, then a
  strafe (at a walk), then a re-path. After 3 rounds the task fails with
  `stuck`. Hopping or shuffling against a wall does not reset it. Once a
  recovery has fired, leaving the circle only starts a new circle; the count
  resets when the bot gets 6 m from where the first recovery fired, so a
  strafe that carries it just over the edge and back is still stuck.
- `follow` keeps `distance` with hysteresis. It walks when the gap is under
  2× the distance, runs beyond that, and stops inside the distance.

### 7.4 Other messages

- `ping`
- `config`: changes `rates` and `perception` mid-session.
- `logout`, `bye` (section 5).

## 8. Safety (enforced in the plugin, not just the controller)

1. **Whitelist only.** The plugin carries out the tasks in section 7.2 and
   nothing else. No console or dev commands, ever.
2. **The plugin's own config wins.** `allowGather`, `allowCombat`,
   `allowSailing`, `allowBossSummoning` and `allowRawControls` are read from the
   plugin's BepInEx config. `welcome.allowedTasks` can narrow them but never
   widen them.
3. **Dead-man rule.** If no controller message arrives for 10 s, the plugin
   cancels everything, stands still and stays quiet. After a reconnect nothing
   resumes on its own.
4. **Chat limits.** `say` is rate-limited (1 per 2 s, burst of 3), truncated to
   the game's line length, and any leading `/` is stripped.
5. **No friendly fire.** `attack` is refused on players and tamed creatures,
   and on anything `BaseAI.IsEnemy` says isn't hostile.
6. **Wards.** `harvest` and `interact` are refused where the game's
   private-area access check fails for the bot.
7. **Bosses.** Interacting with offering bowls is refused unless
   `allowBossSummoning` is set.
8. **No destroying items.** There's no drop or trash task. Items leave the bot
   only through `deposit`.

## 9. Chat and the LLM

The controller doesn't talk to model providers itself. LLM calls, provider
configuration and credentials go through
[terva](https://terva.sh): its agent core supports two dozen providers, and its
chat and play modes already cover personas, speakers and per-chat sessions.
Two ways in are open, and choosing between them is a spike:

- **Embed terva** in the controller through its Go SDK
  (`packages/agent/sdk`), with the task menu offered as tools.
- **Make the controller a terva connector extension.** Valheim chat becomes a
  connector, the same as terva's Discord bridge, and the same process also
  provides the game tasks as tools.

Either way, the controller still validates every task terva proposes before it
goes to the plugin.

```
chat event
  ├─ not addressed to the bot? ────────────────► log to chat history, done
  ├─ matches the command grammar? ─────────────► tasks directly (no LLM)
  └─ otherwise ─► LLM(persona, state summary, recent chat, task menu)
                    └─► say text + at most one proposed task
                          └─► controller validates (owner? allowed? args?) ─► task
```

- **Addressed** means a whisper to the bot, a message containing its name
  (`agentName`, or else the character's name) or one of the controller's
  `nicknames`, or a reply within 20 s of the bot speaking to that player.
- **Command grammar** (the fast path; works with no API key): `<name> follow`,
  `stop`, `status`, `come here`, `wait here`, `set home`, `go home`. Only
  `owners` can issue tasks.
- **Home** is where the speaker stood when they said `set home`, one per
  world, kept in the controller's `dataDir` across restarts. The bot also
  goes home by itself, as the `home` config allows: when follow loses its
  leader (`whenLost`), when stuck recovery gives up on anything but the way
  home (`whenStuck`), when an outage cut a moving task short, including a
  logout or a controller restart (`afterOutage`), and after `idleMinutes`
  with no task when it is more than 5 m from home.
- **After a respawn** the bot does what `home.afterDeath` says, once it is
  back in the world: `home` (the default) walks home; `resume` picks a
  follow back up for the same player, and goes home if there was none or
  the player has gone (the follow then fails with `target_lost`); `stay`
  stays at the spawn point. Any order given after the death wins, except
  `set home` and `status`. The controller reads the death and the respawn
  from `state.phase` rather than the `died` and `respawned` events, so a
  lost event does not lose them. The plugin reports `dead` from the death
  until the respawn wait ends, about 10 s.
- **LLM input** is a short summary, not the raw streams. It covers health,
  biome, time of day, the current task, notable inventory, threats in view, the
  last ~20 chat lines, and the allowed tasks offered as tools.
- **LLM output** is one `say`, plus at most one task. The controller checks the
  task against the owner list, `allowedTasks` and argument validity before
  sending it.
- **Budgets:** each player gets a cooldown, the controller has a daily token
  cap, and an LLM failure makes the bot fall back to a canned "Hm?" rather than
  silence.

## 10. Valheim API touchpoints

Every signature below was read from the metadata of `assembly_valheim.dll` in
the local Steam install on 2026-10-28. Re-check them after game updates;
`SetControls` in particular has changed shape before.

| Need | Hook (verified signature) |
|---|---|
| The bot's player | `Player.m_localPlayer` (static field) |
| Drive movement and attacks | Harmony prefix on `PlayerController.FixedUpdate()` that substitutes our inputs into `Player.SetControls(Vector3 movedir, bool attack, bool attackHold, bool secondaryAttack, bool secondaryAttackHold, bool block, bool blockHold, bool jump, bool crouch, bool run, bool autoRun, bool dodge)` |
| Look direction | `Character.SetLookDir(Vector3 dir, float transitionTime)`, or `Player.SetMouseLook(Vector2)` for mouse-style deltas. State lives in `Character.m_lookYaw` (Quaternion), `Character.m_lookDir`, `Player.m_lookPitch`. |
| Incoming chat | Postfix on `Chat.OnNewChatMessage(GameObject go, long senderID, Vector3 pos, Talker.Type type, UserInfo sender, string text)` |
| Outgoing chat | `Chat.instance.SendText(Talker.Type type, string text)` |
| Emotes | `Player.StartEmote(string emote, bool oneshot)` |
| Vitals | `Character.GetHealth/GetMaxHealth()`, `Player.GetStamina/GetMaxStamina/GetEitr/GetMaxEitr()` |
| Status effects | `Character.GetSEMan().GetStatusEffects()` → `List<StatusEffect>` |
| Inventory | `Humanoid.GetInventory()`, `Inventory.GetAllItems()` → `List<ItemDrop.ItemData>`, `Inventory.m_onChanged` (an `Action`) |
| Equip and use | `Humanoid.EquipItem(ItemData, bool triggerEquipEffects)` → bool; `UnequipItem(ItemData, bool)`; `UseItem(Inventory, ItemData, bool fromInventoryGui)` |
| Interact | `Player.Interact(GameObject go, bool hold, bool alt)` |
| Characters nearby | `Character.GetAllCharacters()` (static), `BaseAI.IsEnemy(Character a, Character b)` (static), `BaseAI.GetTargetCreature()` |
| Resources and structures nearby | `Physics.OverlapSphere`, then `GetComponentInParent<TreeBase / TreeLog / MineRock / MineRock5 / Destructible / Pickable / ItemDrop / Container / CraftingStation / Bed / TeleportWorld / Door / Fireplace / Smelter>` |
| Entity ID | `ZNetView.GetZDO().m_uid` (a `ZDOID`) |
| Tool tier | `int m_minToolTier` on `TreeBase`, `TreeLog`, `MineRock`, `MineRock5`, `Destructible` |
| Pathfinding | `Pathfinding.instance.GetPath(Vector3 from, Vector3 to, List<Vector3> path, Pathfinding.AgentType agentType, bool requireFullPath, bool cleanup, bool havePath)` |
| Time and weather | `EnvMan.instance.GetDayFraction()`, `EnvMan.IsNight()` (static), `GetDay()`, `GetCurrentEnvironment()` → `EnvSetup` |
| Biome | `WorldGenerator.instance.GetBiome(Vector3)` or `Heightmap.FindBiome(Vector3)` (static) |
| Synced time | `ZNet.instance.GetTimeSeconds()` → double |
| Ward check | `PrivateArea.CheckAccess(Vector3 point, float radius, bool flash, bool wardCheck)` (static) |
| Game version for `hello` | `Version.GetVersionString(bool includeMercurialHash)` (static) |
| Menu and character select | `FejdStartup.ParseArguments()` handles `+connect` (the literal is in the assembly); `FejdStartup.m_profiles` + `OnCharacterStart()` look like the path to picking a character automatically |
| Logout | `Game.instance.Logout(bool save, bool changeToStartScene)` |

## 11. Relationship to ValheimAI

ValheimAI has no LICENSE file, so its code is all rights reserved. Nothing in
this repository is copied from it. What we took is the design, which copyright
doesn't cover:

- The architecture (plugin → WebSocket → controller → planner/LLM).
- The port and the `/ws/` path.
- The plugin dialling the controller.
- Per-capability safety switches, and the rule that the plugin only carries out
  actions it explicitly implements.

Where we differ:

- The controller is Go, not C#/.NET 8.
- `action` becomes `task`, with IDs, a status lifecycle and failure reasons.
- Positions are arrays rather than `{x,y,z}` objects.
- New: the handshake with capabilities and token, plus the `perception`,
  `inventory`, `chat` and `event` streams.

## 12. Open questions / spikes

1. **WebSocket client inside Valheim's Mono runtime.** Does
   `System.Net.WebSockets.ClientWebSocket` work, or do we need websocket-sharp?
   Spike this first, because everything depends on it.
2. **Running unattended on the VM.** Does the game keep simulating when the
   window is unfocused or minimized, or when the RDP session drops
   (`Application.runInBackground`)? If not, run the VM console session
   persistently.
3. **Character select automation.** `+connect` is parsed by `FejdStartup`.
   What's left to confirm in game is that selecting from `m_profiles` and
   calling `OnCharacterStart()` joins without a click.
4. **Where the brain runs.** For M1, keep the controller on the VM (simpler)
   or on WSL (easier to iterate on)? The protocol supports both.
5. **Perception size.** Measure snapshot size in a busy base before deciding
   whether deltas are needed.

## Appendix: one M1 exchange

```text
P→C hello      {character:"Bjorn", phase:"in_world", capabilities:[stop,move_to,follow,look_at,say,emote]}
C→P welcome    {owners:["Sothr"], allowedTasks:[…same…]}
P→C state …    (5 Hz)
P→C chat       {from:{name:"Sothr",owner:true}, text:"bjorn, follow me", dist:4.2}
C   grammar match "follow" (no LLM)
C→P task       {id:"t-1", task:"follow", args:{player:"Sothr", distance:3.5}}
C→P task       {id:"t-2", task:"say", args:{text:"Right behind you."}}
P→C task       {re:"t-1", status:"accepted"} … {re:"t-1", status:"running"}
P→C task       {re:"t-2", status:"succeeded"}
P→C chat       {from:{name:"Sothr"}, text:"bjorn how are you holding up?"}
C   addressed, no grammar match → LLM(summary: hp 47/60, Black Forest, raining, following Sothr)
C→P task       {id:"t-3", task:"say", args:{text:"Wet, bruised, and still swinging. Lead on."}}
```
