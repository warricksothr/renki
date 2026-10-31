# Behaviour engine

Named definitions in `settings.json` now run through the controller's
behaviour engine. Each definition selects a Go template and supplies its
parameters, trigger, conditions, repeat policy, priority and enabled flag.
The engine validates the definition and the template's parameters before
saving. Invalid definitions found on disk are logged and skipped.

The `move_to` template takes the same parameters as the body task and has a
60-second timeout. Bounded `collect`, `gather`, `stow` and `combat` templates
use the same engine, permissions and body ownership rules described below.

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
If an approach fails as unreachable/stuck and the known chest is within 10 m
of configured home, stow tries that home point once before giving up on the
chest. This is bounded recovery, not a saved entrance route.
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

## Bounded beech gathering

`gather` visits an explicit world/site, fells a bounded number of isolated tier-zero
full beeches, harvests each tree's newly observed nearby beech logs, and stows only the
products gained during that run. For example, use these template parameters:

```json
{"site":{"world":"DizzysHellheim","pos":[175,84,-260],"radius":30},
 "to":["<destination-chest-id>"],"maxTrees":2,"returnHome":true}
```

An authorized player can say `Tas, go chop wood` (also `chop wood` or `gather
wood`) in game or through the panel. This creates/starts the manual once-only
`Owner wood gathering` instance. It reuses its saved same-world work site when
safe trees confirm it, or travels to a distant saved site and revalidates there.
Otherwise it chooses an isolated tier-zero beech in current perception within
60 m, preferring lower remembered danger, and orders known chests within 10 m
of home. Recent remembered beech locations can supply a distant work-site hint;
the controller rechecks actual trees and surroundings after arrival. Ordinary
beech observations now enter the bounded landmark memory. Home and the selected
site must remain within the 150 m home range. The command requests up to two
complete trees, returns home and deposits only this run's product gains.
Missing tools, food, capacity, fresh observations, safe trees or storage produce
an explanation. Existing active or queued work must be stopped first. `Tas,
stop` cancels through ordinary owner controls. A saved command definition does
not resume itself after restart. User-authored definitions with that name are
preserved; rename one before using the command.

Explicit-site callers can also configure a disabled definition with a manual
trigger and single run, then start it through the authenticated behavior API.
The bot first approaches within 12 m of the site's center (or the site radius,
if smaller), so merely standing inside a broad zone cannot skip the approach
and then fail because its central trees are beyond native acquisition range.
The site radius is at most 40 m. It requires active food, 80% health, dry footing,
a usable carried axe, four free slots and 60 spare carry weight; initial stamina
must be at least 40. Native C# owns every approach, strike, falling-log escape and
collection. Protection distances remain 32 m for standing trees, 8 m for logs,
and 15 m around Tas, with authoritative native checks before swings.

Before leaving home, `returnHome` gathering repairs a carried axe below 90%
durability at a currently observed home workbench. It uses `interact` with
`repair:true`, then requires a newer actual inventory reporting an axe at least
90% repaired. Failed, stale or ineffective repair cannot start the journey.
The station routine also repairs other eligible carried gear through the same
normal game button. This does not craft replacements, upgrade equipment or
infer that a workbench can repair gear needing a forge.

`maxTrees` defaults to one for older definitions and accepts at most three.
At most nine harvest tasks and eight minutes per requested tree are allowed.
New logs observed within 32 m of the felled tree or 24 m of a completed cut
join its unfinished chain. Their identities remain tracked if they roll beyond
the site; dry approaches are bounded to 60 m from that tree. Missing tracked
logs stop the chain rather than silently authorizing another tree. The extra
cut budget covers collateral trunks felled by the first tree.
Each full tree/log chain must finish with measured Wood gain before selecting
another tree. Existing resource IDs at that tree's start
are excluded from its log chain; nearby new IDs are candidates, not proof of
spawn lineage. A successful native task must be followed by fresh inventory and
perception, with its target gone. Missing/unsafe logs stop the run. It requires
at least two log harvests and measured Wood gain before delivery. Baseline Wood,
BeechSeeds, Resin and Feathers are retained; only gains in those products enter
the existing ordered-chest stow routine. Optional return revisits the site's
center. `returnHome` instead pins known home, uses native dry travel to confirm
the home floor before storage, and finishes at home after the deposit. It cannot
combine with `return`. A matching native home receipt and a strictly later pose
are required; standing below a floor does not confirm arrival. Each home move
retains its 120-second bound. Storage failures keep items in the pack and use
stow's shortfall status.

Between complete trees, capacity or axe-durability pressure chooses delivery
rather than another cut. Low stamina gets one bounded 20-second rest; insufficient
recovery chooses delivery. Less than four minutes left also chooses delivery.
Missing/unsafe descendants still stop the run; they cannot authorize cutting a
different tree. These checks are the first bounded work-cycle controls, not the
planned broader cargo-value, repair and resupply policy.

Owner orders, stop, permissions and session/world changes revoke the run through
the engine. The command remembers its requested site in the saved behavior;
it does not infer work sites from watching owners, mine rocks or craft tools.
Focused checks: `just check controller test-packages ./internal/behaviour`.
Controller-level full gathering/delivery remains a separate live check. Run
`just check vm gather-trial --gather-trees 2` to require two distinct complete
trees, task-matched break proof, combined gains-only storage and final home.
`--gather-site X Y Z` supplies a known location hint without binding a stale
resource ID; the controller acquires fresh eligible trees there.

## Combat

The `combat` template delegates one bounded native encounter to C#:

```json
{
  "template": "combat",
  "params": {"world": "DizzysHellheim"},
  "trigger": {"kind": "manual"},
  "repeat": {"mode": "once"},
  "priority": 20,
  "enabled": false
}
```

It requires the current world and active food. Optional params `at: [x,y,z]`
prefer nearby fighting ground; optional `entity` restricts native acquisition.
It dispatches one `attack` with `kite=true` and a 150 second timeout. C# owns
survey, positioning, target acquisition, drawing, kiting, waiting, melee and
escape; phase reports never produce extra movement orders from this template.
The controller and plugin must separately allow combat. Editing an active
instance cancels it through the normal lease/stop handshake; a new manual
start validates the new goal. No instance is installed or enabled by default.
See SELF_DEFENSE.md for the current single-Greyling limits and live-test status.
