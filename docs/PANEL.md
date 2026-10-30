# The control panel

A web page, served by the controller, for watching the bot live: where it
is, what it is doing, and why, with buttons to stop it, send it home or
log it out. TKT-01M628X106 (Web control panel to watch and steer the bot)
tracks the remaining page controls for behaviour instances. The engine
already exposes authenticated controls described in [BEHAVIOURS.md](BEHAVIOURS.md).

## Turning it on

On the VM, once:

    just vm panel-setup     # Tailscale address, firewall rule, a random token
    just vm restart
    just vm panel-token     # the address and the token, printed here only
    just vm panel-address   # just the address, without printing the token

Then open the address from any device on the tailnet and log in with the
token. The login lasts 30 days on that browser; **Log out** ends it.

`panel-setup` (scripts/vm/setup-panel.ps1) finds the VM's Tailscale address,
adds the firewall rule `RenkiPanel` (controller.exe, TCP 8780, from
100.64.0.0/10 only), and writes `panel` into `controller.local.json`:

```json
"panel": { "listen": "100.101.221.77:8780", "token": "…" }
```

`RENKI_PANEL_TOKEN` overrides the token. The controller refuses a panel
without a token of at least 16 characters. If it cannot listen, as when
Tailscale is not up yet at logon, it logs that and tries again every 30 s;
the bot runs either way.

On the dev box, add a `panel` block with `"listen": "127.0.0.1:8780"` to
`controller.local.json` and run `just controller run`.

## What it shows

- **Header:** connection, character and world, phase, the running task,
  health, a warning when the bot carries more than it can (and whether it
  can still move), and how far home is.
- **Buttons**, while the plugin is connected: **Stop** ends whatever the bot
  is doing, as "stop" in chat does. **Go home** sends it home, as "go home"
  does, and says why not when it can't (no home in this world, too heavy
  to move, `move_to` not allowed). **Log the bot out** asks first, then logs
  the character out to the menu; logging back in needs the VM
  (`just vm start`). The answer shows beside the buttons, and each press is
  in the feed as a `panel action` line with where it came from. The page's
  own **Log out** only ends the login.
- **Map:** top-down, north up, a grid every 10 m, opening zoomed to the
  perception radius. The bot with its facing and its trail over the last
  few minutes; from perception, players, creatures (red when hostile,
  ringed when after the bot), structures and resources, each kind with its
  own shape and colour (see the legend; doors hollow when open); walls
  as grey lines under everything, from the plugin's layout (faint for
  upper storeys, so doorways show); home, a
  diamond for each place the bot had to leave things because it carried too
  much, the walker's latest path (dashed when it had no navmesh path and
  went straight), and a circle for each stuck recovery. Things of one kind
  close together on screen draw as one marker with a count ("6 chests").
  Labels go on in order of importance and are left off where they would
  overlap; single trees, rocks and the like have none. Hover any marker for
  what it is, its distance from the bot and its details, every item of a
  group. Scroll or use + and − to zoom; drag to pan, which stops it
  following the bot.
- **Terrain and location:** biome colors, water shading with interpolated
  shorelines, and 5 m height contours from loaded terrain sampled every 4 m.
  Unsampled areas stay dark/unknown. The header names the biome, supporting
  surface and shelter; character and terrain heights are separate. Water
  above feet appears when immersed; hovering terrain reports seabed depth
  instead. Floors, bridges, waves, caves and obstacles need their own live
  checks: this backdrop does not establish walkability.
  Terrain, Elevation, Remembered and Resources buttons control layers;
  resources start hidden to keep the view readable. Faint remembered markers
  show last-seen time on hover and never duplicate an entity in current
  perception. The north arrow, scale bar, facing marker and perception range
  ring help with orientation. The range ring is not a guarantee that every
  object inside it was reported.
  Authenticated `GET /api/map` supplies current-world/connection terrain and
  up to 512 nearest remembered landmarks; the page refreshes it every 5 s.
  Terrain is bounded to 256 cached tiles, resets on controller/plugin
  reconnect or world/menu changes, and repopulates from the plugin. Persistent
  exploration is separate work. [ATLAS.md](ATLAS.md) proposes durable terrain,
  saved-world viewing and age/provenance styling; these are not implemented yet.
  A map request failure retains the last
  scoped view until a later successful refresh.
- **Voice:** the model the bot's chat replies use, who chose it and when,
  and a list of what the backend offers (its `/models`, for an
  OpenAI-compatible backend such as cpa-api). **Use** switches to the one
  picked, recorded as the panel's change in the settings store; the next
  reply uses it, with no restart. Each reply's audit line (`voice`) names
  the model, its latency in `ms` and its `tokensIn` and `tokensOut`, so
  models can be compared from the log.
- **Settings:** every runtime setting (docs/BRIDGE-PROTOCOL.md, Runtime
  settings), with its value, whether it is the config's or a change and
  who made it, and a pin. A change applies as soon as the box loses focus
  or the switch flips, recorded as the panel's; **Reset** returns a setting
  to the config's value, and **Pin** keeps the agent from changing it.
  Lists are comma-separated. Changes from chat or another page show while
  the section is open.
- **Inventory:** what the bot carries, laid out as in the game, with stacks,
  a wear bar on items that wear (red below a quarter), equipped items
  outlined, and free slots and weight above. Hover a slot for the detail.
- **Say:** words typed above the feed reach the bot as if the owner
  picked beside them had said them in the game, with no need for its
  name. It answers in the feed (`said to the panel`), not in the game.
  "Come here", "set home", "pick that up" and "follow me" need that
  owner's character in the bot's sight, and are refused when it is not.
- **Feed:** the audit log's records as they are written (docs/AUDIT.md), in
  the same lines `just vm audit` prints. Filters: chat, tasks, the plugin's
  decisions (`trace`), the controller's decisions, other frames, and state
  (once a second). **Pause** stops it scrolling. The day picker beside it
  reads a past day's audit file instead (its newest 2000 records, with
  **Load earlier** for more; state and perception left out), and **Live**
  goes back.
  Routine perception scans and navigation/terrain cache statistics are under
  the unchecked Diagnostics filter; navigation decisions remain visible.
- **Task view:** any line about a task starts with its id; click it for
  everything about that task in the controller run it belongs to (ids
  start again with each run): the chat that led to it, the request, its
  statuses, the plugin's traces and the controller's records, with how it
  ended and how long it took. The same cut as `just vm audit -task`.

A new page starts with the last 500 records, the newest state, perception
and inventory, and a state a second for the last five minutes for the trail.

## How it works

`internal/panel` serves three embedded files and these endpoints, all
behind the login: `GET /api/status`, a snapshot of the bot,
`GET /api/inventory`, a read-only snapshot of its pack, carrying weight and
world/connection scope (inventory is null before observation or outside a live
world),
`GET /api/events`, server-sent events carrying each audit record (`record`,
with the timeline's `line` and a category `cat`) and a status every second
(`status`), and `POST /api/act/{stop,home,logout}` for the buttons, which
answers `{"text": …}` with 200, 404 for an unknown action or 409 when it
could not be done. `GET /api/settings` lists the runtime settings with who
set each, `POST /api/setting` with `{"key": …, "value": …}` changes one as
the panel (400 bad value, 404 unknown key, 409 pinned or owners-only), and
`GET /api/models` lists the voice's models. `POST /api/say` with
`{"as": owner, "text": …}` speaks to the bot as that owner. `GET /api/days` lists the
audit files, `GET /api/timeline?day=…[&before=…]` reads one, and
`GET /api/task?id=…&at=…` gives a task's records from the run going at
`at`. `POST /api/task` accepts an exact `{ "task": "move_to", "args":
{ "to": [1, 2, 3] }, "timeoutS": 60 }` and returns 202 with its session
task ID. Invalid arguments return 400; connection/capability refusals return
409. It uses normal direct-order semantics and all controller and plugin
safety checks. `timeoutS` is 0 (unbounded) through 600; IDs are assigned by
the session. `just vm task <task> '<args-json>' [timeout]` sends one without
exposing panel credentials. A press or a change must carry
an `X-Renki-Act` header: a page on
another site cannot add one without the browser asking this server first,
which it never allows, so a forged form cannot press the buttons. The page has no build step. Scripts can use the token as a
bearer token: `curl -H "Authorization: Bearer $TOKEN" http://…/api/status`.

Chest markers show last-known contents on hover, using the same per-world
container memory as Tas. Quantities and the last-opened time describe that
observation; another player may since have changed the chest. An unopened
chest shows “Contents unknown”, while an observed empty chest is labelled
“Empty when last opened”. Groups show a bounded preview; zoom in to inspect
individual chests. Seeing a chest again does not refresh its contents.

The map's **Places & zones** editor manages persistent named points and
circular areas. Tas marks evidenced storage/cooking/harvesting purposes;
owners can add, rename, move, resize and delete them. Your changes take
precedence over automatic discovery. **What's nearby?** shares Tas's
last-known object and chest memory. See [Places and zones](WAYPOINTS.md) for
height handling, chat commands, named movement and the API.


## Persistent goals

The Goals section keeps an objective visible across individual routine failures.
“Test one Greyling encounter nearby” starts a bounded goal at Tas’s current
position using existing combat permissions. Progress includes attempts, recovery
reasons, recent task outcomes and native success evidence. Stop goal cancels its
continuation; the main Stop button also cancels queued goals. Suspended goals
require explicit Resume with fresh observations in the original world. Uncertain
actions cannot be replayed. See [GOALS.md](GOALS.md) for budgets, policy limits
and the authenticated `/api/goals` contract.

## Players and presence

**Players & presence** controls Tas's shared map marker and Marco/Polo routine,
and assigns operator, companion or guest profiles. Observed marker state comes
from the game; offline changes apply on the next join. Profiles separate chat
command authority from locating/greeting permissions. See [PLAYERS.md](PLAYERS.md)
for the permission matrix and the two opposite direction exchanges.
