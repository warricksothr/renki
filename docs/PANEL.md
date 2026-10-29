# The control panel

A web page, served by the controller, for watching the bot live: where it
is, what it is doing, and why. TKT-01M628X106 (Web control panel to watch
and steer the bot) tracks what is still to come: buttons, behaviour
switches and a per-task view.

## Turning it on

On the VM, once:

    just vm panel-setup     # Tailscale address, firewall rule, a random token
    just vm restart
    just vm panel-token     # the address and the token, printed here only

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
  health, and how far home is.
- **Map:** top-down, north up, a grid every 10 m. The bot with its facing and
  its trail over the last few minutes; from perception, players, creatures
  (red when hostile, ringed when after the bot), structures (doors hollow
  when open) and, zoomed in, trees, rocks, pickables and drops; home, the
  walker's latest path (dashed when it had no navmesh
  path and went straight), and a circle for each stuck recovery. Scroll or
  use + and − to zoom; drag to pan, which stops it following the bot.
- **Inventory:** what the bot carries, laid out as in the game, with stacks,
  a wear bar on items that wear (red below a quarter), equipped items
  outlined, and free slots and weight above. Hover a slot for the detail.
- **Feed:** the audit log's records as they are written (docs/AUDIT.md), in
  the same lines `just vm audit` prints. Filters: chat, tasks, the plugin's
  decisions (`trace`), the controller's decisions, other frames, and state
  (once a second). **Pause** stops it scrolling.

A new page starts with the last 500 records, the newest state, perception
and inventory, and a state a second for the last five minutes for the trail.

## How it works

`internal/panel` serves three embedded files and two endpoints, all behind
the login: `GET /api/status`, a snapshot of the bot, and `GET /api/events`,
server-sent events carrying each audit record (`record`, with the
timeline's `line` and a category `cat`) and a status every second
(`status`). The page has no build step. Scripts can use the token as a
bearer token: `curl -H "Authorization: Bearer $TOKEN" http://…/api/status`.
