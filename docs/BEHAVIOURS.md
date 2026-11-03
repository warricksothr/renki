# Behaviour engine

Named definitions in `settings.json` now run through the controller's
behaviour engine. Each definition selects a Go template and supplies its
parameters, trigger, conditions, repeat policy, priority and enabled flag.
The engine validates the definition and the template's parameters before
saving. Invalid definitions found on disk are logged and skipped.

The `move_to` template takes the same parameters as the body task and has a
60-second timeout. Bounded `collect`, `gather`, `stow`, `combat` and `loot`
templates use the same engine, permissions and body ownership rules described below.

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
runs while that condition holds or until it holds. The repeat condition only
decides whether another run starts: a run in progress is stopped by `enabled`
and `when`, never by its repeat condition, so a collect whose `until` goal is
reached mid-trip still stows and counts as a success. A manual, timer or
event run preempted by higher-priority work restarts from its first step
afterwards on the same terms; idle work is dropped and waits for its next
dwell. `cooldownS` applies
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

The engine waits for an idle body, with one exception: an owner's follow can
be lent to a *passive* instance while the followed player stays in one place
(see [Idle and passive](#idle-and-passive)). Every other running task, and a
follow while its leader walks, keeps the body, so jobs cannot take Tas beyond
a leader's leash.
Templates in Go implement bounded steps; instance JSON contains no script,
loops or variables. Each trigger, start, step, finish, failure and interruption
is logged with its instance name. Template steps retain session capability
and allowed-task checks, plus the plugin's normal safety rules. No behaviour
is installed or enabled by default, except the built-in `Loot` instance.

Run `just check controller test-packages ./internal/behaviour ./internal/session ./internal/app ./internal/panel`
for focused engine, ownership and API checks. `just check ci` also runs them
under the race detector. Route and combat checks in docs/TESTING.md remain
separate live tests.

Fresh remembered chest contents can also satisfy stock conditions after a
controller restart. Live observations take precedence over a remembered
copy of the same chest. See [MEMORY.md](MEMORY.md) for persistence and ages.

## Idle and passive

Every instance has a `mode`: `idle` (the default, also written by leaving
`mode` out) or `passive`.

- **Idle** instances run only when Tas has no task, as described above.
- **Passive** instances also run alongside an owner's follow. They still run
  when Tas is idle, like idle ones.

Only a template whose work stays around one player can be passive; today
that is `loot`. Validation refuses `passive` for any other template and for
idle (marker) triggers. A passive instance never starts a follow itself.

### Lending a follow

A follow is *an owner's* when it was sent outside the engine (chat, the
panel's follow button, the voice) and it is the body's only running task.
While Tas runs one, the engine does not queue idle instances, and checks the
passive ones in priority order, name order breaking ties: the *passive
list*. The built-in list is the `Loot` instance alone.
Work-alongside gathering joins it in its own ticket.

A passive instance borrows the follow when all of these hold:

- the followed player has stayed in their settle area (see [Reach](#reach))
  for at least the instance's `settleDelayS`, by the instance's own settle
  tracker, so the settle time is tracked in one place, the loot tracker;
- its trigger fired and it is ready (loot: an eligible drop is in reach of
  that player);
- its `when` conditions and repeat policy allow a run.

The engine then takes the body with `Session.TryLoanFrom`, which accepts
exactly that running follow. The run's first step replaces the follow, which
ends `superseded` and reports `Lent()`. A loan is not an owner order:
`ControlRevision` does not change, so queued work is kept. During the run the
anchor is the followed player, so loot works only around them, inside its
reach.

### Giving it back

When the run ends, the engine sends the same follow again through the same
handle (`Automation.Return`): same player, arguments, timeout and origin. It
does so whether the run succeeded, failed, timed out or was cut short. If
the follow cannot be sent, because `follow` is no longer allowed for example,
the run's step is stopped, as it is for any interrupted run, and Tas tells
the followed player it cannot follow them again. The
new follow is the owner's again and can be lent again. A run that sent no
step leaves the follow running and sends nothing.

### Yielding

- **The player moves on.** When the followed player leaves their settle
  area, by stepping beyond its edge plus the slack or going 30 s unseen, the
  run ends at once (`followed player moved on`) and the follow restarts.
  Tas falls in behind them.
- **An owner order wins.** Any new body order during the run (chat, panel,
  voice, home) revokes the handle, as it revokes every run. Nothing is sent
  again, and the log records `follow not returned`.
- **Survival first.** Retreat's flee and defence are direct orders and win
  the same way. They end a lent follow as they end a plain one: the follow is
  not restarted. Logout, a reconnect or a world change also revoke
  the handle, so no follow is restarted. A death revokes it too, but the
  lent follow is then handed to the chat brain (`Options.ResumeFollow`,
  `Brain.ResumeLentFollow`). That leaves the same resume after the respawn
  as for a follow that was running when Tas died, so Tas comes back to
  following once it has respawned, if the home rules say resume. The
  engine learns of the death a tick late, so the home keeper takes that
  resume only until it has handled the respawn
  (`Keeper.ResumeAfterDeathWhileDead`). A zero-health frame that the dead
  phase never follows also ends the loan, but then no respawn comes and
  the follow is not restarted. The game is not known to send one.

A lent follow (`Task.Lent`) ends `cancelled`/`superseded`, which the chat
brain answers as it does every superseded task. It says nothing, does not
send Tas home and keeps its anchor, and the follow restarts. A follow that
ended on its own before the run's first step went out is not marked lent and
is not sent again. A lent follow that failed before the step replaced it,
a death for example, is handled as any follow's failure is.

### Visibility

`GET /api/behaviours` gives each instance `passive: {follow, player}` when
its latest run borrowed a follow: the follow's task ID and the player
followed. With `running`, the current run is passive. The audit log's
`behaviour started` record carries `passive: true`, `follow` and `player`.
The end of a loan logs one of four records:

- `follow returned`, with the new task `id`;
- `follow not replaced`: nothing is sent, because the follow is still
  running or it ended on its own (failed or timed out) before or while the
  run's step went out;
- `follow not returned`: an order, a death or the world intervened;
- `could not return the follow`, a warning: sending it failed, the step is
  stopped, and Tas tells the player it cannot follow them again.

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

## Loot near owners

The `loot` template picks up drops near the player Tas is with, its
*anchor*, while Tas is idle or, if the instance is passive, while it follows
a settled owner (see [Idle and passive](#idle-and-passive)):

```json
{"items":["Wood","Resin","BeechSeeds","Feathers"],"radius":30,
 "maxWeightFraction":0.8,"maxTargets":16,
 "baseRadius":8,"growS":120,"settleDelayS":5,
 "settleRadius":6,"settleMaxRadius":10,"settleSlack":2}
```

Every field is optional; these are the defaults, and a zero means the
default. `items` are exact prefabs (no `wood` or `*` groups). Loot's reach
is measured from the anchor, who must be alive and in fresh perception, and
grows while the anchor stays in one place (see [Reach](#reach)): from
`baseRadius` up to `radius`, at most 40 m. Each run takes the nearest eligible
drop, sends `pickup`, and waits for a newer inventory frame that holds more of
that item before choosing the next. A success without that gain fails the
run. A drop someone else took (`invalid_target` or `target_lost`) is skipped.
It shares `collect`'s capacity check: whole free slots and the larger of live
and pack weight, against `maxWeightFraction` (0.1–0.9) of carry capacity.
A run ends after `maxTargets` (1–64) pickups or when nothing is eligible. It
carries what it picked up; delivering it is separate work. A successful run
records `picked up N near <anchor>` as its last result, which the panel shows
on the instance's card, N counting only pickups the pack confirmed.

### The anchor

The anchor is the player Tas's last order served: the player it was told to
follow or escort, or else the player who gave the order. The chat brain
records it when it sends a body order (`Brain.Anchor`): the chat commands
(follow, escort, stop, wait, come here, go home, pick up, use, cook, fetch),
whether typed in game or in the panel's chat box, and a task the voice
proposes. A speaker counts only while Tas can see them in the world, so an
owner typing in the panel from outside it does not become the anchor. A
refused order does not move it, and neither do the food, gather
and waypoint commands, which start goals and walks rather than work beside
someone. The panel's task buttons have no speaker, so only their `follow` and
`escort` set the anchor, to the player followed; the others leave it. The
anchor must still command Tas, as an owner or an `operator` profile;
companions and guests never anchor loot, and an anchor who loses that grant
stops anchoring it. Drops near other owners, outside the anchor's reach, are
left alone, and with the anchor out of sight or dead loot does not run.

Until anyone has given an order since the controller started, a single
commander in sight stands in as the anchor. With several in sight and none
chosen, loot waits rather than guess. The deliberate-drop rule below still
looks at every live owner, since any of them can put something down.

### Reach

A player walking through gets only a short reach, and one who stops to fell
trees or fight gets a wider sweep. The shared loot tracker follows every
commander in sight with a *settle area* (`Settle`, in
`internal/behaviour/settle.go`):

- **Opening.** The first sighting opens a circle of `settleRadius` (6 m)
  around them, and the settle time starts.
- **Drifting.** While they stay inside it, its centre follows their position
  as an average with a 30 s time constant (`SettleDrift`), so moving the work
  over a few metres draws the area along without resetting it.
- **Widening.** After `settleDelayS` (5 s) in the area, a step into the
  `settleSlack` (2 m) band past its edge widens the circle to that distance,
  up to `settleMaxRadius` (10 m, or `settleRadius` if that is larger). Before then the area does not widen, so a
  jog or run through leaves it within the delay, and a walk through gains
  less than a metre of reach before it leaves.
- **Leaving.** A step beyond the edge plus the slack, or 30 s unseen
  (`SettleGap`), opens a fresh area where they stand, and the reach falls back
  to its base.

The reach is `baseRadius` (8 m, or `radius` if that is smaller) until the anchor has been settled
`settleDelayS`, then grows linearly to `radius` (30 m) over `growS` (120 s),
and stays there (`LootReach`). With the defaults, a drop 20 m from the anchor
is in reach after about 70.5 s settled. `baseRadius` must be at most `radius`,
`growS` at most 600 s, `settleDelayS` at most 120 s, the settle radii at most 20 m
with the maximum at least the opening radius, and `settleSlack` at most 10 m;
all of them are above zero, since a zero means the default.
The default shape is tracked for every commander all the time. An instance
with a different area shape gets its own tracker from the first time it
checks for work, and that tracker is dropped after 10 minutes without a
check (Tas busy, or the instance gone), so it starts again from the base
reach. The defaults are starting points, to be tuned with the panel
view (TKT-01M6EYEMTX).

### The deliberate-drop rule

An owner who puts something down means to leave it. A drop is deliberate when,
the first time perception shows it:

1. a live owner is within 2 m of it, and
2. no tree, log or stump broke within 10 m of it from 10 seconds before
   until 2 seconds after. The 2 seconds cover a split log whose wood is seen
   a frame before the log goes.

Loot ignores a deliberate drop for 5 minutes from that first sighting, then
treats it like any other. A *break* is a tree, log or stump that was in one
fresh perception frame and is missing from the next, placed where it was last
seen. Felling and splitting therefore excuse the wood they produce, even when
it lands at the owner's feet. The rule is `DeliberateDrop` in
`internal/behaviour/loot.go`, a pure function of the drop's position and
time, owner positions and breaks, so that the haul report (TKT-01M6DDFTDD)
can score pickups by the same rule.

Apart from that 2-second lag, a drop is judged once. Its verdict survives a missed frame or Tas walking out
of range, and is forgotten only after the drop has been out of sight for
5 minutes. Limits: a drop already lying near an owner when Tas first sees it
counts as deliberate if nothing broke near it. A tree that
simply leaves perception range as Tas walks counts as a break. A drop not yet
judged is never picked up.

### The built-in instance

On its first start the controller saves an instance named `Loot`: this
template with the defaults above (it saves `items`, `radius`, `maxWeightFraction` and `maxTargets`, and leaves the reach fields to their defaults), a 2-second timer, `when.owner.radius` 40,
`repeat` `trigger` with a 2-second cooldown, priority -5 (above idle chores,
below gathering and every owner order), `mode` `passive`, enabled. The timer
queues a run only when an eligible drop is in reach, and Tas is idle or
following a settled owner, so an empty check leaves no log line. Edit, clone,
stop and disable it like any other instance. Disabling is the off switch and
survives restarts. Deleting it brings it back with the defaults on the next
start. A saved `Loot` that is still the earlier idle built-in, on or off,
becomes passive on the next start and keeps its on/off setting. One an owner
edited stays as it is. Set `mode` to `idle` to keep loot out of follows.

The old `preemptFollow` parameter is gone. The mode replaces it, and a
definition that still names it fails validation as an unknown field. No
saved instance can hold it, because validation always refused it.

Focused checks: `just check controller test-packages ./internal/behaviour`.

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
After a tree or rolling log breaks, Tas completes its checked escape leg before
waiting for the descendants to settle. Distance from the old network pivot
cannot skip retreat when striking near the end of a long trunk.

When only passive deer block otherwise safe full trees, gathering may make one
short running approach toward a freshly observed deer within 20 m, stopping 8 m
from its recorded position. Native dry routing and hostile avoidance still apply.
It waits at most 20 seconds for newer observations to show all original falling
and bystander exclusions clear. It never attacks or follows a moving deer.
Approaches have a 30-second bound; their destinations must be within 40 m of the
work site and 150 m of home. At most one per requested tree is allowed in a run.
Players, other creatures, structures and walls keep their original protections.
Wildlife outside the approach bounds receives one 20-second observation window
to leave or come within range, rather than an immediate failure or a longer chase.
With optional work-site defense enabled, a single confirmed ordinary Greyling
pursuer within 40 m gets up to 15 seconds to enter the existing 20 m native defense
range. The unfinished tree/log chain stays pinned while Tas waits. This never
authorizes a chase, a substitute attacker, or combat outside that admission range.
Starting inside the work zone can defend a freshly confirmed pursuer before
repair or travel preparation. A refused harvest receipt without a resource ID
uses the engine's matching task correlation and the original explicit target;
it never marks the interrupted resource complete or changes the retry target.

Before leaving home, `returnHome` gathering repairs a carried axe below 90%
durability at a currently observed home workbench. When started away from home
without a nearby home workbench, it
first makes a dry, hostile-aware return, verifies the home floor and a newer
local scan, then repairs and resumes the outbound trip with the original cargo
baseline. A missing home workbench stops preparation without looping.
It uses `interact` with
`repair:true`, then requires a newer actual inventory reporting an axe at least
90% repaired. Failed, stale or ineffective repair cannot start the journey.
The station routine also repairs other eligible carried gear through the same
normal game button. This does not craft replacements, upgrade equipment or
infer that a workbench can repair gear needing a forge.
Players near home do not prevent repair or the outbound trip. Nearby hostiles
or confirmed pursuers still prevent preparation; all bystander and falling-tree
exclusions apply again before harvesting.

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
Checked movement waypoints use a 0.6 m support-height tolerance, separate from
the home marker's 1.75 m eye-height allowance. A body within 0.4 m of a tight
corner may consume it only after a fresh physical probe validates the direct
connection to the following point; the final point and another floor cannot
be skipped this way.
Dry probes also require the rise to be shared across their midpoint: each
half-edge allows at most 0.3 m. This preserves ordinary stair treads while
rejecting a vertical floor lip that endpoint-only step checks can misread.
All three heights use measured support. Starting footing uses the nearest
surface within the existing 0.6 m downward sampling range or one 0.3 m tread
above the feet while stepping; a higher floor cannot become footing.

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
Explicit definitions may opt into `defend:true` (default false). At the work
site, one freshly observed unstarred Greyling actively targeting Tas within
20 m may trigger the existing native kite/melee task. This requires normal
combat permission, food, 80% health, 40 stamina, a usable axe, safe footing and
no nearby bystander or additional threat. At most two encounters and the same
overall work budget are allowed, with four minutes remaining before dispatch.
An interrupted refused cut retries that exact resource; it does not count as
a break or authorize another tree. Resumption requires the exact native
Greyling death receipt and subsequent fresh state/perception with no living
target. The chain and original product baseline survive defense. Injury,
failed combat or an unavailable interrupted resource stops the run. Owner chat
does not silently opt into combat.

Controller-level full gathering/delivery remains a separate live check. Run
`just check vm gather-trial --gather-trees 2` to require two distinct complete
trees, task-matched break proof, combined gains-only storage and final home.
`--gather-site X Y Z` supplies a known location hint without binding a stale
resource ID; the controller acquires fresh eligible trees there.
`--gather-defend` explicitly enables the optional defense for that trial and
temporarily enables combat permission, restoring it off during cleanup.

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
See SELF_DEFENSE.md for single-enemy limits and live-test status. Autonomous
combat discovery still selects Greylings; wild Boars require an explicit target.

## Food work while idle near markers

Select a saved `berries`, `cooking` or `storage` marker in the panel's place
editor and enable **Idle food work**. Other marker types have no automatic
food action. Nothing is enabled by default. The authenticated behaviour API
exposes the same preset to tools:

```json
{"action":"idle-food","marker":{"world":"your world","name":"pantry","kind":"storage"},"enabled":true}
```

The preset waits for 30 seconds of fresh idle observations in that marker's
zone, at most 2 m above/below its floor. Point markers use a 3 m work radius;
large zones are limited to 40 m. Leaving, editing or deleting the marker,
world/session changes and owner orders discard queued work. Active work uses
the same pinned marker and body lease. Safety checks also run during native
steps. Disable persists the off switch and stops active work; Stop clears the
current attempt and dwell, allowing another attempt after a new dwell period.
An enabled policy may resume after an owner task ends and Tas becomes idle.

Each attempt lasts at most six minutes:

- `berries`: collect up to four fresh visible raspberry, blueberry or mushroom
  targets, then return to the marker. Skip nearby creatures, unsuitable floors,
  insufficient capacity and yields exceeding 20 carried items of that kind.
  This is wild-food foraging; it does not plant or harvest cultivated crops.
- `cooking`: prepare at most three carried raw meats/tails using a currently
  seen, heated, empty supported cooker within 4 m. Match native removal,
  spawned output, pickup and inventory receipts. Missing supplies or uncertain
  partial cooking stop the batch. It does not withdraw ingredients or fuel fires.
- `storage`: deposit surplus ready food into currently visible chests within
  4 m and the marker zone. Keep two of every supported ready-food prefab.
  Existing exact transfer receipts prove delivery; inaccessible/full boxes
  leave an actionable result rather than discard food.

All actions require fresh surroundings, dry footing, an active meal, at least
25 HP and 80% health at admission, no nearby hostiles and no injury during the
batch. Newly eaten food may raise maximum HP before healing; continued work
still requires 40% health and checks actual injury separately. There
is a five-minute cooldown after completed/failed attempts and a limit of 12
attempts per controller session. Empty/depleted sites therefore back off.
Definitions and enabled flags persist; counters and dwell reset on restart.
The editor shows enabled state, attempt count and last result. Cross-site
production, hunting, crop cultivation and pressure-driven return/repair remain
separate goal work.

Generic definitions can use `trigger.kind: "idle"` with `marker` and `dwellS`
(10–600 seconds). They require `repeat.mode: "trigger"`, `cooldownS >= 60` and
`maxRuns` from 1–100. The `marker_food` template takes `{ "marker": {...} }`;
its marker must match the idle trigger. A manual trigger can run one bounded
food batch when already at that marker. `collect` also supports `carryOnly`
without storage destinations or a return trip; the enclosing routine owns
what happens next.

## Rest while idle at home

`marker_rest` uses the same idle scheduler and current named marker. Its
parameters are `marker` and `sleep`. It selects only a freshly observed bed
positively owned by Tas, within12m of home and on the current floor. It never
claims another bed. Rest requires fresh surroundings, known game time,
grounded shelter near home, no water, no close threat and no new injury.
The selected home, marker and bed stay pinned for the bounded90s attempt.

The authenticated API supplies two separate editable presets:

```json
{"action":"idle-rest","marker":{"world":"your world","name":"home rest","kind":"place"},"sleep":true,"enabled":true}
{"action":"idle-rest","marker":{"world":"your world","name":"home rest","kind":"place"},"sleep":false,"enabled":true}
```

Night mode waits15s of idle time, then sends the normal bed sleep interaction.
Success needs both the native receipt and a newer `in_bed`/`sleeping`
observation beside the selected bed. Other players still determine whether
the night can skip, and native roof/fire/wet/enemy refusals remain visible.
Day mode waits120s, at lower priority than food/cargo work, then makes one
dry/hostile-aware move near the bed and releases the body. It waits nearby
without trying an invalid daytime sleep or holding a task open. Already
being within2m horizontally of the bed needs no movement. Native waiting approaches use
a2m arrival radius and require a fresh sheltered floor observation within
2m and75cm vertically, avoiding a demand to climb into crowded bed frames.

Both presets have a five-minute cooldown and100-attempt controller-session
limit; none are enabled by default. Their ordinary behaviour rows expose
enable/disable, results and manual start through the shared panel/API. An
active chore or human order prevents idle admission. New commands revoke
the rest lease. The marker may be left only for the nearby owned bed under
the rest guard. All idle triggers stop queuing chores while Tas is in bed
or sleeping, including after the native interaction has completed. The game
wakes him normally after sleeping; explicit new orders can interrupt rest.
Automatic eating also waits while attached to the bed or sleeping.
