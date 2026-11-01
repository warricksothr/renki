# Food resupply

Owners and configured operators can request `Tas resupply food` or `stock up
on food`. Tas needs an idle body, a known home within 150 m, and the ordinary
movement, withdrawal, interaction, pickup and food-use permissions. This starts
one durable `food_resupply` goal. It does not replace arbitrary current work.

The goal returns to its pinned home, checks its pack, then searches the finite
set of unique visible chests within 32 m of home. Last-known contents rank
candidates; actual normal chest access and transfer receipts determine what
is present. A missing food probe returns actual contents even when native
status is `invalid_target`. That expected negative advances the search.
Distinct empty chests do not count as identical failures. Each chest/item/phase
pair is tried once, with the ledger saved across explicit resume. Inaccessible
chests are skipped. Permissions and execution faults still stop the goal.

Ready meals take precedence. If allowed storage cannot supply the reserve,
the goal obtains supported raw boar/deer meat or neck tails, and wood if the
selected empty station is cold. It uses the existing native single-input
[cooking routine](COOKING.md), including its one-wood supporting-fire receipt,
raised-floor approach and multiplayer output receipts. It can cook at most
four batches. No unconfirmed insertion, removal or pickup is blindly retried.
It cannot bootstrap cooking without an active meal: the native health/food
gate remains enforced. A true shortage becomes `needs_plan` with its evidence.

Completion requires a small carried reserve (default two meals, configurable
from one to three), an active meal with at least five minutes remaining, and
80% health. Eligible empty food slots are filled before measuring the reserve.
The food keeper yields while this goal owns eating, then resumes afterward.
Transfers require the exact task/chest/direction receipt plus a newer pack
and measured gain, allowing one directly observed food refresh. Cooking
requires matching removal yield, output IDs, per-ID pickups and the final
native inventory receipt; a success string alone cannot prove it.

The goal pins world, home, candidate identities and injury counter, and retains
typed phases, dispatch intents, attempts and receipt summaries. All of its
native tasks are charged before dispatch. Limits are ten minutes, 256 native
tasks and four minutes of observation/recovery waits. There is a 64-chest
storage bound; larger areas require explicitly narrowing storage rather than
silently ignoring the remaining boxes. A lost checkpoint stops progression.
After an outage, explicit resume preserves spent budgets. A pending uncertain
action cannot be replayed. Stop or a new owner order cancels all continuation.
The bundled restricted Starlark policy can select only the host's proposed
action and cannot grant permissions, change targets or declare false success.

New chat escort requests enable food maintenance. Native escort yields only
at a healthy, grounded, dry, quiet follow/wait boundary when reserves are empty
or its remaining food is under three minutes. Defence and escape take priority.
The controller then runs the same food goal, retaining the original leader.
Verified completion can hand back to escort through the same body lease,
using the current owners/operators as the protected group. A newer body order,
world/connection change or revoked commander prevents that continuation.
An unavailable home or failed resupply leaves Tas stopped; it does not invent
a route or resume the escort after a restart. Direct native escort requests
can opt in with `resupply:true`; the native default remains false.

The existing authenticated goal API accepts, for example:

```json
{"action":"start","spec":{"kind":"food_resupply","world":"Midgard","center":[0,30,0],"radius":150,"reserve":2,"storage":["current-chest-id"]}}
```

`center` must match current known home. Omit `storage` for the full eligible
area. An optional validated `escort` object pins a continuation; its leader
must still be a configured commander at handoff. `GET /api/goals` provides
compact summaries with phase/counts and recent receipts. `GET /api/goals?id=ID`
returns one full checkpoint and candidate ledger for diagnosis. Missing receipt
data remains unknown, rather than being reported as zero observed events.
These are bundled host adapters; general procedure authoring tools remain
proposed in [PROCEDURE-TOOLS.md](PROCEDURE-TOOLS.md).

Use `just check controller test-packages ./internal/goal ./internal/behaviour
./internal/food ./internal/session ./internal/app ./internal/chat` and
`just check plugin test-core EscortTests` for focused checks; `just check ci`
is the full gate. `just check vm food-check --join --require-cooking`
supervises the production goal, without planning or dispatching its steps.
`--storage` can scope a real supply test; it does not claim excluded chests
are empty. `--require-fuel` additionally requires an actual cold-fire fuel
receipt, never just a successful hot-station batch. Native traces and complete
checkpoints are saved in `.checks/food-*`.

Trials log out after unsafe, uncertain or failed outcomes. `--keep-online`
retains Tas after success only with a fresh safe idle shelter observation near
home and no nearby observed threats. Native plugin changes still require a
client restart; controller-only changes can use `just vm deploy-controller`.

New food goals can also acquire missing supplies after storage is exhausted.
They prefer exact live ready-food yields and loose raw ingredients, then visit
remembered raspberry, blueberry or mushroom sites within 80 m of home. A
remembered site offers a position to check; it never proves a harvest or an
empty area. Cold cooking fires can obtain wood from observed branches or
loose wood when storage has none. The native movement task checks dry ground,
known obstacles and hostile avoidance for each visit.
Source admission also rejects currently perceived hostile exclusion areas in
the horizontal plane: 16 m for an ordinary Neck, 24 m for an ordinary Boar,
and 40 m for other or starred hostiles. These match native route clearance;
fresh perception can rule out an occupied source but cannot establish safety
from threats loaded only by the plugin. Other nearby creatures retain the
existing conservative exclusion, and a pursuing creature blocks further work.

If those sources cannot supply food, current attack permission can admit an
explicitly selected isolated unstarred wild Boar or Neck within 30 m of Tas.
Active food, recovered health, stamina, pack capacity and the native weapon,
bystander, pursuit and retreat guards remain required. There is no blind
wildlife discovery attack, and no hunt when combat permission is disabled.
Observed death is a separate receipt from food collection: meat must still
appear as an eligible live drop and reach the pack. Any injury still stops the
food goal; this adapter does not reset the injury baseline to continue hunting.

The acquisition checkpoint limits a goal to 12 resource interactions/pickups,
six remembered-site visits and two hunts, within the original overall task and
time budgets. A failed visit consumes its attempt but does not assert that the
site was empty. Pickable products have a short persisted observation deadline;
matching nearby drops can be picked up once, or fresh measured inventory gain
can prove ordinary auto-pickup. Success text, disappearance and remembered
stock do not prove a gain. Every uncertain action remains non-replayable.
After acquisition, Tas returns to the original home floor before cooking,
completion or escort handoff. Older saved goals without an acquisition
checkpoint retain their original storage-only scope on explicit resume.

`just vm food-check --require-acquisition` additionally requires a verified
collection receipt outside storage. The panel shows bounded forage-visit,
collection and hunt counts; the full checkpoint retains source attempts.
Controller-only deployment now preserves an offline game as well as a running
one, keeping the independent panel available without launching Valheim.

## Outdoor startup recovery

`just vm recover-home WORLD` can recover an adequately supplied bot saved
outside, without disabling `RecoveryGuard`. It reads that exact world's pinned
home from controller memory while the game is offline and arms a five-minute,
one-shot native order. The plugin consumes the order before moving. Invalid or
wrong-world orders log out; an absent order retains passive sheltered healing.

Admission requires at least 50 HP and 70% maximum health, two active foods,
25 stamina, dry grounded feet and six metres of hostile clearance. Normal
native dry movement avoids hostiles. Injury, water, less than eight stamina,
close threats, cancellation, bridge/session loss or permission loss log out.
The return permits 120 seconds, 160 metres travelled and at most 80 metres
from home. It cannot combat, collect supplies or accept ordinary body tasks.
An operator stop aborts it, and reconnect cannot resume it.

Arrival requires the original home floor, actual shelter and twelve metres of
hostile clearance. It then resumes passive healing; ordinary tasks remain held
until the existing 95% health guard releases them. The supervisor keeps the
client online only after same-world, same-connection safe-home evidence. On
failure it stops the game and restores the independent panel. Observations,
compact results and an audit window stay under `.checks/recovery-*`.
Use `just check plugin test-core RecoveryReturnTests` and
`just check vm test-recover-home` for policy bounds and cleanup checks.

`just vm food-check --reserve 2 --escort Nimah --keep-online` also verifies
the food goal's durable handoff to a named current commander. This bounded
check disables collection and defense, requires exact native request,
acceptance/running receipts and the pinned escort identity, and observes ten
seconds of quiet follow/wait at safe home before stopping escort. It proves
the handoff without requiring the leader to be present; it does not prove
following a moving player or defending one. Normal escort settings and combat
permission remain separate. `just vm trial-status` reads compact progress for
food, cooking, recovery and escort evidence as well as harvesting/combat.

## Household storage targets and idle episodes

A `food_resupply` goal can add `stock` to produce food beyond Tas's personal
reserve. `stock.targets` maps supported ready-food prefabs to desired totals in
an explicit `storage` chest set. Counts are1–20 per food and at most40 overall.
For example, submit through `/api/goals` with `action:"start"` and this spec,
using the current world, its actual home coordinate and current chest IDs:

```json
{
  "kind": "food_resupply",
  "world": "your-world",
  "center": [-35.9, 37.66, 6.53],
  "radius": 200,
  "reserve": 2,
  "storage": ["1:70031"],
  "stock": {
    "targets": {"Mushroom": 4, "NeckTailGrilled": 2},
    "sources": ["west mushrooms"]
  }
}
```

The panel's Household food controls configure one food target, destination IDs
and named forage markers; the API supports several foods in the same episode.
A named source must be a current `berries` marker, no more than120m from home
and within the goal's travel envelope. Its position, kind, radius and revision
are pinned. Editing/deleting it stops the episode. Sources select search areas;
only currently observed safe pickables/drops can earn collection receipts.
Explicit household source markers replace automatic remembered-site search;
no household hunts are admitted. Omitting sources keeps the existing bounded
live-yield/remembered-forage search, without hunting.

Personal eating, recovery and carried reserve take precedence. At home, normal
one-meal withdrawal requests inspect each unique destination; these can move
one available meal into the pack. Matching container snapshots are authoritative
for three minutes in the current controller run. Memory cannot satisfy a target,
and unavailable or missing chests leave stock unknown. A moved probe meal is
retained for personal needs or returned as surplus. Deposits require an exact
task/chest/direction receipt plus newer inventory showing the measured decrease.
Partial/full destinations are tried once per food before requesting a new plan.
Only freshly observed stored totals count toward household completion.

Pressure decisions favour existing surplus deposits, usable home ingredients,
then safe nearby sources. Collection can continue across multiple yields;
fulfilled cargo, low free slots,70% carry weight, raw ingredients needing home
cooking, distance or danger pressure commit the episode to delivery. It keeps
that return phase until arriving on the original home floor.
Once household work has started, returning takes precedence over remote personal resupply or passive
healing when carried reserves fall short, active nourishment drops below five
minutes, or health needs recovery. Pending collection gains are accounted for
first; improved food or capacity during the return does not cancel delivery.
Unrelated cargo that prevents further work reports a capacity problem. Sources
are visited at most once per episode; depleted/unsafe areas do not restart their budgets.
The existing ten-minute deadline,256 tasks,12 collections, six visits and four
cooking batches remain hard bounds. Long explicit source visits/returns permit
180s within that deadline. Missing fuel, stations or sources request a new plan.
After each household cooking batch, a precise dry move returns to the home
waiting point before eating or passive healing. This slice does not cultivate
crops or repair equipment.

“Enable idle food work” saves a `household_food` behaviour at the chosen storage
marker:30 seconds idle, five-minute cooldown, at most12 episodes per controller
session. Each episode creates its own durable goal under the engine's existing
body lease. It may leave the admission marker to collect, while marker identity,
world, injury, threat, footing and ownership checks continue. Commands interrupt;
queued and active episodes never restart uncertain native actions. The behaviour
status exposes its `episodeId`, including the last completed episode, and the
goal reports known stock, shortages and the current pressure decision. Idle work
is opt-in and can be disabled in the panel.

Use `just vm food-check --storage IDS --stock-target Mushroom=4 --sources
"west mushrooms" --keep-online` for a supervised one-shot trial. Repeat
`--stock-target` for several foods; `--require-deposit` requires a verified
household deposit. Add `--idle-marker "home storage"` to verify the idle dwell
and episode handoff. That helper disables its temporary behaviour afterward.
