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

Fresh remembered chest contents can also satisfy stock conditions after a
controller restart. Live observations take precedence over a remembered
copy of the same chest. See [MEMORY.md](MEMORY.md) for persistence and ages.

## Bounded collection and stowing

`stow` and `collect` are manually configured templates. They do not install
background jobs, choose arbitrary storage, or interrupt a running follow.
Use the existing behaviour API to save a disabled definition, then start it
when its world, site and destination chest IDs have been checked.

`stow` takes these parameters:

```json
{"world":"DizzysHellheim","items":["Stone","Flint"],"from":"pack",
 "to":["<first-chest-id>","<overflow-chest-id>"],"keep":{"Stone":5},
 "overflow":"tell"}
```

Items are exact prefabs or the selectors `wood`, `ore`, `food`, `*`. Wood
covers Wood, FineWood, RoundLog, YggdrasilWood and Blackwood. Ore covers
CopperOre, TinOre, IronScrap, SilverOre, BlackMetalScrap, FlametalOre,
FlametalOreNew and CopperScrap. Food uses the game's Consumable item type
(including potions). All equipped items and all gear remain protected.
`keep` retains at least the named count, including equipped stacks. `from`
currently accepts only `pack`. There may be at most 16 ordered chests.

Remembered chests provide locations; old contents never establish capacity.
Each chest is approached if necessary and attempted once. The game's transfer
fills compatible partial stacks first, then empty slots. Exact task-matched
receipts and updated pack observations determine what remains for the next
chest, including partial transfers into a chest with less room than expected.
Busy, vanished, full or unreachable chests can fall back to the next named
chest. Permission failures and cancellation stop the run. Nothing is dropped
or destroyed. A shortfall remains in behaviour status and the panel's
Storage needed message. `tell` also announces it in game; an identical
shortfall is repeated only after five minutes or when its counts/reason change.
`leave` suppresses the announcement. Rerun after resolving storage; a
successful run clears its shortfall.

`collect` performs one bounded collection-and-delivery trip:

```json
{"site":{"world":"DizzysHellheim","pos":[-50,32,18],"radius":12},
 "items":["Flint","Stone","Wood"],"to":["<destination-chest-id>"],
 "keep":{"Wood":5},"maxTargets":8,"maxWeightFraction":0.8,
 "drops":false,"return":true}
```

It visits the site when outside its radius, chooses current pickables by their
output prefab, collects their spawned drops, stows the selected carried items
above `keep`, then optionally returns to the site's centre. It requires exact
item prefabs, a radius up to 40 m and 1–64 target attempts. Spawned products
count toward the target budget; finishing a picked target may exceed that
budget, up to a hard cap of 128 attempts. Each body step has a 60/90-second
timeout. Missing fresh observations wait at most five seconds.

Only resources with known primary yields qualify. Extra/random-product and
theft-provoking pickables have no automatic yield. Carry checks reserve empty
slots for the whole expected yield and use the larger of live state weight
and the pack's item weight. The default limit is 80% of carry capacity and
can be configured between 10% and 90%. Yield counts include possible skill
bonuses and are upper bounds, not evidence that anything was collected.
After use, a fresh inventory or visible product is required. The normal game
pickup/interaction and chest-access rules remain authoritative.

Arbitrary existing drops are not chosen unless `drops:true` explicitly opts
into collecting the selected materials in that area. Spawned products near a
pickable are followed as part of that target. Normal game auto-pickup may
still take incidental items along a route. Automatic owner-drop inference,
work-site discovery, leader detours/resumption, resource exploration, tool
swings and chat shortcuts remain separate work. This template is suitable
for staged loose flint/stone/branch tests; it does not fell trees or mine rock.
