# Testing renki in the game

Everything that can run without Valheim is covered by `just ci`. That covers
the controller, the protocol on both sides, the plugin's task runner, safety
rules and movement maths, and the C# bridge client talking to the real Go
controller. This page covers what only the game can show. Each check names the
ticket it closes. Tickets in `review` are waiting on these checks, and
`git ticket list --status review` lists them.

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
| 2 | Start the controller with the game at the menu | Controller logs `plugin connected` and `session started`; plugin logs `session started as Bjorn` | TKT-01M61T36CC, criterion 1 |
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
  command it from afar with `/s` (shout).
- Under Unity's Mono, the WebSocket client is the one unknown. If check 2 or 3
  fails, the plugin log will say so, and websocket-sharp is the fallback
  (TKT-01M61T36CC).

When a check fails, note it on the ticket it names, with the log lines.
