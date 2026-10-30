# Testing renki in the game

Everything that can run without Valheim is covered by `just ci`. That covers
the controller, the protocol on both sides, the plugin's task runner, safety
rules and movement maths, and the C# bridge client talking to the real Go
controller. This page covers what only the game can show. Each check names the
ticket it closes. Tickets in `review` are waiting on these checks, and
`git ticket list --status review` lists them.

## Development checks with concise output

Use `just check <recipe> [arguments...]` to save full output in a unique,
gitignored `.checks/` log. Success prints one result line; failure prints the
last 80 lines, the full log path, and returns a nonzero status. Read the
relevant diagnostic from that log before rerunning a check. Logs remain until
you remove them; avoid dumping whole logs into an agent conversation.

- Controller iteration: `just check controller test-packages ./internal/home ./internal/chat`.
  Package arguments can include test flags, for example `-run TestRespawn`.
- Movement iteration: `just check plugin test-core ApproachRecoveryTests`.
  Without a class name, `test-core` runs all tests except interop, without
  building a controller. `just plugin test` still builds it and runs interop.
- Format once after editing: `just check fmt`.
- Before committing: `just check ci` runs the complete `just ci` gate.
- After committing and before pushing: `just check review` retains the review
  locally. Review exit 1 means findings; exit 3 means the review failed.

Keep routine investigation narrow: use a file/line search first, read the
relevant function, and inspect retained task/trial summaries before fetching
another audit window. Do not tail a trial's `commands.log`: one line can contain
a large map or history response. Use bounded evidence recipes below instead.
Keep durable decisions and remaining game checks
on the ticket so a resumed session can read those instead of old transcripts.
Rerun checks after changes or new evidence, rather than repeating passing
suites while waiting for the game. The complete gate is still required.

For a single live task, `just vm task-check <task> '<args>' <timeout>` sends
once, waits for its terminal status, saves the complete task history in
`.checks/`, and prints only the result and evidence path. Its query timestamp
selects the current controller run so recycled task IDs cannot select an older
run. It exits nonzero for refusal, failure or a wait limit; it never resends or
cancels a task. Use a finite positive timeout and inspect retained diagnostics
before retrying. The audit summary below adds pack/chest and health evidence.

For a short incident that needs state and perception as well as decisions,
`just vm audit-window <since> <until>` saves only that ISO timestamp window
to `.checks/` and prints its path. `just vm audit-summary <since> <until>`
prints bounded task statuses, movement/harvest traces, minimum health and the
last pack/chest counts from that same retained evidence. It keeps only the
last 20 task events and 10 traces in its output; narrow the time window when
those are insufficient. Inspect selected fields locally instead
of downloading or displaying a whole day. `just plugin game-source <Type>`
saves the installed game's decompiled API under `.checks/game-source/` for
local inspection; do not commit or redistribute those files.

## Terrain map check

Use `just check controller test-map` for interpolation, negative grid positions,
unknown samples and shoreline clipping, and `just check plugin test-core TerrainCacheTests`
for cache reuse, bounded per-step sampling, refresh, invalidation and resend.
Both are included in `just check ci`; Node.js 20+ and npm are needed for the Preact panel and map maths.

`just controller preview-map --serve` opens a read-only fixture preview at
127.0.0.1:18780 using the actual panel assets. `--status <status.json> --map
<map.json>` uses captured `/api/status` and `/api/map` snapshots instead.
Add `--camera <recorded.jpg>` to exercise real image decode, or `--offline` for
saved-map fallback. Run `await renkiPreview.check()` in the native browser
console for fixture-only responsive/control checks (see PANEL.md).
It creates `.checks/map-ui-preview.html`, works without Valheim or credentials,
and rejects task actions. Use the browser console for bounded fixture checks.

In the panel, verify the biome and support at home, with character height
above the underlying terrain height. Toggle Terrain/Elevation/Remembered/
Resources and inspect the compass, scale bar and marker hovers. Pan beyond
sampled tiles to confirm unknown areas stay dark. Inspect a shoreline from
dry land; compare water shading and contour levels to the game. A loaded
terrain sample cannot establish floor, bridge, cave or navigation clearance.

With the owner present, landscape a small safe area and confirm the changed
heightmap refreshes its tiles. Check shallow water and swimming, then return
to dry land and check the location readout. Restart the controller and
reload the page; the same stable world/epoch should restore retained terrain.
After logout, confirm Follow Tas automatically retains its fogged saved atlas;
reload offline to check fallback, then rejoin to check live mode returns. A
manually selected saved world should stay pinned on rejoin. Live paths and
markers must clear. A different world/epoch must not inherit the old terrain.
Inspect the terrain cache diagnostic for maintenance time and worst frame.

For bounded repeatable persistence evidence, run `just vm atlas-check` after
fresh terrain arrives. It stores full API responses under `.checks/` and prints
sample/tile counts, health, version and the evidence path. Run
`just vm atlas-check --baseline <that-path>` after `just vm reconnect`, logout,
or a game restart. It queries the original saved world/viewport and requires
all baseline canonical samples to remain with no older measurement times;
changed heights/biomes are counted. It also detects conflicting overlapping
transport borders. The helper is read-only: it neither starts the game nor
changes tasks, settings or terrain. Age/capacity expiry can legitimately make
a long-delayed baseline fail; use a short controlled window for restart proof.


## Food recovery check

With Tas safe at home, withdraw a small amount of ordinary food from an
owner-authorized chest, preserving unrelated items. Use `just vm task use_item
'{"item":"Raspberry"}'` (replace the prefab with the actual food).
Confirm success reports one consumed and matching active food, inventory falls
by exactly one and health rises through normal regeneration. Retry the same
food immediately: it must refuse without another inventory decrease. Check a
missing prefab and a carried non-food item refuse, then verify full food slots
refuse a fourth fresh food. Do not resume harvesting below 80% current maximum
health.

Use `just check controller test-packages ./internal/food ./internal/app
./internal/protocol` for automatic upkeep policy, action sequencing, stale
state/connection guards, permission checks and owner-request rate limiting.
With eligible spare meals carried at home, observe automatic `use_item`
success, one-item consumption and updated `food.active`/`food.carried` in
`just vm panel GET /api/status`. Let health regenerate to at least 80% of its
new maximum before work. Verify an existing movement task is preserved. Owner
requests need an injured, foodless live case; do not injure Tas to stage it.

## Chest-transfer check

Use a storage chest visible in perception and only Tas's test items.
After adding a task to allowedTasks, `just vm reconnect` refreshes the
controller/bridge welcome without reloading Valheim. This stops active tasks.
`just vm task deposit '{"container":"<id>","items":[{"prefab":"BeechSeeds","count":5}]}'`
returns the task ID. Confirm a `container` snapshot reports exactly five
moved and the pack loses five, then use `just vm task withdraw` with the
same arguments to take those five back. Inspect task results with
`just vm panel GET '/api/task?id=<id>&at=<ISO-time>'` or a narrow
`just vm audit-window`. Check pack + chest counts before/after.

Repeat with a chest that has only partial room, one held open by Nimah,
and a ward Tas cannot access. Check partial counts or explicit refusal,
no inventory UI, and that the owner can open the chest afterwards. Cancel
while approaching or waiting for access and confirm no late transfer or
use lock remains. `just vm gather` reads the plugin safety switch;
`just vm gather true` enables gathering after the next restart when that
work is authorized. It does not bypass controller allowedTasks.

### Live seed-transfer verification, 2026-10-29

On `a04704d`, deposit `t-1` moved five BeechSeeds into chest `1:30592`
at 18:11:52Z, changing the pack from 15 to 10. Withdraw `t-2` took those
five back at 18:12:02Z, restoring pack 15 and the chest's original contents
(one rag chest armour). No unrelated items moved.

Empty chest `1:30601` was observed with a request to move all of an absent
test prefab (`t-3`, no mutation). Requesting 20 seeds (`t-4`) moved the
available 15 and reported exactly five remaining, failing `invalid_target`
for the missing source items. Withdraw `t-5` restored all 15 to the pack
and left that chest empty. This verifies conservation and a partial result
from missing source items; destination capacity, busy and ward refusals
remain staged live checks.

With Nimah holding chest `1:30592` open, deposit `t-1` in the next controller
run (18:18:12Z) failed `refused`: `the chest is in use`. Tas retained all
15 seeds. Destination capacity and ward refusal still need staging.

## Wall-aware route check

From inside the raised home, stand outside the east wall and say `come here`.
The walker should reject the direct or partial navmesh path, trace `planning
a local route`, then `local route found`, leave through the west doorway and
walk around the building. Inspect the planning trace's `expanded`, `probes`,
`ms` and `worstFrameMs`. Repeat a return home and a follow across the doorway.
Try stopping during planning, a blocked doorway, and a newly built/demolished
wall. A partial path must end in a replan or an explained failure, never a
straight drive through the remaining wall. These checks need the game: the
Core regression uses the measured house walls but cannot establish physical
support, stair clearance or door response.

For distant cold-start routing, use `just check vm resupply-food` when Tas is
away from home and needs food. Local-route traces may show `progressLeg:true`
with a short checked `end`; the `to` remains the real destination. Verify
measurable travel and repeated replanning, then actual final-floor arrival,
task-matched food withdrawals and observed eating. A short leg alone is not a
successful home return. The deterministic regression covers the recorded
375 m hillside trip, a wall detour, enclosed starts and unknown support; native
terrain, tile availability, doors and the raised home still require this check.

### First wall-route check, 2026-10-29

On `fe9384a`, come-here task `t-1` started at 17:57:37Z from
`(-36.8, 36.09, 7.64)` and reached Nimah east of the house at
`(-0.93, 33.25, 9.02)` at 17:57:53Z. It left through the west doorway.
The local route used 1121 expansions / 8961 probes, 238.69 ms total CPU
over 2.8 seconds; the slowest planning frame took 12.95 ms.

Return-home task `t-2` failed at 17:58:12Z: it spent its entire eight-second
local-search budget starting 35 m away (3207 expansions, 513.4 ms CPU,
2.11 ms worst frame). The navmesh already supplied a wall-safe route to
`(-37.47, 34.14, 6.13)`, under the raised floor. Preserve useful, wall-safe
navmesh prefixes and start local search near their ends. Stop and replan
there; never invent a final segment through a wall or raised floor.

On `dd9ef44`, the first order after restart again had no usable navmesh and
exhausted local search. Retrying home at 18:03:30Z used the safe prefix,
repositioned from beneath the floor and found a 29-waypoint local route
(434 expansions, 84 ms total, 2.28 ms worst frame). After stuck recovery
and a second short local plan, task `t-2` succeeded at 18:03:59Z, ending
at `(-36.5, 36.09, 8.26)`, 25/25 HP. During local planning, recheck once
per second for newly built usable navmesh paths without resetting the
search's existing budget when no safe path is available.

On `a04704d`, exact east-side walk `t-6` (18:14:37Z–18:15:08Z) again
left through the west doorway. After recovery at the door, the trace at
18:14:52Z recorded `navmesh became usable during local search`, exercising
the asynchronous recheck. Return home `t-7` (18:15:31Z–18:15:44Z) succeeded
at `(-37.03, 36.09, 7.92)`. Tas finished at 25/25 HP, with all 15 seeds.
Close-follow waiting across a wall, cancellation during planning and changed
or blocked walls/doorways still need separate live checks.

### Cached local-planner geometry

The plugin retains nearby upright box and capsule solids in a spatial index.
Discovery runs near Tas (10 m, due every 0.25 s), ahead along the travel
direction (12 m around a point up to 16 m ahead, 0.5 s), and farther out
(40 m, 2 s). At most one scan starts per rendered frame; the oldest due
pass wins and results are processed across frames. Near objects have first
refresh priority, then the forward corridor. Unchanged geometry is reused.
Changed geometry drops its old bounds before a queued rebuild. The cache
holds at most 4096 records, rebuilds at most 16 per frame, sweeps at most
32 per frame, and evicts records unseen for 10 s or beyond 64 m. Processing
scan results and rebuilds targets 0.75 ms combined; Unity's broad-phase
query itself and the sweep can exceed that target and are measured.

Cached bounds only reject legs, and a candidate blocker must still have
the same live transform/shape and remain enabled. Doors, rigidbodies,
tilted/sheared boxes, complex meshes and unindexed space use live physics.
Exact support samples are shared within one planning physics step, with
different heights kept separate; the next step resamples even a previously
missing floor. Session/world changes clear geometry and buffers. Planner
physics queries use fixed buffers and reject a leg on saturation.

Run `just check plugin test-core NavigationCacheTests` for deterministic
geometry, invalidation, support lifetime, priority and detour regressions.
These establish fewer simulated live checks on an identical route; actual
Unity CPU savings still need the repeated house exit/return check. Compare
`supportHits`, `supportQueries`, `boundsRejected`, `physicsCasts`,
`saturated`, `expanded`, `ms` and `worstFrameMs` in the walker result trace.
The `navigation` / `obstacle cache` trace every 30 s reports occupancy,
pending work, rebuilds, removals, discovery saturation and maintenance CPU.
Try demolishing/building a wall or floor and opening/closing the door
during planning; no stale blocker should survive a freshness check, and
an uncached obstacle must still be checked by physics. Also confirm that
stopping during planning stops Tas. These live checks are deferred until
the owner is available; do not count them as passed from Core tests.

## Raised-goal recovery check

For TKT-01M647SMRM3XXAQS7Y9CW2TKZ9 (Retry raised goals from other approaches
before giving up), place the bot directly beneath the raised home floor and
send it home. It should trace `off-level goal; repositioning`, walk away or
around the goal on a complete navmesh path, then trace `retrying path` for
the original home position. A usable replan should reach home; an inaccessible
floor should end after at most three approach attempts or 30 seconds of this
recovery. If no complete water-safe detour exists, it waits between path
retries instead of walking a blind detour. Cancel during a detour and confirm
the bot stops. Also try a detour beside deep water and confirm it is rejected.
These require Valheim; unit tests cannot establish whether a house's stairs
or floor appear in its navmesh.

### Live verification, 2026-10-29

Deployed `6baa828` after merging PRs #5–#9. The panel reported that version,
and Tas rejoined DizzysHellheim at 25/25 health.

- Two ordinary walks home from different directions succeeded (tasks `t-3`
  and `t-6` in the controller run started by this deployment).
- With Nimah beneath the house, `come here` brought Tas down to about
  `(-39.38, 33.83, 7.02)`. Sending him home to `(-35.9, 37.66, 6.53)`
  started task `t-8` at `2026-10-29T16:56:54.668Z`. He walked out from under
  the building, climbed back up, and succeeded at `16:57:04.192Z` (9.5 s).
  This run found a complete navmesh route; the separate off-level detour,
  retry-limit, water rejection and timeout branches have automated coverage
  but were not observed live.
- Follow task `t-4` ran and was cancelled with `stopped` by the panel Stop
  button. A subsequent home task succeeded.
- The bot's tombstone disappeared before interact task `t-1` reached it;
  the task reported `target_lost` and the controller explained that the
  grave was gone. Its items had already been recovered and stored by the
  owner. A later `get your things` with no remembered grave sent no body
  task and explained that there was nothing to fetch.
- Tas subsequently died to a Greyling at about `(-27.67, 30.51, 40.38)`
  (`16:59:35Z`), roughly 35 m from home and outside the 30 m home-shun
  radius. He respawned about ten seconds later. With `afterDeath=home`,
  task `t-11` took him home and succeeded at `17:00:20.712Z`. He ended idle
  at 22/25 health. His inventory was empty before this death, so it did not
  provide a populated tombstone for a positive retrieval check.

The next staged death carried 15 BeechSeeds. Interact task `t-18` reached
the populated grave at `(-16.34, 31.76, 33.26)` but never used it and ran
through stuck recovery instead. The interaction sight check now follows
Valheim's hover rule: ignore the player's own rigidbody and use the nearest
other ray hit, retaining wall and floor blocking. Regression tests cover
that rule. The fix landed in PR #10 and was deployed as `d369b07`.

### Populated-grave verification after the interaction fix

- An initial attempt died near the house exit before reaching the seed
  grave. After respawning, the bot logged `died near home; not going back`
  and waited, including through later deaths at spawn. Explicit owner
  commands resumed movement. The owner cleared the attacking Greylings
  before the successful retrieval.
- On deployed `d369b07`, `get your things` started interact task `t-10` at
  `2026-10-29T17:17:40.410Z`, targeting the same untouched tombstone
  `2344638503:696`. At `17:17:51.245Z`, the task succeeded and the bot said
  `Got my things back.` The tombstone disappeared and inventory reported
  exactly 15 BeechSeeds.
- An explicit panel Go home started task `t-11`, which succeeded at
  `17:18:20.151Z`. Final status was idle near home at
  `(-36.79, 36.09, 7.62)`, 25/25 health, with all 15 seeds still carried.

TKT-01M62N12MX (Controller: corpse run after a death) is complete: the
on-demand retrieval and explicit return home, plus staying away from a
home that killed the bot until ordered, were verified live. Resuming a
follow through an actual death remains covered by automated tests; it was
not staged in these live checks.

## 1. Set up a game with BepInEx

Use the dev box's own Valheim for a first pass, or the bot VM once
TKT-01M61T36CT (Set up the bot VM) is done.

1. Install [BepInExPack_Valheim](https://thunderstore.io/c/valheim/p/denikson/BepInExPack_Valheim/)
   into the Valheim folder, or into an r2modman profile. Start the game once
   so BepInEx creates its folders, then quit.
2. Tell the build where the game is: `plugin/game.local.props` (see
   `plugin/Directory.Build.props`), or `VALHEIM_DIR`. If the plugins folder is
   not `<game>/BepInEx/plugins`, as with an r2modman profile, set
   `BEPINEX_PLUGINS` too.
3. `just plugin deploy` copies `Renki.Plugin.dll` and `Renki.Core.dll` into
   `BepInEx/plugins/renki/`. The game supplies Newtonsoft.Json and BepInEx
   supplies Harmony.

On the VM, `just package` builds `dist/renki-vm/` instead: the Windows
controller, the plugin DLLs, the start and stop scripts, and example configs.
Copy the `plugin/` DLLs into `BepInEx/plugins/renki/`, and the rest anywhere.
Windows refuses scripts that were copied in from elsewhere, so either run
`Unblock-File *.ps1` once in that folder, or start them with
`powershell -ExecutionPolicy Bypass -File .\start-bot.ps1`.

## 2. Run the controller

Copy `controller.example.json` to `controller.local.json`, set `owners` to
your character name, and either fill in `llm` or delete it to run on the
command grammar alone. Then run `just controller run` on the dev box, or
`start-bot.ps1` on the VM, which starts the controller itself.

You can check the controller without the game: `just controller fake` turns
stdin lines into chat from `Sothr`. A line `Name: text` speaks as Name, and
`/whisper text` whispers.

## 3. Launch the game so it joins by itself

Valheim joins a server and picks a character from its own arguments:

    valheim.exe -joinserverwithcharacter <host:port> <character-file> -password <pw>

`<character-file>` is the save file's name without extension, usually the
character's name in lower case. The game ignores the argument unless another
one follows the character, so with no password put any extra argument there.
`start-bot.ps1` does this for you from `bot.local.psd1`.

The plugin writes to `BepInEx/LogOutput.log`; every line starts `[renki]`.
The controller logs to stderr, or to `scripts/vm/logs/` on the VM.

## 4. Checks

Work down the list. The first failure usually explains the ones after it.

| # | Do | Expect | Ticket |
|---|---|---|---|
| 1 | Start the game with the plugin installed | `renki 0.1.0 loaded; dialling ws://…` in LogOutput.log | TKT-01M61T36H, criterion 4 |
| 2 | Start the controller with the game at the menu | Controller logs `plugin connected` and `session started`; plugin logs `session started` | TKT-01M61T36CC, criterion 1 |
| 3 | Stop the controller for 30 s, then start it again | Plugin logs `disconnected; retrying`, then a new session within 30 s | TKT-01M61T36CC, criterion 2 |
| 4 | Launch with `-joinserverwithcharacter` | The character is in the world with no clicks; with a password set, no prompt | TKT-01M61T36P8, criteria 1 and 2 |
| 5 | Say `bjorn status` (as an owner) | A status line in chat with health, biome and time | TKT-01M61T36M4 (state), TKT-01M61T36QB |
| 6 | Say `bjorn follow me` and walk 200 m through trees and uneven ground | Follows at about 3.5 m, runs to catch up, sprints when you sprint | TKT-01M61T36N6, TKT-01M61T36NQ criterion 1, epic criterion 2 |
| 7 | Block it in (a fence or a pit) and walk away | Jumps, strafes and re-paths, then says it is stuck | TKT-01M61T36N6 criterion 3 |
| 8 | `bjorn stop`, `bjorn come here`, `bjorn wait here` | Stops; walks to where you stood; stays put | TKT-01M61T36QB, epic criterion 3 |
| 9 | Log off while it follows | Within about 5 s: "I lost track of you" | TKT-01M61T36NQ criterion 2 |
| 10 | Someone who is not an owner says `bjorn follow me` | A polite refusal; nothing moves | Epic criterion 3 |
| 11 | Ask it something (with `llm` configured) | An in-character reply; "wave at me" also waves | TKT-01M61T36QX, epic criterion 4 |
| 12 | Kill the controller while it follows | It stops within 10 s and stays stopped after the controller is back, until told again | TKT-01M61T36MN criterion 4, epic criterion 5 |
| 13 | Get it hit, killed, and respawned | `damaged`, `died`, `respawned` in the controller log at debug level (`-debug`) | TKT-01M61T36M4 criterion 3 |
| 14 | Someone joins and leaves | `player_joined`, `player_left` | TKT-01M61T36M4 criterion 3 |
| 15 | Run `stop-bot.ps1`, or `POST /control/logout` | The character saves and returns to the menu, and does not rejoin | TKT-01M61T36P8 criterion 3, TKT-01M61T36PT criterion 3 |
| 16 | Unfocus, minimise, and disconnect RDP while it follows | It keeps following in all three cases | TKT-01M61T36D |
| 17 | `tas set home`, walk 30 m away, `tas go home` | "This is home now."; it walks back to where you stood | TKT-01M628X11 |
| 18 | `tas follow me`, then log off | "I lost track of you. Heading home.", and it walks home | TKT-01M628X11 |
| 19 | `just vm restart` while it follows | Back in the world: "Something cut me off. Heading home." | TKT-01M628X11 |
| 20 | Leave it away from home with nothing to do for 10 minutes | "Nothing to do. Heading home." | TKT-01M628X11 |

## Things worth watching

These values were set from reading the game's code, not from play, and may
need tuning:

- Turn rate: `Driver.TurnRate`, 540°/s.
- Waypoint reach: `PathFollower.Reach`, 1 m.
- The sprint speed above which the leader counts as running:
  `FollowTask.RunningSpeed`, 5.5 m/s.
- Whether `Pathfinding.GetPath` fails on the first call while the navmesh
  tiles build. The bot falls back to a straight line and re-paths when stuck.
- Chat range: a `normal` line only reaches the bot when you are close, so
  command it from afar with `/s` (shout): `/s Tas follow me`.
- Under Unity's Mono, the WebSocket client is the one unknown. If check 2 or 3
  fails, the plugin log will say so, and websocket-sharp is the fallback
  (TKT-01M61T36CC).

When a check fails, note it on the ticket it names, with the log lines.
`just vm audit 1 -task t-6` (docs/AUDIT.md) prints everything about one
task, from the chat that started it to how it ended, including the plugin's
own decisions along the way.

## Bounded collection/delivery check

Use `just check controller test-packages ./internal/behaviour ./internal/app ./internal/protocol`
for stow/collect policy and bridge regressions. Shared perception-yield wire
fixtures run through Go and C# in `just check ci`. Save disabled manual
instances through `/api/behaviours` as documented in BEHAVIOURS.md.

First check a `stow` job with Tas's test seeds and a known chest, preserving
at least ten seeds through `keep`; verify the task-matched chest receipt,
pack counts, and unchanged unrelated items. Restore test seeds with
`just vm task withdraw`. Repeat using a full/busy first chest and another
available destination. Check a nonzero storage-needed report, its deduplicated
announcement and panel visibility, then make room and rerun. Stop while
approaching or waiting for an observation; no later step may run.

Stage a small safe area containing loose stone, flint or branches. Record
initial pack/chest counts and current resource IDs/yields. Configure explicit
world/site/chests, one or two target attempts, `drops:false`, and a carry
reserve. Start `collect`, observe interaction/product pickup, exact deposits
and optional return to the site. Confirm unknown yields, unselected materials
and targets beyond the radius are not chosen. Clear threats beforehand and
avoid deep-water flint; the normal walker may refuse a wet route. Do not
claim tree-felling/mining, cache invalidation, chest capacity/wards or retreat
verification from this collection check.

The initial seed-stow check on cf04d60 failed before dispatch: the first
inventory arrived while state was still connecting, and the subsequent
world transition cleared it. No seed/chest mutation occurred; Tas remained
25/25 HP. ObservationSchedule now forces state before world observations
on every phase/world/player transition, with welcome resetting its deadline.
`just check plugin test-core ObservationScheduleTests` covers the scheduling
race. After deployment, verify the first in-world state precedes inventory
and a manual stow works without waiting for a pack mutation or reconnect.

### Collection/stow live checks, 2026-10-29

- Seed stow passed on 91b4835 without a reconnect or pack mutation: deposit
  five BeechSeeds, keep ten, then withdraw five to restore all 15. The chest's
  rag armour was unchanged. Initial state preceded inventory after the
  startup scheduling fix.
- Stone collection/delivery/return passed at 22:04:58Z–22:05:12Z after explicit
  doorway opening/repositioning. This was a controlled loop, not an
  unattended house-exit pass.
- Resumed flint delivery passed at 22:14:31Z–22:15:14Z: direct chest approach
  failed, one configured-home fallback entered through the west door, and
  the matching receipt deposited both previously collected flints.
- Fresh flint collection/delivery passed at 22:23:51Z–22:25:01Z on the
  height-aware waypoint build. Tas left home without manual doorway staging,
  used pickable 1:39745 at (-28.15,30.43,42.11), picked up Flint1, recovered
  from failed direct chest approach through configured home, and deposited
  Flint1. Receipt t-7 and newer inventory showed chest 1:30592 holding
  Stone1/Flint3/ArmorRagsChest1, pack BeechSeeds15. All states were 25/25 HP.
  Both temporary check definitions were deleted afterward.

The flint loop proves bounded recovery, not efficient direct chest routing:
Tas still approached the underside before using home. The path planner
retains floor heights; the follower now also requires level-aware arrival
for intermediate/final waypoints and replans when standing below one.
`just check plugin test-core LocalRouteTests` covers stacked-floor arrival.
Repeated unattended routes, full/warded and multi-chest overflow checks,
deliberate-drop detection, and leader/site
resumption remain open. Tree cutting and mining primitives are described below; crafting remains unimplemented.
Exact receipts, audit-window paths and remaining checks are on the stow,
hauling and navigation tickets; keep full logs outside this document.

### Bed/sleep checks

Use `just check controller test-packages ./internal/chat ./internal/protocol`
for sleep aliases, owner checks and safe bed selection. The shared bed
fixture distinguishes owned, mine, explicitly unclaimed and unknown beds.
`just check plugin test-core BedSleepTests` checks bounded ownership waiting
and that claiming alone never establishes sleep. The full gate checks the
additive interact sleep arguments in both languages.

Live: inspect ownership first. Tell Tas `sleep`/`go to sleep`; expect claim or
set-spawn if needed, then lying down, reported as `in_bed`. Actual `sleeping`
waits for the other players. Taken/unknown beds must not be selected for
claiming. Check the game's roof/fire/wet/enemy/night refusals and stop during
a claim wait; no later use may occur. Preserve players' beds.

At 22:22:51Z–22:22:52Z, `sleep` selected free bed 1:38548, set spawn, confirmed
ownership and tried sleep. The game refused "You can't sleep at this time"
(day 51, fraction about 0.34). Perception then showed `mine:true,owned:true`;
bed 1:38536 remained someone else's. This exposed an interaction reach bug:
the bed's broad effect-area trigger counted as a nearby collider. Interact
now excludes triggers from reach and sight. At 22:28:17Z–22:28:19Z, the final
build walked Tas from (-36.53,36.09,7.96) to (-32.03,36.09,11.97), beside the
physical bed collider, then relayed the same daytime refusal. HP stayed 25.
Nighttime bed entry passed on 2026-10-29 at 22:40:47Z with bed
`1:38548`; Tas entered `in_bed` at full health. Shared time-skip, automatic
home/join/respawn claiming, persistent bed status and actual respawn-at-bed
remain staged work.

### Successful route reuse

`just check plugin test-core RouteMemoryTests` covers reversible entrance
trails, joins to nearby home/chest goals, stacked floors, blocked/new joins,
demolished support, one-way drops, bounded work/storage and backtracking.
These are deterministic policy/probe checks, not Unity physics proof.

Live: with threats clear and Tas healthy, move from inside home to nearby
open ground and back, then repeat. Capture `recorded successful route`,
`reusing recorded route` and rejection traces with exact task receipts and
HP. Verify a reusable route crosses the actual doorway and avoids repeating
the underside approach. Compare validation samples/CPU with local-search
expansions and total/worst-frame cost. A one-way fall may legitimately reject
reverse reuse; normal planning must still work. Stop/death/reconnect must
discard the active task, and reconnect starts without saved route hints.
Do not remove a player's wall/floor or change ward access merely to stage an
invalidation test; keep those live mutations explicitly unverified.

On 2026-10-29, level-terrace outgoing and reverse chest routes both reused
recorded trails through the west doorway without fallback (45/48 samples,
1.61/1.84 ms total validation, HP 25). A separate steep exterior start stalled
and safely fell back to normal planning. That slope and real geometry
mutations remain open; exact receipts and costs are on the navigation ticket.

### Supplied-tool harvesting

Use `just check plugin test-core HarvestTests` for resource-specific tool
selection, unavailable/broken tools, destruction correlation, permission
requirements and task cancellation. Protocol checks cover default collection
and explicit `collect: false`. These do not prove Unity attack hitboxes,
network attribution or falling-log safety.

The reach regressions use the recorded raised-house and failed small-tree
stances. In game, start inside the raised home and issue `just vm task-check
harvest '{"entity":"<current-live-id>","collect":true}' 90` without an outside
waypoint or manual closer move. Confirm the approach uses outdoor support and
physical collider contact, then measure new products, deposit only those gains
and return home. Missed swings permit one closer supported approach before
failure; this needs real collider verification as well as the offline checks.
For stumps, verify Tas stays beside the resource, aims down from the tool's
attack height and does not use jump/climb recovery during the final approach.
After a restart, active food prefabs must remain stable rather than becoming
localized name tokens; eligible spares should still be recognized as refreshes.
PvP policy checks cover both states enabled, either disabled, unknown state
and environmental hazards. In game, confirm Tas and nearby Nimah expose
their actual `pvp` flags. A stationary stump/small-tree swing may continue
near a player with known PvP protection; full trees/logs, mining, animals and
player structures retain their proximity guard. Do not stage damaging PvP
hits to validate the guard.

First verify no-tool refusal on a loaded mineable rock while Tas has an axe
and no pickaxe. Then use a small isolated tree away from the home/players,
check actual swings and tool wear, measured wood gain and unchanged test
seeds, deposit only that gain and return. Use an isolated full tree for the
separate fell/escape/settle/log-split benchmark. Observe each descendant by its
new entity id; never infer an arbitrary neighbouring tree as a descendant.
Capture exact windows with `just vm audit-summary`. Stop on health loss or
new threats, and record unverified falling/rock/controller-chain checks on
the harvest ticket. No default harvest behaviour is installed by this task.

The first supplied-axe small-tree trial on 2026-10-29 produced two wood and
used one durability, but an approaching Greyling interrupted task completion
and the return encountered another attacker. It does **not** pass the full
benchmark. Tas was logged out and stopped at 2.7/25 with AxeFlint 1,
BeechSeeds 15 and Wood 2 preserved. The owner subsequently cleared the route.
Post-incident health/proximity guards and first-hit retreat still need live
validation. Recover with normal food before another harvest; task success,
delivery/return, full-tree escape, log splits and mining remain open.

On 2026-10-30 UTC, build a1a1a41 completed an unstaged raised-home stump
loop: approach Beech_Stub 1:39764 from inside, break it after four observed
local hits, collect two new Wood, return to the named home storage waypoint,
deposit exactly those two Wood and return home. Original Wood 2, seeds 15
and the axe remained in the pack; the chest reached Wood 69. Tas stayed on
terrain beside the stump. The house exit still needed stuck recovery and
replanning; full-tree escape, log splits and mining remain unverified. Exact
task receipts and audit windows are recorded on the harvest ticket.

### Places and zones

Use `just check controller test-packages ./internal/home ./internal/chat
./internal/session ./internal/app ./internal/behaviour` for world-scoped
persistence, owner commands, deletion suppression, save rollback, stale edit
conflicts, evidenced standard-place selection, remembered nearby contents
and named movement resolution before bridge dispatch. World changes during
resolution must send nothing. `just check ci` also checks panel JS syntax.

`just controller preview-map --serve` includes automatic storage and an
owner-managed woodlot. In the browser, open Places & zones, select a marker,
rename/resize it, and preserve a raised floor's Y while changing X/Z. Add a
zone with map placement, both preserving height and using measured terrain.
Verify nearby known/empty/unknown chests, save/delete/go requests, an unsaved
edit surviving a map refresh, and a delayed save response discarded after a
world change. The preview remains read-only; isolate mocked API responses
in the browser for edit-flow checks. Never send fixture tasks to the game.

In game, confirm observed purposes are created, owner edits survive controller
restart, deletion does not respawn an automatic role, and a named move reaches
its saved floor. Confirm cooking equipment is visible as a station. Creating
a zone must not start harvesting. Keep full harvesting/log/rock benchmarks
and unverified game cases on their tickets.

The a1a1a41 live check on 2026-10-30 UTC observed automatic storage,
cooking, flint and stone places. Create/rename/move/resize, persistence over
controller restart, suppression of a deleted automatic storage role,
remembered nearby chest contents and publishing places on the map passed.
Named movement reached the raised indoor floor; saved-place Go also passed.
Temporary test places were deleted. The retained home storage zone is
centered on its accessible indoor approach. Zone edits started no harvest.

### Autonomous full-tree/log benchmark

`just vm harvest-trial --wait-s 60` uses normal native harvest tasks with combat
disabled, without owner staging. It requires active food, 80% health, 40 stamina,
a usable carried axe, four spare slots and 60 spare carry weight. It selects one
nearby tier-zero full Beech1, with a conservative 32 m falling exclusion from
all characters and structures (8 m for settled logs), plus 15 m around the bot.
The native task owns approaching, aiming, swinging, falling-trunk escape and
collecting. The supervisor issues no movement tasks.

The benchmark then considers only newly observed nearby beech logs, excluding
the initial resource IDs. Proximity is not proof of spawn lineage. At most five
harvest tasks and eight minutes of work are allowed, within 40 m of the initial
position. Each successful task needs matching local-hit/break evidence, and a
positive benchmark needs at least two log harvests, a positive observed Wood
inventory delta and no observed unfinished nearby new logs. Unsafe/unavailable
logs, lost ownership, changed world/connection, health loss and stale inventory
abort instead of requesting owner clearance. Existing native route/threat guards
remain active. It always logs out/stops and restores combat off.

Evidence under `.checks/harvest-trial-*` retains exact resource IDs, per-task
histories, before/after item counts, observations and audit paths. The console
prints a bounded outcome with task/log counts and Wood gain. Use `just check vm
test-harvest-trial` for selection, scoped inventory, native-goal sequencing,
evidence requirements, failure and cleanup checks; it is part of `just check ci`.
This does not establish mining, crafting, delivery home or indefinite harvesting.

`just check vm gather-trial --wait-s 60` tests the controller's bounded `gather`
behavior through full-tree/log harvesting, gains-only home-chest delivery and
return to the work site. It uses one uniquely named manual, once-only instance;
the supervisor monitors scope, health, nearby hostiles and travel without issuing
movement or harvesting tasks itself. It requires the same supplied axe/food and
isolated beech as the native benchmark, selecting from current perception within
60 m because the controller can approach its site before native acquisition.
It also requires remembered storage within
10 m of home and home within 150 m. Remembered chest IDs are hints; only a unique
visible chest within 0.4 m can replace an old ID for this trial.

Success requires instance-attributed native steps, distinct tree/log identities,
task-matched local-hit breaks, measured inventory gains, exact deposit receipts
and a successful return task with a matching final pose. Original products must
remain in the pack. Missing audit proof fails the run. Cleanup stops/deletes the
temporary behavior, logs out/stops the game, restores combat off and restarts
the offline panel, including after partial request failures. Evidence stays in
`.checks/gather-trial-*`; the console reports compact task/log/gain/deposit/return
results. `just check vm test-gather-trial` checks proof reconciliation;
`just check vm test-combat-trial` also checks supervisor dispatch and cleanup.
Both are included in `just check ci`. This single cycle does not establish chat
site inference, repeated hauling or the complete M2 milestone.

Trials also retain a bounded harvest projection in `summary.json`: matching
contact, safety-scan, escape and break evidence, with damage-event count for the
whole audit window. Re-read an existing audit with `just vm harvest-evidence
AUDIT TASK_ID [TASK_ID ...]` (up to five IDs), without fetching broad history or
starting the game. Repeated messages retain counts and their latest details.

To retry a fallen trunk without felling another tree, use `just vm harvest-logs
X Y Z --wait-s 60` with the known log area's center. This explicitly considers
existing nearby tier-zero beech logs, uses the same safety/cleanup bounds, and
reports `mode: logs_only`. A pass requires a log break, collected Wood and no
observed unfinished nearby logs; it does not claim a full-tree benchmark.

Physical reach uses actual collider contact and strike sight from axe height;
a bounds center is never contact evidence. Logs probe at Tas's strike height
with a 3D collider-ray fallback. Support projection uses the lower of the pivot
and measured contact. High or unsupported contacts still refuse.
For supported box/sphere/capsule/convex-mesh log colliders, exact nearest-surface
queries precede the ray fallback; nonconvex meshes retain physical rays.
Complete protection/settling queries reuse a buffer that grows from 256 to 2048, then
refuse rather than consume truncated results. Before swings, up to seven
outward six-metre escape corridors are checked, one per physics tick, with
supported, dry, wall/body-clear steps every half metre.

Use `just check plugin test-core HarvestTests` for contact/reach, lower log
support, escape geometry, health and tool policy. Use `just check plugin
test-core ScanBufferTests` for complete-query retry, storage reuse and the cap.

Verified live slices include full-beech felling, falling escape, stump breaking,
and a three-task trunk/halves run with 20 Wood gained and no damaged events or
cleanup errors (audit `.checks/audit-window-4KNHu1.jsonl`). A subsequent full-tree
run felled another beech but refused its elevated log. A continuous full-tree
benchmark, collection-disabled and failure branches, mining and controller-level
gathering/delivery remain separate checks. Current results and exact evidence
are on the [harvest ticket](../.tickets/tickets/TKT-01M62N12F04QEC0F53JCS1M21A.md).
See [BEHAVIOURS.md](BEHAVIOURS.md) for the bounded `gather` template.

### Explicit melee

Use `just check plugin test-core MeleeTests` for encounter policy, stamina
reserves, target eligibility and the attack task's runner cancellation paths.
Use `just check controller test-packages ./internal/retreat ./internal/protocol
./internal/app` for attack/retreat arbitration and protocol checks. Shared
fixtures include task_attack and state_combat. These do not prove real strike
contact, animation timing, block effectiveness or roll displacement.

To isolate native closing/melee from lure preparation, use
`just vm combat-trial --direct --entity ID --wait-s 30`. This requires an explicit
observed Greyling ID and preserves the same health, threat, ownership and cleanup
supervision. It submits one native attack with `kite:false`; success proves the
direct encounter, not pursuit or kiting. It cannot combine with preferred ground
or harvest options.

`just vm combat-trial` performs one automated live attempt with normal controls
and no owner staging/clearance prerequisite. It restarts Tas, temporarily enables
combat, delegates local ground/target selection and kiting to C#, requires active
food/80% health/40 stamina, and records evidence under `.checks/combat-trial-*`.
It always logs out/stops/restores combat off, then restarts only the
controller so the web panel stays available while Tas is offline. Additional creatures, unsafe
health, exhausted escape, lost task ownership and more than 40 m travel abort
the trial. No eligible encounter is an aborted trial, never a pass. Optional
`--entity` restricts native target selection, `--at X Y Z` names preferred nearby
fighting ground, and `--wait-s` bounds food/stamina preparation to 0..120 s.
It does not spawn/reset targets or scout beyond its bounded local area. Native
preparation moves to checked ground and can make short checked out-and-back
scouting excursions while acquiring. Use `just check plugin test-core ScoutingTests`
for finite route selection, return, stamina/time/travel/progress bounds and
route changes. Outbound route changes, stuck progress and Walker refusals stop
that leg and advance to another bearing, with at most four recoveries and the
same candidate/time/travel bounds. At the limit Tas returns instead of starting
another excursion. A bounded trail records actual outbound positions; current
path/support/grade/water/character checks must approve every return connector.
The planner reserves travel for that winding trail rather than just distance
from the origin. A refused return or unsafe body state still exits immediately.
Return legs do not count as outbound search coverage. Compact trial reports
retain recovery-reason counts and the latest trail length/point count; raw
traces remain on disk. No eligible enemy after returning remains a failed trial. Use
`just check plugin test-core KitingTests` for pursuit/pacing/bounds and
`just check vm test-combat-trial` for single-goal dispatch, interruption and cleanup;
these are included in `just check ci`. A repeatable isolated test arena remains
future work; positive contact/block/dodge checks still require actual observations.

The first deployed native trial on 2026-10-30 UTC (684504b) accepted one
`attack {"kite":true}` goal and failed `unreachable` after an 8.6 s local survey:
no supported clear fighting ground with a checked escape corridor was found
near Tas's hillside position. No enemy was acquired or damage observed;
minimum HP was 76.1. Automatic logout, shutdown and combat-off reset completed
without cleanup errors. This verifies the bounded refusal, not successful
kiting or melee. Evidence: `.checks/combat-trial-20261030T021553504234Z/summary.json`
and `.checks/audit-window-qI99Fo.jsonl`. This exposed an unintended flat-ground
requirement: the full corridor inherited a single-step height limit. The
corrected validator follows support every 0.5 m, permits up to 30 degrees of
inclination, and rejects adjacent inclination changes over 15 degrees when
their height residual exceeds 0.15 m. The residual is the next support height's
deviation from the previous segment's continuation. This distinguishes mild
mesh joins observed in the game from physically sharp ledges and ridges.
Fighting-ground stance diameters and dodge corridors share this policy, with
candidate rejection traces for live diagnosis. Use `just check plugin test-core
GroundCorridorTests` for gradual uphill/downhill terrain, changing grade,
sharp crests/dips/ledges and unsupported/blocked intermediate ground. Positive
pursuit/engagement checks still require live evidence.

Ground selection checks four complete diameters through a 3 m stance region
and requires at least one clear 6 m escape corridor among eight directions.
It does not require every outer direction to be open. Use `just check plugin
test-core FightingGroundTests` for outer obstacles with alternative exits,
blocked escape rings, central steep/sharp/unsupported terrain, gradual slopes,
missing dry projections and obstruction changes. These policy checks do not
prove real dodging or retreat; actual moves retain their own clearance checks.

The deployed directional-ground build (7d61e9b) accepted its first candidate
on the same hillside on 2026-10-30 UTC: center [174.1,84.69,-246.99], checked
escape endpoint [180.1,83.91,-246.99]. Native preparation reached search and
ended `invalid_target` because no eligible single Greyling was available;
the last perception contained only a Boar. Minimum HP 74.3 matched the
food-derived maximum, without observed damage. This verifies ground selection
and stage progress, not pursuit or melee. Automatic logout/shutdown restored
combat off, independently confirmed by status/gate checks. Evidence:
`.checks/combat-trial-20261030T045020394084Z/summary.json` and
`.checks/audit-window-ysrr4a.jsonl`. The next acquisition improvement is bounded
native scouting; repeatedly searching from the same spot cannot establish a
positive encounter when an eligible target is absent.

Each trial's `summary.json` also projects matching-task ground-rejection counts,
physical probe stages and at most six representative failures from its audit.
It also retains the selected area/escape, distinct native modes, outbound/return
scouting counts and bounded scouting-refusal examples, scoped to that task.
The console prints only counts and the terminal task reason; exact segment
positions, inclinations and collider evidence stay in the retained summary and
audit. Failures in this optional diagnostic projection are recorded separately
and do not replace the encounter result or interfere with cleanup.
Use `just vm combat-evidence AUDIT TASK_ID` to print the same bounded projection
from a retained audit without starting the game. Relative audit paths resolve
from the repository root.

The corrected build (33cd427) also returned `unreachable` from the same hillside
on 2026-10-30 UTC. Its 34 candidate traces reported 16 grade refusals, 8 sharp
changes and 10 support/clearance failures. No enemy was acquired; no damage was
observed, and automatic shutdown restored combat off without cleanup errors.
This is a failed encounter attempt, not positive kiting evidence or proof that
every rejected point is physically unusable. Summary:
`.checks/combat-trial-20261030T022800895969Z/summary.json`; candidate traces:
`.checks/audit-window-df13eN.jsonl`.

Read the plugin gate with `just vm combat`. First deploy with it off, allow
attack in the controller's allowedTasks and verify a command is rejected by
the plugin. For an owner-staged live check, use `just vm combat true` before
restarting the game, provide food and a usable melee weapon, and keep one
ordinary unstarred Greyling in clear open ground with the owner ready to
clear it. Use `just vm task-check attack '{"entity":"<live-greyling-id>",
"style":"melee"}' 90` and `just vm audit-summary` to retain exact evidence.
The first slice refuses players/tamed creatures/other species and keeps
combat off by default. Verify normal aim/contact, block and supported dodge,
bounded escape on deteriorating conditions, stop/reconnect cleanup and a
separate return to shelter. Do not deliberately damage Tas for a threshold.
Keep unobserved cases open on the attack and retreat tickets.

On 2026-10-30 UTC, dd3edda passed the deployed permission-gate smoke:
controller allowedTasks included attack, but the plugin with AllowCombat=false
rejected the request as not_allowed. Tas stayed idle and sheltered. Browser
preview checks passed for stamina, block mode, retreat reasons and clearing
finished combat status. No positive combat outcome is claimed by these checks.

The subsequent owner-staged attempt never entered melee: ordinary retreat took
over after confirmed Greyling hits, and t-10 attack arrived outside the original
12 m admission bound. Repeated escapes exhausted stamina and led into a Boar.
Minimum observed health was 68.1; final observed health was 76.1/76.7. The game
was stopped and combat config reset off. This motivated explicit bounded closing
and the automated trial backstops above; it proves no swing/block/dodge outcome.


## Persistent combat objective

`just check vm resupply-food` prepares Tas through normal native controls with
combat off: return to the known home within 500 m, try up to four ordinary-food
withdrawals from remembered nearby storage, request at most four each of two
food types, and observe automatic eating. Every claimed transfer requires a
task-matched chest receipt and newer scoped inventory; memory only selects
candidates. Damage, unsafe water, loss of ownership, world/connection changes
and time/travel limits abort. Evidence and cleanup use the shared trial harness,
which always logs out/stops and leaves the panel available. It is an explicit
operator preparation check, not a background supply policy.
Long return trips walk; a final approach within 40 m may run with at least
30 stamina. Damage and survival preemption still abort preparation.
If rejoin has already started the configured outage return, preparation may
wait up to 120 s for that exact native task to finish. Its outgoing request must
name the known same-world home; another order, changed scope or damage aborts.
Preparation never replaces the active task to get control of the body.
A remembered food chest whose ID is no longer visible can suggest a unique
currently perceived chest within 0.4 m in three dimensions. Ambiguous matches
are skipped; old contents never prove current stock. The transfer still needs
the exact current chest/task receipt and newer inventory evidence.
If a hinted food is missing, the returned chest snapshot overrides that hint.
Within the same four-request budget, preparation can check other visible home
chests through a one-item ordinary-food request. Missing food still returns the
actual chest contents, allowing the next request to use that observation.
Only verified moved food counts as resupply; a missing item is a search outcome.

During a trial, `just vm trial-status` reads the newest local evidence directory
and prints only position, observed travel, health, food, task and observation
age. Pass an evidence directory to inspect a particular run. It makes no VM
requests, and reports success only from the completed supervisor summary.
Full observations, task histories and audit frames remain on disk; inspect
specific diagnostics only when this compact progress indicates a problem.
Final summaries also derive minimum/final HP and damage-event counts from the
native audit, including frames between supervisor polls and during cleanup.
An older run's sampled `minimumHP` alone cannot establish that no damage occurred.

Task histories now include the chest receipts naming that task, so the panel's
task view and preparation supervisor can inspect exact moved/remaining counts
without reading a complete audit window.

`just check vm goal-trial` submits one durable `combat_trial` goal and follows it
across failed native tasks. It retains compact progress, the final goal record
and full observations in `.checks/`. A changed target or a close confirmed
attacker can produce another native attempt, while unchanged failures exhaust a
bounded recovery wait and request a new plan. A goal success requires an exact
native “observed Greyling death” receipt; orchestration progress alone does not
close melee/block/dodge checks. Cleanup cancels unfinished goals, logs out and
stops the game, restores combat off and starts the controller/panel alone.
See [GOALS.md](GOALS.md) for restart uncertainty and policy boundaries.

## Game-camera debugging

Enable Game camera in the panel, then use `just vm camera` to save a JPEG and
matching metadata under `.checks/`. Inspect the image for a concrete navigation,
aiming or pose hypothesis; keep structured task receipts as completion proof.
Check default five-second delivery, then the one-second minimum interval and
capture cost on the VM. Zero/subsecond intervals must be refused. Check settings
changes, capture off, disconnect/menu/world replacement and unavailable images.
The panel should clear old scenes, and a pinned superseded image URL should
return409. Ordinary audit/event responses must contain metadata without image
bytes. Verify capture with the game unfocused and record whether minimized
rendering supplies new frames before relying on it in unattended diagnostics.


Use `just vm camera-diagnostic 1 15` for a bounded fast-capture check, followed
by `just vm camera` for JPEG evidence and `just vm camera-config` for scope and
expiry. Keep the game/connection online throughout: observe normal five-second
frames, one-second frames during the override, then five-second frames after
automatic restoration. Confirm persisted settings do not change. Test early
release, a stale release ID, conflicting pins, and an explicit normal settings
change displacing the override. Disconnect/world replacement must clear the
lease without configuring a replacement connection. Focused regression coverage
runs through `just check controller test-packages ./internal/app ./internal/session ./internal/panel`.


`just vm camera-check` runs the bounded live 5 -> temporary1 -> automatic5
cycle, captures matching JPEG/map evidence, checks connection/world and health,
and retains a compact result plus the full audit under `.checks/`. It requires
an in-world bot with normal five-second capture enabled; it always logs out,
stops the game and leaves the controller/panel online. If the saved normal rate
was different, record it before `just vm camera-rate 5` and restore it after the
trial. The diagnostic itself never changes saved settings.

## Deterministic Preact browser checks

See [PANEL-UI-TESTING.md](PANEL-UI-TESTING.md) for `just panel ui-install`,
`just panel fixture`, `just check panel ui-test`, `just check panel examples`
and compact artifact inspection. The actual built panel/shared renderer runs
against recorded fixtures with a locked browser/font and no game requests.
The browser suite is required by `just check ci`; missing prerequisites fail.
Screenshots and structured diagnostics live in `.checks/`, with a separate
visual-review checklist that never treats a passing assertion as design approval.

