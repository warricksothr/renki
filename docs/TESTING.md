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
relevant function, and query audit logs for one task with
`just vm audit 1 -task t-3`. Keep durable decisions and remaining game checks
on the ticket so a resumed session can read those instead of old transcripts.
Rerun checks after changes or new evidence, rather than repeating passing
suites while waiting for the game. The complete gate is still required.

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
