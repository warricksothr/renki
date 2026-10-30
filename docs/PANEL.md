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

- **Overview and safety bar:** connection, character and world, phase, the running task, active meals,
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
  The hot cache holds 256 tiles; the durable atlas accumulates measured terrain
  across reconnects and restarts under stable world/epoch identity. The **Map**
  selector defaults to **Follow Tas · last map offline**: it keeps his last
  saved terrain under a fog overlay after logout and opens the most recent
  saved world on a fresh offline page. Rejoining restores live mode. Choosing
  a specific saved world pins it until Follow Tas is selected again. Stale
  cells fade further; hover
  shows measurement age, source and spacing. Saved viewing clears live actors
  and objects. Viewport requests are bounded to 512 m per side; wider views show
  a central detail region. [ATLAS.md](ATLAS.md) documents retention, quotas and
  the read-only `/api/atlas/worlds` and `/api/atlas/map` endpoints. Older plugins
  without trustworthy identity/freshness remain session-only. Exploration is
  separate work.
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
- **Activity:** a bounded timeline beside the camera, with tasks and chat visible
  by default. Task outcomes show readable reasons; each record has expandable
  **Technical details** containing the original audit line (`just vm audit`).
  Warnings and errors stay visible even when their category is unchecked. Under **Filters & history**: chat, tasks, the plugin's
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

`internal/panel` serves the built, embedded Preact workspace and shared terrain renderer and these endpoints, all
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
which it never allows, so a forged form cannot press the buttons. The frontend build is described below. Scripts can use the token as a
bearer token: `curl -H "Authorization: Bearer $TOKEN" http://…/api/status`.

Chest markers show last-known contents on hover, using the same per-world
container memory as Tas. Quantities and the last-opened time describe that
observation; another player may since have changed the chest. An unopened
chest shows “Contents unknown”, while an observed empty chest is labelled
“Empty when last opened”. Groups show a bounded preview; zoom in to inspect
individual chests. Seeing a chest again does not refresh its contents.
Matching newer observations reconcile old network IDs in the shared memory,
including beds, doors and stations. Contents inherited from a previous chest
at that place are labelled “needs checking” until the current chest is opened.

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

## Game camera

The Game camera panel shows a latest-only JPEG of the rendered game view,
including the HUD, with capture time, age, dimensions, capture cost and task.
Enable capture there; the default interval is five seconds. `screenshot.intervalS`
accepts 1–300 seconds, with a native hard ceiling of one capture start per second.
`screenshot.width` accepts 320–1280 (default 960); height preserves the screen
aspect ratio up to 720. JPEG quality accepts 30–90 (default 70). These values
also appear in Settings. Capture starts disabled in a new configuration.

Authenticated `GET /api/camera` returns availability, config, scoped metadata,
receipt age and a pinned image URL. `GET /api/camera.jpg` returns the current
JPEG; the pinned URL refuses superseded frames with 409. Refresh metadata and
retry rather than attach a new scene to old metadata. Images are cleared on
connection/world/menu changes and disabled capture, and are kept only in memory.
Offline capture is unavailable; a retained file is historical evidence.

Runtime camera tools work while the same game connection stays online:

- `just vm camera-config` reports effective `config`, normal `baseConfig`, and
  an active `override` (ID, expiry, world UID and connection) or `null`.
- `just vm camera-rate 5` changes the normal interval using agent permissions.
  The interval is **seconds per capture**: 1 is fastest, 5 default, 300 slowest.
- `just vm camera-diagnostic 1 30` enables one-second capture for 30 seconds.
  The controller automatically restores normal settings on expiry. Both the
  interval and duration are bounded to 1–300 seconds. No diagnostic setting is
  persisted; reconnect/restart receives normal settings.
- `just vm camera-config '{"action":"release","id":"<override-id>"}'` ends that
  diagnostic early. A stale ID cannot cancel a newer diagnostic.
- `just vm camera-config '{"action":"temporary","enabled":false,"durationS":30}'`
  temporarily disables capture to diagnose overhead.

Authenticated `POST /api/camera` requires `X-Renki-Act`, like other panel
mutations. Besides `temporary`/`release`, it accepts
`{"action":"set","key":"intervalS","value":5}` or
`{"action":"reset","key":"intervalS"}`; keys are `intervalS`, `enabled`,
`width` and `quality`. Agent changes honor pinned settings. Temporary diagnostics
require an in-world connection with a stable world UID, refuse overlapping
leases, and end on world/connection changes or a conflicting camera pin (checked
at most one second later). Any successful normal runtime setting update,
including a same-value or unrelated setting update, displaces the diagnostic;
cleanup restores the latest normal settings rather than overwriting a user's
new choice. Restores target the original connection so they cannot leak to its
replacement. A successful API response means the config was sent; subsequent
frame metadata proves the plugin's actual capture rate.

Use `just vm camera` to save the latest JPEG and matching metadata in `.checks/`
for vision debugging. Image bytes stay outside ordinary audit/event streams and
model context; the audit retains capture metadata only. Tools can inspect a
saved image when a concrete failure needs visual reasoning. A screenshot is
visual context, not proof of a task's success. Current capture uses the game's
rendered camera, which can be third person; it does not change the bot's view.
The native screen framebuffer is capped at 3840×2160, JPEGs at 512 KiB, and only
one capture can be in flight. Images use a separate latest-only low-priority
bridge slot; control messages take priority over pending images. GPU readback
and JPEG encoding costs must be measured on the deployment.

**Save map snapshot** exports the displayed map as a JPEG and a matching JSON
bundle with rendered-image bytes, map data, world/connection scope, viewport,
layers and load/capture times. Agent browser tools call
`window.renkiSnapshots.map({maxWidth:640,quality:0.7})` to obtain the same
image/metadata/data bundle without clicking or introducing another renderer.
Retrieve only the image and compact metadata when reasoning visually; the full
map remains available separately. Saved/offline atlases are labelled as such.

`just vm map-snapshot` captures matching live scope, `/api/map` and saved terrain
into `.checks/map-snapshot-*/`, then embeds the actual panel assets in a
self-contained read-only `map.html`. A native HTML preview/screenshot tool can
render it without a logged-in browser or an open panel. `just vm map-snapshot
--html` emits that document directly for a rendering tool; do not print its full
source into model context. This uses the default panel viewport; an exact view
with the user's pan/zoom/layers comes from the panel export/browser function.
The capture is sequential and records its scope checks; it is not an atomic
photograph of a changing world. It cannot infer physical visibility or success.


## Workspace and frontend development

The Preact workspace source is in `panel/src`; `controller.jsx` adapts the
established control APIs and contains the single canvas map renderer. Preact
mounts its stable control containers once, then the adapter owns their contents.
Camera, food/current-goal overview and activity records use Preact components.
The map, its hover content, terrain layers and `window.renkiSnapshots.map`
share that renderer; standalone previews embed the same built assets.

The page scrolls as a document. The command field sits beside current status,
with a direct link from current work to goal controls. Map and camera share a
balanced desktop row; the map grows into available height, while the camera
preserves the whole image. Activity and Kit & objectives follow in two columns.
Places & storage and People & preferences form a separate reference row.
Inventory and the place editor start collapsed; their summaries retain counts
and pack status. Goals stay expanded. Activity, long goals, settings and place
editors have bounded independent scrolling; expanding one does not clip other
controls. Section links follow the reading order. Below 1101px the map and camera
stack; below 761px the control cards stack too. Stop, Go
home and Log bot out remain in the safety bar (a fixed bottom dock on mobile); **Sign out** ends only the panel
session. Logging the bot out still asks for confirmation.

Game camera stays prominent without opening a disclosure. Capture defaults
remain off, five seconds, with a one-second minimum. Enable capture with its
checkbox; edit **Base interval … seconds**, then **Apply**. Temporary capture overrides
show the effective interval, base interval and expiry beside these controls. Width, quality, pins and
reset remain in Settings. The panel shows loading, disabled, pending, offline,
failed/decode-error and stale states. It reports both receipt age and capture
age, and dims images older than three capture intervals (at least 15 seconds).
A disconnect, world/connection replacement, disabled setting or failed request
clears the scene. A blob URL is shown only after a matching scoped image
request, and the image's load/decode result is checked. The image policy permits
`blob:` alongside same-origin/data images, as introduced in the camera CSP fix.

Use Node.js 20+ and npm with the existing Go/.NET tools. `just panel build`
installs exact lockfile versions with `npm ci` and bundles Preact locally;
there are no CDN scripts or runtime third-party requests. Generated JS/CSS are
committed under `internal/panel/static` for Go embedding. `just panel verify`
checks that those bytes match the source before controller recipes can build.
`just controller build`, `just controller build-windows`, `just controller run`,
`just controller preview-map` and `just package` rebuild the frontend before
embedding/previewing it. Root build/test/lint/fmt/ci/clean route to the panel
component first. `just check panel test` exercises camera scope/races and
presentation; `just check ci` includes reproducibility and the full gate.
Edit sources in `panel/src`, then use `just check panel fmt` and
`just check panel build`; do not edit generated assets.

`just controller preview-map --serve` is a self-contained read-only fixture.
Pass `--camera <recorded.jpg>` to exercise real JPEG decoding (metadata timing
is replayed), `--camera-state disabled|pending|failed|broken|stale|image` for
camera states, or `--offline` for saved-map fallback. These previews identify
fixtures visibly; recorded images do not establish a live outcome. Mutations
are refused inside the document, with no credentials or network fallback.
From a native browser console, `await renkiPreview.check()` checks controls,
all layers/exports, simulated keyboard pan/zoom, task/history drilldown,
scroll/focus reachability, image decode and camera transitions, and offline
fog/rejoin. Supply `--camera` to include the full image checks. Existing
`--status`, `--map`, `--atlas` and `--map-only` work with recorded map bundles.
`just vm map-snapshot` still calls this same preview path.

Keyboard: Tab to links/buttons/disclosure summaries; Enter/Space activate them.
The map itself accepts arrows to pan, +/− to zoom and C to follow the bot.
Inventory items have keyboard focus and accessible item descriptions. The
native task dialog supports Escape and returns focus to its task link.
