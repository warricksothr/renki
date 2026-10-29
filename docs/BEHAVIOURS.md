# Behaviour engine

Named definitions in `settings.json` now run through the controller's
behaviour engine. Each definition selects a Go template and supplies its
parameters, trigger, conditions, repeat policy, priority and enabled flag.
The engine validates the definition and the template's parameters before
saving. Invalid definitions found on disk are logged and skipped.

The initial `move_to` template takes the same parameters as the body task
and has a 60-second timeout. Loot and stow are separate upcoming templates;
this engine alone does not collect or store items.

## Control API

`GET /api/behaviours` lists definitions and runtime status. All endpoints
use the panel's existing authentication. Mutations also require
`X-Renki-Act`, as other panel controls do. `POST /api/behaviours` accepts:

- `{"name":"trip","action":"save","definition":{...}}`
- `{"name":"trip","action":"start"}` or `stop`, `enable`, `disable`, `delete`.
- `{"name":"trip","action":"clone","target":"another trip"}`.

Clones start disabled. Editing or disabling an active instance stops its
current run. Start enables a disabled instance and requests one run now;
starting an already running instance does nothing. Stop clears that
instance's pending work and stops its run. Disable also persists the off
switch, preventing later automatic triggers. Enabled timers and events
can trigger again after a stop. A direct body order clears all work already
queued by the engine and interrupts its run.

For example, save a disabled movement definition with:

```json
{
  "name": "visit workshop",
  "action": "save",
  "definition": {
    "template": "move_to",
    "params": {"to": [-36, 36, 8], "stopDistance": 2, "run": false},
    "trigger": {"kind": "manual"},
    "when": {"stranded": false},
    "repeat": {"mode": "once"},
    "priority": 10,
    "enabled": false
  }
}
```

Do not start example coordinates in another world. Definitions can include
`when.near.world` and a position/radius to restrict their world and location.
`just vm panel` can call these endpoints using credentials held on the VM.
The engine controls do not yet have dedicated page or chat controls; those
are tracked separately.

## Scheduling and conditions

Triggers are `manual`, `event` (with `event`) or `timer` (with `everyS`,
1–86400 seconds). Built-in observation edges emit `inventory_full`,
`drop_appeared`, `idle_at_home` and `nightfall`. Damage, low-health, item-added
and player join/leave events also reach the engine. `stock_low` is accepted
for the forthcoming chest-memory emitter; it is not generated yet.
Pending triggers coalesce into one request per instance. A bounded incoming
event buffer logs overflow rather than blocking the bridge.

`when` conditions are all required, both to start and to keep running:

- `items`: prefab/count minimums; `minFreeSlots`.
- `maxWeightFraction`: 0–1 against weight capacity; `encumbered`, `stranded`.
- `night`: boolean.
- `owner`: optional owner `name`, required `radius`; the player must be alive,
  currently configured as an owner and observed within five game seconds.
- `near`: `world`, `pos` and `radius`; `atHomeRadius`.
- `stock`: a list of `container`, `prefab`, `minCount`, `maxAgeS` tests against
  exact chest snapshots. Missing, future-dated or old observations fail.

Repeat `mode` is `once`, `trigger`, `while` or `until`. The latter two need
`condition` with the same condition fields. They repeat after successful
runs while that condition holds or until it holds. `cooldownS` applies
between runs (at least one second); optional `maxRuns` bounds completed
attempts. Failures do not loop automatically. Timer intervals, cooldowns,
run limits and counters are controller-session runtime state; saved
policies and enabled flags survive restart, but unfinished manual work and
queued triggers are not replayed. A manual Start explicitly begins a new
run allowance.

## Body ownership and audit

The highest-priority ready instance runs; name order breaks equal-priority
ties. A higher-priority instance stops the current run and queues it to
restart. Only one run owns the body, including the gaps between steps.
Owner commands, retreat and home orders revoke that ownership before their
body task is sent. A stale run cannot send another step or a stop that
would cancel the new order, even before the streamed state catches up.
Reconnects, death, logout and world changes also invalidate ownership.

The first engine waits for an idle body. It never borrows control from an
active follow, so jobs cannot take Tas beyond a leader's leash. Supporting
loot detours during following remains part of hauling-template work.
Templates in Go implement bounded steps; instance JSON contains no script,
loops or variables. Each trigger, start, step, finish, failure and interruption
is logged with its instance name. Template steps retain session capability
and allowed-task checks, plus the plugin's normal safety rules. No behaviour
is installed or enabled by default.

Run `just check controller test-packages ./internal/behaviour ./internal/session ./internal/app ./internal/panel`
for focused engine, ownership and API checks. `just check ci` also runs them
under the race detector. Route and combat checks in docs/TESTING.md remain
separate live tests.
