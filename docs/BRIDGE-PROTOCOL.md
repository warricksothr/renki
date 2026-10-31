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
`sitting`, `sleeping`, `in_bed`, `attacking`, `blocking`, `encumbered`, `stranded`,
`in_shelter`, `near_fire`.

`in_bed` means attached to a bed, including waiting for the other players;
`sleeping` is the game's actual sleep state.

Optional `food` reports actual active meals and ordinary carried foods while
alive in the world. For example:

```json
{"active":[{"prefab":"CookedDeerMeat","remainingS":850.5}],
 "carried":[{"prefab":"NeckTailGrilled","health":25,"regen":2,"canEat":true}]}
```

`remainingS` is the game's remaining meal time. Carried foods are unique by
prefab and exclude unusable world-level stacks. `canEat` comes from current
game eligibility, including duplicate/full slots and consumption effects.
Missing `food` means unknown; empty arrays mean observed empty. The panel
status API exposes the same observation.

Optional `pvp` is the local player's current PvP setting. Player entries in
`perception.characters` also include `pvp` when their network state is valid.
Explicit `false` means disabled; omission means unknown.

`encumbered` means `weight` is over `weightMax`. The game then lets the bot
walk but not run, stops its stamina coming back, and drains stamina while it
moves. With no stamina left it cannot move at all until it carries less:
that is `stranded`, and body tasks then fail with `encumbered` (6.6) rather
than going through stuck recovery. The game's auto-pickup never takes what would cross `weightMax`,
and neither does `pickup` (`too_heavy`).

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
| `structures.kind` | `container`, `station`, `bed`, `portal`, `door`, `fire`, `smelter`, `tombstone`, `other` |

- `characters`: the bot itself is left out. `stars` is the creature's
  level less one. `hostile` means the creature would fight the bot, and
  `targetingMe` that it has picked the bot as its target.
- `structures`: only pieces of the kinds above, and graves (`tombstone`)
  with something in them. Walls, floors and the like are left out, so a
  base cannot fill the cap with them. Walls come in `layout` (6.9). `mine`
  is set on the bot's own bed and its own graves, `tag` is a portal's name, and `open` is set on doors
  only. Beds also report nullable `owned`: false means explicitly unclaimed,
  true means claimed, and absence means unknown. `mine` distinguishes the
  bot's own bed from another player's. Unknown ownership is never free.
- `resources`: `minTool` is the lowest tool tier that can harvest it.
  Optional `yield` describes a drop/simple pickable's primary output:
  `{prefab, count, stackMax, weight}`. Count is a capacity upper bound
  including possible skill bonuses; weight is per item. Unknown, extra-drop
  or theft-provoking pickables omit it. PickableItem objects are included
  as pickables. This metadata does not prove pickup or navigability.
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
      "durability": 140.0, "durabilityMax": 250.0, "equipped": true,
      "weight": 2.0, "type": "OneHandedWeapon" },
    { "slot": [3, 1], "prefab": "Wood", "stack": 42, "stackMax": 50,
      "weight": 2.0, "type": "Material" },
    { "slot": [4, 1], "prefab": "Amber", "stack": 3, "stackMax": 20,
      "weight": 0.1, "type": "Material", "value": 5 }
  ]
}
```

`prefab` is the item's prefab, as drops and `deposit` name it. The few items
the game has no prefab for go by their name token instead (`$item_…`).
`quality` is set on items that have levels, `durability` on items that wear,
and `equipped` on what the bot holds or wears. `weight` is the weight of one
of the stack, and `type` the game's item type (`Material`, `Consumable`,
`OneHandedWeapon`, `Tool`, `Helmet`, …). `value` is what a trader gives for
one, set only on valuables such as coins, gems and amber. The controller
uses these to choose what to drop when the bot carries too much. A burst of changes, such as
picking up a pile, is one message.

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

`channel` is one of `normal`, `shout`, `whisper`, `ping`. `pos` is where
the speaker stands: the game sends their head, about 1.6 m higher, so the
plugin takes their character's position when it is loaded near the bot and
otherwise the ground under the head. For a `ping` it is the pinged point.

### 6.5 `event` (as it happens)

```json
{ "v": 1, "type": "event", "seq": 777, "t": 18236.00,
  "event": "damaged", "by": "0:5521", "amount": 6.5, "damageType": "blunt" }
```

| `event` | Extra fields |
|---|---|
| `damaged` | `by`, `amount`, optional `damageType`, `pos`, attacker `prefab` |
| `died` | `by`, optional `pos`, attacker `prefab` |
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

A final receipt may include `entity`, the exact native-selected target ID.
For an acquisition/kiting attack this binds the enemy selected after dispatch.
Successful combat requires `detail: "observed Greyling death"` and a bound
target; disappearance or unloading does not prove death. An explicitly named
target cannot be substituted by another receipt entity.

| `reason` | When |
|---|---|
| `unsupported` | Not in `capabilities`. |
| `not_allowed` | Blocked by the safety config (section 8). |
| `bad_args` | Missing or malformed arguments. |
| `invalid_target` | ID unknown, or the wrong kind of thing. |
| `target_lost` | Target left range or was destroyed by someone else. |
| `unreachable` | No path. |
| `deep_water` | No navmesh path, and the straight line crosses water too deep to wade (over 1.4 m), where the bot would drown. |
| `stuck` | Stuck recovery gave up. |
| `timeout` | `timeoutS` elapsed. |
| `no_tool` | Nothing in the inventory meets `minTool`. |
| `inventory_full`, `missing_materials` | — |
| `too_heavy` | Taking it would carry the bot past `weightMax`. |
| `encumbered` | Over `weightMax` with no stamina left: the game will not let the bot move until it carries less. |
| `refused` | The game would not do it: `interact` on a bed someone else owns, or with no roof over it. `detail` is what the game said, in its own words. |
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

### 6.9 `layout` (when the walls in range change, and once after `welcome`)

```json
{
  "v": 1, "type": "layout", "seq": 34, "t": 18230.50,
  "walls": [
    { "a": [-40.0, 36.1, 1.0], "b": [-38.0, 36.1, 1.8], "h": 2.0 },
    { "a": [-38.0, 36.1, 1.8], "b": [-38.0, 36.1, 3.8], "h": 1.0 }
  ]
}
```

The walls within the perception radius, nearest first, at most 512. Each is
the line along its foot, from `a` to `b`, and `h` is its height in metres.
A wall is any building piece that is not a structure (6.2), whose solid
colliders fill a box at least 1.2 m tall and at least twice as long as it is
thick, and no more than about a metre thick: wooden and stone walls,
palisades and fences. Floors, roofs, stairs, posts and half walls are not
walls. Walls do not move, so a new `layout` goes out only when the set
within range changes, usually as the bot walks, and each one replaces the
last. It comes after the `perception` of the same scan.

### 6.10 `container` (on access, and after a transfer)

```json
{ "v": 1, "type": "container", "seq": 35, "t": 18231,
  "entity": "1234:301", "pos": [110, 32, -80], "size": [5, 2],
  "items": [{ "slot": [0, 0], "prefab": "Wood", "stack": 12 }],
  "task": "t-18", "direction": "deposit", "moved": { "Wood": 12 },
  "remaining": { "Wood": 3 } }
```

Contents use the same item fields as `inventory`. A snapshot follows normal
chest access/ownership, before moving items and again after the transfer.
`task`, `direction`, `moved` and `remaining` accompany the latter; empty
counts are omitted. Counts reflect actual game transfers, including partial
success before a failure. The bot's pack is also reported immediately.
Session snapshots reset on a new connection or world; they are observations,
not a claim that an unseen chest still has those contents.

### 6.11 `terrain` (loaded-heightmap measurements, at most once a second)

`terrain` carries `world` and `tiles`. Each tile has signed integer `x` and
`z` indices in a 16 m grid (use floor division for negative positions),
`waterLevel`, and 25-element `heights` and `biomes` arrays. Samples are
corners 4 m apart, row-major with +x inside each row and rows towards +z:
index `iz * 5 + ix` is at `(x * 16 + ix * 4, z * 16 + iz * 4)`.
Unknown heights/biomes are JSON null. See `testdata/plugin/terrain.json`.

Tiles replace the same indices in the current connection/world cache.
One plugin update samples at most one tile (25 loaded-heightmap lookups).
The bounded cache prioritizes nearby/ahead tiles, refreshes known samples
after 30 s and missing samples after 5 s, and invalidates affected tiles
after heightmap regeneration. Unloading a heightmap preserves previous
measurements. Identical freshly sampled tiles are sent as confirmations; a
cached resend retains its original measurement times. The plugin/controller each
retain a 256-tile session cache; accepted measured data also feeds a separate
durable atlas, not exploration coverage or evidence of a safe route.

New producers include `scope` (`namespace`, decimal-string `worldUid`, `epoch`,
`source`, `stream`, `game`, integer `generator`, plugin/version fingerprint `mods`),
`capturedAt` in producer monotonic seconds and each tile's 25-element `measuredAt`
array in the same clock. Unknown corners have null times; carried loaded values
keep their original times. Times must be finite, nonnegative and no later than
`capturedAt`. The receiver estimates observation time from receipt minus elapsed
producer age. Repeated producer ticks never renew stored freshness. Optional
`state.worldUid` must match the scope; a same-name UID change clears live map
state. Legacy terrain without a manifest is accepted for the session only.
See `testdata/plugin/terrain-atlas.json` and [ATLAS.md](ATLAS.md).

`state.location` is optional for older plugins. It includes optional
`groundHeight` (terrain below the character), `support` (`terrain`,
`structure`, `ship`, `other`, `swimming`, `airborne`, `liquid`, `unknown`),
`inWater`, and `waterDepth` (water above the character's feet, metres).
It distinguishes a raised supporting floor from the ground underneath.
Character elevation remains `state.pos[1]`; the existing `swimming` and
`in_shelter` flags retain their meaning. See `state_location.json`.

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
| `move_to` | body | `to`: `[x,y,z]` or entity id; `stopDistance` (default 1.5); `run` (bool); `dry` (bool, default false) | M1 |
| `follow` | body | `player` (name or id); `distance` (default 3.5); `run`: `"auto"` matches the leader | M1 |
| `look_at` | instant | `to`: position or entity | M1 |
| `say` | instant | `text`; `channel`: `normal`, `shout` or `whisper`; `to` (whisper target) | M1 |
| `emote` | instant | `name` (`wave`, `sit`, `point`, …) | M1 |
| `pickup` | body | `entity` (a drop) | M2 |
| `drop` | instant | `prefab`; `count` (at least 1) | M2 |
| `interact` | body | `entity`; `alt` (bool); `hold` (bool); `sleep` (bool, beds only) | M2 |
| `harvest` | body | `entity` (tree, log, stump or mineable rock); `collect` (default true) | M2 |
| `equip` / `unequip` | instant | `item` (prefab) / `slot`: `right`, `left`, `hands`, `head`, `chest`, `legs`, `shoulder`, `utility`, `trinket` or `ammo` | M2 |
| `use_item` | instant | `item` (carried food prefab; eat one) | M2 |
| `deposit` | body | `container`; `items`: `[{prefab, count \| "all"}]` | M2 |
| `withdraw` | body | `container`; `items` | M2 |
| `craft` | body | `recipe` (prefab); `count`; walks to the needed station | M2 |
| `sleep` (planned standalone task) | body | Chat currently uses `interact` with `sleep:true` | M2 |
| `repair_all` | body | `station` | M2 |
| `attack` | body | `entity`; `style`: `melee`, `ranged` or `auto` | M3 |
| `guard` | body | `around`: player or position; `radius` | M3 |
| `flee` | body | `from`: entity or position; `distance` (default 20); `toward`: entity or position (optional) | M3 |
| `controls` | body | raw `move` `[x,z]`, `run`, `jump`, `attack`, `block`, `dodge`, … ; `durationS` ≤ 1 | debug |

Notes:

- `interact` with `sleep:true` uses only a bed and cannot combine `alt` or
  `hold`. It claims an unowned bed or sets the bot's spawn point if necessary,
  waits at most three seconds for claim ownership, then uses it again to lie
  down. A changed spawn point alone is not sleep success. Another player's
  bed is refused; the game's night, roof, shelter, fire, wetness and enemy
  checks remain authoritative. Cancellation discards the whole sequence.

- `deposit` and `withdraw` use a storage chest entity ID and 1–256 unique
  prefab requests with a positive integer count or `"all"`. Both require
  AllowGather and AllowMovement. The bot approaches within sight and reach,
  requests normal access, waits for ownership and current contents, then
  tops up compatible stacks before using empty slots. It preserves item
  metadata, skips equipped items and respects carry weight on withdrawal.
  Tombstones are not storage. A ward/private chest or a chest in use is
  refused; access times out after five seconds. All endings release any
  acquired use lock, including cancellation and death. Late access replies
  after cancellation are consumed without opening a UI or moving items.
  Partial transfers report exact `moved` / `remaining` counts in `container`,
  followed by failure explaining missing/equipped items, weight or space.
- `move_to` and `pickup` arrive within `stopDistance` across the ground and
  within 1.75 m up or down, so a bot under a raised floor has not reached a
  point on it. When the navmesh reaches only part of the way, the walker's
  path trace says how far short it stops; if that is within 4 m of the goal
  across the ground but off its level (under the floor, at the foot of a
  rock), the walker tries bounded alternative approaches before failing
  `unreachable` (7.3).
  Intermediate and final path waypoints also require arrival on their level;
  standing directly beneath a waypoint does not advance the path follower.
- `pickup` walks to the drop and lets the game's auto-pickup take it, as it
  does for any player within about 2 m, picking it up directly if it is still
  there once the bot stands on it. A drop the game merges into a nearby stack
  of the same item is followed there. It succeeds when the bot carries more
  of the item than it did, which can also come from another stack of it that
  auto-pickup took on the way, and fails with `inventory_full` when none fits,
  `too_heavy` when the stack would carry the bot past `weightMax`,
  `target_lost` when the drop is gone and the bot did not get it,
  `invalid_target` when the game will not hand it over after a few tries,
  or `stuck` or `encumbered` as `move_to` does. A pickup that got some of the
  stack succeeds, and its detail says how many stayed behind and why, ending
  in one of `no room for the other N`, `the other N is too heavy to carry`,
  `the other N is out of reach` or
  `the game kept the other N`. The controller reads these phrases to tell the
  owner what happened, so they are part of the protocol. It needs `allowGather` and `allowMovement`, since it
  walks.
- `drop` puts up to `count` of a prefab down at the bot's feet, as a player
  drops from the inventory, never from a stack it has equipped. It succeeds
  with `dropped N PREFAB`, and fails with `invalid_target` when the bot has
  none it may drop. The game marks a player's own drop so its auto-pickup
  does not take it straight back. It needs `allowDrop`, which is on by
  default and needs no movement.
- `interact` walks to the entity until a ray from the bot's eye meets it
  within 2.5 m, as the game's own hover finds things, faces it, and uses it
  through its `Interactable` as a player pressing the use key would; `alt` is
  the modifier key, and `hold` keeps the key down for 1 s, repeating as the
  game does. It succeeds when the game says it did something, with what the
  game said on screen in `detail`, and fails with `refused` when it did not.
  - What it may use goes by what the thing is, and only what is named here:
    beds and doors; offering bowls with `allowBossSummoning`; pickables,
    beehives and sap collectors with `allowGather`. A ship's helm is
    `not_allowed` even with `allowSailing`, until the plugin sails: taking it
    holds the bot there.
  - A bed answers the game's call with false whatever it did, so for a bed
    success is what changed: it became the bot's spawn point, or the bot lay
    down in it (its own bed already, at night). A ship is named by its hull,
    and what would be used is its helm. Most offering bowls take an item
    rather than a use, so `interact` on one mostly comes back `refused`.
  - A tombstone is the bot's own grave when all of it fits in the pack, and
    using it takes everything; otherwise it counts as a chest. If the pack
    has filled by the time the bot gets there, it fails `inventory_full`.
  - Drops (`pickup`), chests (`deposit`, `withdraw`), crafting stations,
    portals and signs are `invalid_target`, since they have their own tasks
    or open a window the bot cannot use. So is a thing with several parts to
    use and none named, such as a smelter's ore, fuel and output.
  - Wards are `not_allowed`: switching one, or joining it, is its owner's to
    do. So is anything else (chairs, carts, fires, dungeon entrances), since
    it sits the bot down, harnesses it, carries it off or takes from its pack.
  - It needs `allowMovement`, and nothing behind a ward the bot has no access
    to.
- `use_item` eats one carried food by prefab through the normal inventory
  use action. It preserves world-level, duplicate-food and full-food restrictions,
  and confirms both the one-item decrease and the matching active food. Missing
  items and non-food consumables are `invalid_target`; game refusal is `refused`.
  Health regenerates normally from active food; the task does not heal directly.
  It needs the plugin enabled and welcome permission, with no gather/movement
  requirement. The controller fills unused food slots before refreshing eligible
  meals, preferring health and then regeneration. It waits for a new observation
  after eating and backs off failures. It defers while attacking, sleeping or
  fleeing and never replaces a body task. When below 80% health with no active
  meal or usable carried food, it asks configured owners through normal `say`,
  at most once per five minutes. Unknown/stale observations, disconnects, menus,
  death and missing permission prevent automatic actions.
- `equip` holds or wears an item by prefab: of its stacks, the one in best
  repair, then of highest quality. One already equipped succeeds as it is.
  A broken item, or one that is not equipment, is `invalid_target`; the game
  saying no (swimming, mid-swing) is `refused` with what it said. `unequip`
  empties a slot, and `hands` both hands; an empty slot is
  `invalid_target`. Both need `allowMovement`: they move nothing, but the
  gear is the body's. The voice is offered `equip` with the gear the bot
  carries to choose from, and `unequip` with the slots.
- `flee` runs from `from` until it is `distance` away across the ground, and
  succeeds then, or when an entity it ran from is gone or dead. With
  `toward` (the owner, or home) it runs there instead, however far, unless
  that lies past the threat (within 60° of its bearing). At the owner or
  a position (home) it stops if the threat is `distance` away, and with the
  threat still close runs on straight away. Otherwise it runs
  straight away, to 5 m past `distance`. It re-aims every 0.5 s as the threat moves, always runs,
  and fails `stuck`, `deep_water` or `encumbered` as `move_to` does. It
  needs only `allowMovement`: running away is never the unsafe choice.
- `harvest` operates on one loaded tree, log, stump or mineable rock. It
  chooses the greatest resource damage among usable carried melee tools that
  meet `minTool`, requires at least 80% health and a work site clear of
  enemies within 15 m, approaches the resource's physical surface, and uses normal
  held primary-attack input with stamina and facing checks. Enemy proximity
  is rechecked every 0.25 s; any health loss cancels work immediately.
  No suitable tool
  yields `no_tool`. Wards, nearby characters and player structures can refuse
  the operation. For a stationary stump or small tree and an ordinary tool,
  a nearby player does not block swings when both PvP states are known and
  either player has PvP disabled. Unknown state, both enabled, animals,
  structures, mining and moving/falling resource hazards keep the exclusion
  guard. A tall tree needs room for its full fall radius and a
  supported escape. Moving logs must settle before work resumes; two
  seconds of quiet observed pose are required even with zero local velocity. Felling or
  splitting backs away before collection. New nearby drops matching the
  resource's actual drop table can be collected through normal pickup tasks;
  existing drop entities are excluded. Collection checks capacity/weight and
  is capped at 32 attempts/25 seconds after break. The whole task has a 180 s
  bound even with `timeoutS: 0`; caller timeouts can be shorter.
  Destruction requires a recent observed real local hit, revoked by any later
  observed competing hit. Remote destruction is correlated, not an
  authoritative final-hit receipt. Absence without that evidence reports
  `target_lost`. Cancelling clears held attack and any pending pickup.
  Choosing a tree and chaining newly spawned logs/stumps are the controller's
  job; this primitive does not silently harvest neighbouring resources.
- `attack` currently implements explicit bounded melee against one unstarred
  hostile Greyling. `style` defaults to `melee`; `auto` also selects melee;
  ranged is not yet supported. AllowCombat and AllowMovement must both be on.
  It uses carried ordinary melee weapons, moving-target aim, stamina-aware
  block/dodge and supported movement. Players/tamed creatures are refused.
  It starts at 80% health and accepts a target within 30 m. Outside 12 m,
  a closing phase walks within the initial 30 m area for at most 12 s,
  requiring 0.5 m distance progress every 4 s. On first reaching 12 m,
  the bot's position anchors the encounter's 12 m pursuit leash, 45 s work budget and
  12 s without observed target-health progress. Unsafe conditions trigger an
  ordinary bounded flee inside the same task; the encounter still fails and
  reports the escape result. Target unloading is target_lost, not a kill.
  Choosing whether to start a fight remains the controller/owner's job.
  Optional state.combat gives the attack task id, entity, mode and reason;
  matching IDs let retreat coordinate with the active encounter and fallback.
  Optional `kite:true` makes `entity` optional and delegates local ground and
  target selection, drawing, kiting and melee handoff to C#. Optional `at`
  names preferred nearby ground as an [x,y,z] position and requires kite=true.
  Encounter ground survey is capped at 25 s; acquisition waits at most 8 s once
  positioned. Kiting has a 40 m/40 s bound and confirms actual following.
  Modes include prepare_search, survey, position, search, draw, kite and hold before melee.
  Starting near buildings or on raised support first uses bounded ordinary
  floor-aware travel to dry open ground: eight candidates, 60 seconds and
  64 m total travel, within the search radius. No acquisition or swings occur
  during preparation; nearby creatures stop it. Ordinary travel can pass
  players, with 2 m separation at its destination. Arena checks still apply
  after discovery. A search that completes no checked leg reports unreachable,
  rather than claiming that no eligible creature was found.
  The same task ID and safety gates apply throughout. See SELF_DEFENSE.md
  for pacing, geometry checks, aborts and the remaining automatic policies.
- `controls` exists for debugging and later RL experiments. It's gated by
  `allowRawControls` and expires on its own, so a lost connection can't leave
  a key held down.

### 7.3 Movement inside the plugin

- **Dry home travel:** home returns request `dry:true`. Native movement uses
  validated route hints or incremental local routing with supported endpoints
  and midpoints at least 0.2m above water, including bridge/floor support. It
  rechecks the immediate movement corridor and cannot switch back to an
  unchecked navmesh shortcut through water. Stuck strafes receive the same
  check; jumping/climbing recovery is disabled for this mode. If already wet,
  the existing bounded shore search first targets dry land and continues out
  of shallow water before planning onward. Other movement keeps its default
  policy. Dry planning retains the existing node, time and per-frame limits.

- **Successful route hints:** completed point `move_to` tasks can retain
  and reuse their actual grounded trail (up to 256 points, 24 trails per game session).
  Failed/cancelled tasks and planned-but-unwalked routes are not retained.
  Floor-aware joins within 3 m of the start and 4 m of the destination allow
  the same entrance to serve nearby home chests, in either direction. Before
  reuse, every joined leg is resampled at most 0.5 m apart through current
  support, water, wall and capsule checks. Validation is limited to 512
  samples, eight per physics step with a 2 ms target budget. The immediate
  short leg is checked again while walking. Failed validation, changed
  geometry or lack of progress invalidates the hint and resumes ordinary
  planning. World/session reset clears hints; no route is persisted to disk.
  Reuse/rejection traces report sample counts and total/worst-frame costs.
  Follow, fleeing and other tasks retain their existing planning policy.
- Get a path from the game's own navmesh
  (`Pathfinding.GetPath(..., AgentType.Humanoid)`). Check every segment against
  the measured wall layout, with body clearance and wall height. Never steer
  past the last waypoint of a partial path toward the original goal.
- **Local routing:** when the navmesh has no path, ends more than 1.5 m short
  across the ground, crosses a known wall, or stuck recovery asks for another
  route, run local A* on 0.5 m cells. Ground probes preserve floors and
  enforce a 0.6 m maximum step; capsule probes check clearance.
  Point movement accepts a supported arrival inside the task's requested
  stopping distance, rather than requiring an occupied target's exact centre.
  Closed doors remain usable through the normal door handler. Planning can
  provisionally pass a closed door only with door handling enabled, valid live
  door data, ward/key access and no active refusal cooldown. Dry movement checks
  the actual door/frame colliders and waits for a bounded opening request;
  requesting an opening is not evidence of physical clearance. Open doors never
  exempt their frames or swinging leaves from physics checks. Nearby door scans
  and route-leg validation use current objects/geometry after rebuilding a base;
  old door refusals are cleared on world/session reset. Unknown support and water
  more than 0.5 m deep are refused. Each search is limited to 4096 expansions,
  48 m from its start on each axis, and 8 s. It advances at most eight nodes
  per physics step, with a 2 ms target budget, holding movement and the stuck
  clock meanwhile. Traces record probes and actual total/worst-frame time.
  For destinations more than 24 m away horizontally, a search may instead
  finish a supported leg at least 16 m from its start that reduces the
  remaining horizontal distance by at least 8 m. The final destination is
  preserved: reaching a leg endpoint triggers another plan, not task success.
  Intermediate heights follow live support; the final floor is still required
  for arrival. This permits progress toward unloaded distant destinations
  without enlarging the local search or trusting an unchecked straight line.
  A search that exhausts early waits at least 1.25 s to allow a retry of
  asynchronously requested navmesh tiles. The task's overall timeout still
  bounds travel, and obstructed or unknown ground can still stop the trip.
  Failure is `unreachable`, or `deep_water` if the original straight line
  crossed deep water. A swimming bot seeks shore before failing. Current
  physics overrides assumptions; local cells are not persisted across tasks
  or restarts, so demolished walls cannot remain stale route blockers.
- **Swimming:** a swimming bot walks, never runs. Below 30% stamina it makes
  for the nearest point it can stand on (searched in rings out to 60 m, the lowest on the nearest ring),
  whatever its goal, and the stuck count holds meanwhile. A point it gets no
  nearer to for 4 s, or reaches still swimming, it gives up, and searches
  again passing over points within 8 m of it. Out of the water it plans its
  way again, so a line back across the water is refused.
- Steering sets the move direction and look yaw every frame.
- **Stuck detection:** the bot is stuck while it stays within 3 m of where it
  was. Every 3 s inside that circle triggers the next recovery: a jump, then a
  strafe (at a walk), then a re-path. In the second and third rounds a climb
  takes the jump's place: the bot stands still until it has stamina for 3
  jumps (at most 10 s, and not at all while overloaded, when stamina does not
  come back), then jumps 3 times 0.8 s apart, walking on. The count holds
  while it waits. After 3 rounds the task fails with `stuck`. Hopping or shuffling against a wall does not reset it. Once a
  recovery has fired, leaving the circle only starts a new circle; the count
  resets when the bot gets 6 m from where the first recovery fired, so a
  strafe that carries it just over the edge and back is still stuck.
- `follow` keeps `distance` with hysteresis. It walks when the gap is under
  2× the distance, runs beyond that, and stops inside the distance. If a known
  wall blocks closing the last metre, it waits within `distance` + 1 m until
  its leader moves farther away.

### 7.4 Other messages

- `ping`
- `config`: changes `rates` and `perception` mid-session.
- `logout`, `bye` (section 5).

## 8. Safety (enforced in the plugin, not just the controller)

1. **Whitelist only.** The plugin carries out the tasks in section 7.2 and
   nothing else. No console or dev commands, ever.
2. **The plugin's own config wins.** `allowGather`, `allowDrop`, `allowCombat`,
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
8. **No destroying items.** There's no trash task. Items leave the bot only
   through `deposit`, or `drop`, which puts them on the ground where anyone
   can pick them up. Nothing equipped is ever dropped.

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
  `stop`, `status`, `come here`, `wait here`, `set home`, `go home`,
  `pick that up` (the drop nearest the speaker, within 10 m), `sleep` /
  `go to sleep` / `go to bed` (the bot's own visible bed, otherwise a confirmed
  free bed within 10 m of the speaker, or the bot when the speaker is out of
  sight; claim/set spawn first, then actually lie down), `claim the bed`
  (the nearest confirmed free bed in that same radius) and `get the door` (the nearest door, likewise;
  `open the door` takes the nearest shut one and `close the door` the
  nearest open one, since using a door swings it either way), and `get your
  things` (`fetch your things`, `corpse run`: the bot's own grave if it is
  in sight, emptied at once, else a walk back to where it last died and the
  grave there emptied, then any other of its graves there, since a bot
  killed over and over in one place leaves one each time; one gone is
  forgotten). These send `interact`, and the bot answers
  with what the game said: "Spawn point set", or "It won't: You need a roof".
  Only `owners` can issue tasks.
- `go chop wood` (`chop wood`, `gather wood`) is also deterministic: authorized
  command players start one controller-owned beech/log/gains-only-delivery cycle,
  with a return to the work site. Missing supplies, safe trees or storage are
  explained before work starts. The previous requested same-world site is saved
  in the manual once-only behavior; restarting does not resume it. `stop` cancels
  normally. See `docs/BEHAVIOURS.md` for current bounds and selection rules.
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
  stays at the spawn point. A death within 30 m of home keeps it from going
  back there, whatever the rule, since what killed it may be there still: it
  says so, and neither the respawn, idling nor an outage sends it home in
  that world until someone gives it an order, however it dies meanwhile;
  this is kept per world in `dataDir` across restarts, and any order in that
  world ends it. Every death is remembered per world (the
  last 5) as a grave to fetch from, until it is emptied or found gone. Any
  order given after the death wins, except
  `set home` and `status`. The controller reads the death and the respawn
  from `state.phase` rather than the `died` and `respawned` events, so a
  lost event does not lose them. The plugin reports `dead` from the death
  until the respawn wait ends, about 10 s.
- **Getting away.** A currently perceived hostile creature that damaged the
  bot in the last 30 s and remains within 15 m triggers retreat at any health.
  With only an AI target (`targetingMe`), health must be below 40%.
  The controller sends `flee`
  from the nearest such creature, 25 m, toward the nearest owner in sight
  within 60 m, else home if it is within 150 m and the creature not within
  15 m of it, else
  straight away (also when the owner or home lies past the creature, as the
  plugin judges it), and says which ("Falling back to you, Sothr", "Heading
  home", "Getting away from the greyling"). At an owner's side or home
  already, it runs straight away while the threat remains close. Damage
  evidence expires, is scoped to this world, and is cleared on death or
  respawn; only a still-perceived hostile creature counts. AI targets are
  not replicated to other clients, so a missing `targetingMe` flag does
  not imply that the bot is safe. Each flee has 60 s. It does not start another while a flee
  runs. Repeated decisions wait 10 s, but a fresh confirmed hit after a
  completed escape can bypass that delay. It needs `flee` in
  `allowedTasks`.
- **Carrying too much.** When the state says `encumbered`, the controller
  drops the least valuable, heaviest things first, just enough to be back
  under `weightMax`: common raw materials (wood, stone, flint, resin,
  feathers), then other materials, then food and potions, then trophies
  and other odds and ends, then valuables (`value`). It never drops anything equipped or any gear (weapons, tools,
  armour, shields, torches, ammunition), and drops nothing at all when even
  everything else would not bring it under; it says so instead. It shouts what it left, and keeps
  where and what per world in `dataDir`, shown by `status` and on the panel
  map, until perception, from within 15 m, has seen none of it there for
  5 s; a place where it sees fewer than it left now remembers that many.
  `overload.drop` (on by default) turns the dropping off; the plugin's
  `allowDrop` does too. It needs `drop` in `allowedTasks`, and the controller
  warns at start when `overload.drop` is on without it.
- **Runtime settings.** `owners`, `nicknames`, `allowedTasks`, the voice on
  or off, the voice's model (`model.provider`, `model.name`,
  `model.baseURL`, seeded from `llm.terva`; the key stays in the variable
  `apiKeyEnv` names), the `home` rules and `overload.drop` can change while
  the bot runs.
  The config file is the base. Changes are kept in `dataDir/settings.json`,
  each with who made it (`owner:<name>`, `panel`, `agent`) and when, and
  are written to the audit log as "setting changed". A reset returns a
  setting to the config's value. An owner or the panel can pin a setting,
  and the agent cannot change a pinned one, nor `owners` or `allowedTasks`
  ever. A change applies at once, with
  one exception: a task added to `allowedTasks` is only offered to the
  plugin in the next `welcome`, so it works from the next connection.
  Taking a task off the list works at once.
- **Saved behaviour definitions.** `settings.json` also holds named
  `instances`, each as `{value, by, at}`. The settings store saves/deletes
  JSON objects atomically, audits changes and returns independent copies.
  The behaviour engine owns template/parameter validation and execution;
  disabled or manual definitions do not run merely because they were saved.
  Enabled event/timer definitions are scheduled by the controller. See
  [BEHAVIOURS.md](BEHAVIOURS.md) for policies, ownership and controls.
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
| Interact | `Interactable.Interact(Humanoid user, bool hold, bool alt)` → bool, as `Player.Interact` calls it (private), then `Humanoid.DoInteractAnimation(GameObject)`. What the game says comes through `Player.Message(MessageHud.MessageType, string msg, …)`, localized with `Localization.instance.Localize` (assembly_guiutils). |
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

### Shared map position

Optional `sharePosition` booleans on `welcome` and `config` select the normal
Valheim shared map marker. Absence leaves it unchanged; explicit `false` turns
it off. `state.sharePosition` reports the observed native setting while in-world.
This is session configuration, so it neither replaces body tasks nor invalidates
automation. The plugin applies it only in a controlled session with its master
switch enabled, using the same game setting as the map's visibility checkbox.

### Bounded encounter search

A kiting `attack` accepts optional `searchRadius` (12–80 m). Native discovery
defaults to 60 m; a named target or fixed preferred `at` defaults to 12 m. A
fixed `at` cannot combine with a search radius above 12 m. Search uses checked
short walking legs; acquisition farther from the initial area triggers a local
fighting-ground survey. The local 40 m lure bound and melee leash remain
unchanged. The final receipt binds the selected entity as described in 6.6.

For distinct area travel, untargeted discovery accepts `searchCenter:[x,y,z]`:
native dry navigation travels to a terrain
entry within 8 m of that center before local scouting begins. `searchRadius`
then defaults to 24 m (12–24 allowed) around the requested center; the entry
height is measured on arrival. Travel has its own 90-second/200 m budget,
five entry candidates, and at most 140 m horizontal departure-to-center range.
Area scouting returns to its observed entry rather than the original departure.
An eligible Greyling found in transit triggers a local fighting-ground survey.
This option requires untargeted `kite:true`, and cannot combine with `entity`,
`at`, `searchSector` or `searchOffset`. Route refusal is not empty-area evidence.
Optional paired `searchAnchor:[x,y,z]` and `searchLimit` (positive, at most 500 m)
also bound actual search/travel against an absolute sphere, including detours.
The requested local area must fit inside it. Durable goals always supply their
immutable anchor and radius minus the 52 m combat/escape reserve; native code
checks this envelope every search/travel tick before acquiring an enemy.

The legacy `searchSector` option selects east
then successive 45 degree bearings toward positive Z. It requires `kite:true`
without `entity` or `at`. The native scout attempts three progressive distances
on that bearing and returns using the same checked short-leg policy and existing
budgets. Omission preserves the full local circuit. An empty attempt with at
least one completed checked leg and successful return reports failed /
invalid_target with `detail: "no eligible single Greyling after bounded sector
scouting and return"` and no entity. That is bounded search evidence, not full
sector coverage or a combat success. A search with no completed leg remains
unreachable; interrupted or failed returns do not produce this receipt.

Optional `searchOffset` shifts the three search distances outward by 0–60 m.
It requires `searchSector` and radius plus offset must not exceed 120 m.
For example radius60/offset60 plans distances80,100,120 m along the bearing.
These are horizontal hints, not fixed terrain-height destinations: native code
projects and physically checks each short leg. Outer travel can probe a finite
fan of progress-making alternatives, one per decision tick; the observed return
trail is revalidated without that fan. Existing time/travel/health/character
limits remain, with an earlier return allowance for long winding trails.

### Rendered camera images

Optional `screenshot` settings on `welcome` and `config` contain `enabled`,
`intervalS` (1–300), `width` (320–1280) and `quality` (30–90). New controllers
default to disabled, five seconds, width960 and quality70. Older plugins can
ignore this optional block. A new plugin emits `type:"screenshot"` while enabled
and in-world, with `world`, `worldUid`, UTC `capturedAt`, bot `pos`, `cameraPos`,
`cameraAngles` (Unity Euler degrees), optional `task`, `width`, `height`,
`captureMs` and base64 `jpeg`. Images are bounded to1280×720 and512 KiB before
base64, fitting the existing1 MiB bridge frame limit. The JSON fixture under
`testdata/plugin/screenshot.json` is shared by both codecs.

This is a latest-only visual observation, not a task or perception receipt.
Capture starts are limited to once per second even after settings changes;
there is one capture in flight and one low-priority pending transport image.
A reconnect drops pending images. The receiver validates JPEG dimensions and
world/connection scope. Audit persistence and panel event streams strip `jpeg`,
retaining metadata only. Authenticated media retrieval lives outside timeline
responses, avoiding base64 images in routine diagnostic context.
