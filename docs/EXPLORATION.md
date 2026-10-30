# Exploration goals: proposed after M2 and M3

This is a proposed follow-on to M2 and M3, not a currently supported goal or an
enabled routine. The current goal host accepts `combat_trial` only. Exploration
needs its own validated specification, supporting runners, host budgets and
success evidence before a script can request it.

The umbrella is TKT-01M66PFAYA1NTW2JHNRDKBK42E — Exploration goals after M2 and M3.
Implementation remains in draft until these dependencies are complete:

- TKT-01M62MZN7QMTJ3ST0KRZH15DXZ — M2: the bot hauls and gathers wood with its owner.
- TKT-01M62N2557R401S3E6JECSAPN4 — M3: the bot survives danger and stands with its owner.

The goal is useful knowledge gained during an expedition Tas can finish and
return from. Distance walked, a successful movement task or an absence of
reported enemies does not by itself establish exploration success.

## First expedition

Start with one bounded, owner-selected land expedition from a known waypoint.
For example: survey unfamiliar ground around the meadow camp for up to five
minutes, stay inside the authorized area, record discoveries and return to that
camp. Exact radius, travel allowance and supply thresholds are tuning decisions,
not new defaults or settings implemented by this document.

The request should specify world, origin/return waypoint, permitted area and
biomes, excluded zones, elapsed-time and travel budgets, required new coverage
and supply/health return thresholds. Capture the waypoint's coordinates/version
at start: editing the waypoint must not silently redirect an active expedition.
An authorized change to the return anchor needs explicit goal revalidation.

For the first benchmark, use dry ground in an approved familiar biome and
already supplied equipment/food. No water crossings, boats, portals, terrain
modification or harvesting detours are implicit. Broader travel modes and active
resource gathering require their own capabilities, readiness checks and scope.
M3 defenses remain available under their existing permissions; an exploration
goal does not turn on combat or select fights as an exploration technique.

Host completion requires both newly observed coverage in the authorized area
and fresh evidence of arrival at the return anchor, with a valid supported
arrival position rather than just a small horizontal offset below its floor.
Returning without enough discovery is a partial outcome. Reaching a distant
point while losing the return option is not success. Unsafe conditions may
require a bounded retreat or hold with a diagnostic, rather than forcing a
return through danger.

## What Tas knows about an area

Reuse TKT-01M62VXZ0J9Q2WH4XN4E6N7F27 — Know which parts of the map the bot has
explored, extending its design with separate kinds of evidence:

- **Map knowledge:** minimap exploration or imported/shared knowledge. Retain
  provenance; this does not establish that Tas personally surveyed a location.
- **Observed coverage:** actual scoped terrain/perception observations, with
  age, resolution and the sensor coverage available at that time. Loaded world
  data and a perception radius are not proof that every object was visible.
- **Visited ground:** a bounded representation of where Tas actually travelled,
  including elevation. A sample on a bridge does not establish coverage below it.
- **Route evidence:** directed segments successfully travelled under particular
  movement/obstacle conditions. Keep endpoints, world, age and validation state;
  a remembered traversal is a candidate for revalidation, not a permanent route.

Resources, useful structures, player encounters and injuries continue using
their existing memory stores and precision/age policies. Terrain is currently
session-only; [ATLAS.md](ATLAS.md) proposes its durable storage and exchange,
without starting exploration or equating loaded terrain with personal coverage.
Exploration references those observations rather than duplicating them in every
checkpoint.
Player sightings remain historical evidence, never guaranteed rendezvous points.
Dynamic obstacles, enemies, resources and containers must be checked again
before relying on them. Old static map knowledge can remain useful while its
route or resource evidence becomes uncertain.

Coverage must describe gaps and truncation. A scan missing an object does not
prove absence or depletion. Revisiting a cell with the same evidence does not
increase novel coverage, and fresh timestamps alone do not count as discovery.
Keep durable coverage/frontier histories bounded and consolidated, with compact
summaries for goals/models and details available through the panel/audit.

## Supporting behaviors

These are proposed behavior contracts; names are descriptive, not current API
template names.

1. **Prepare an expedition.** Resolve the anchor, permissions and scope; check
   health, stamina, meals, carried food, suitable gear/durability, weather/night
   constraints when observed, free capacity and a conservative return allowance.
   Use known storage and existing transfer routines for explicitly authorized
   resupply. A missing item produces a specific shortfall; crafting/repair must
   wait for the corresponding tools instead of assuming equipment can be made.
2. **Choose a frontier.** Find a reachable-looking boundary between useful known
   coverage and unknown/stale coverage. Score likely information gain against
   outbound and return cost, observed grade/water, danger and excluded areas.
   Resource or biome searches can later supply priorities, without inventing
   unseen targets. Unknown travel cost remains uncertainty, not zero cost.
3. **Travel one leg and survey.** Go to a bounded approach/vantage point through
   native navigation, observe new ground, and report structured coverage and
   discoveries. C# owns frame-level navigation, support/water checks, local
   obstacle handling and immediate escape. A survey records what the existing
   sensors actually cover; looking in a direction does not create new knowledge
   unless observations support it.
4. **Recover or choose another frontier.** Record failed approach/edge, reason,
   time and relevant geometry/condition revision. Try a bounded alternate
   approach or another candidate; repeated ticks do not clear the failure.
   Retry after meaningful new evidence or an explicit cooldown/revalidation.
   One failed approach does not mark an entire region permanently unreachable.
5. **Return, recover and report.** Turn back before the work budget is consumed,
   reserving resources for return and contingencies. Prefer revalidated return
   options; an outbound drop or one-way descent does not prove reverse travel.
   If return becomes unsafe, follow native retreat priorities and preserve a
   partial diagnostic. Reaching safety does not silently expand the next work
   area. Stowing, eating, repairing or sleeping are explicit supporting steps,
   each bounded and allowed only when its native behavior exists.

The initial expedition should carry sufficient supplies. Supply acquisition,
crafting, camp construction and automatic sleep scheduling can extend the same
contracts later without blocking a small supplied land benchmark.

## Persistent goal composition

Extend the [existing goal host](GOALS.md) instead of adding a second autonomy
loop. The host receives an exploration spec and exposes only the implemented
prepare, frontier, travel/survey and return runners. A compact policy chooses
between those runners, bounded observation waits and a needs-plan outcome.
A conceptual sequence is prepare, choose, travel/survey, reassess, repeat if
budgets permit, then return and report.

Native tasks remain individually bounded. The host enforces total elapsed time,
travel, attempts, distinct frontier failures, wait time, world/area/biome limits
and a reserved return allowance across every child runner. Count actual travel,
including failed approaches and recovery motion, rather than planned straight
lines only. Missing or stale telemetry makes progress/cost uncertain and blocks
another outbound leg. Emergency safety can exceed a work boundary; continuation
requires revalidation and does not quietly increase the authorized radius.

Persist the original scope, baseline coverage, chosen/failed frontiers, supply
and return state, remaining budgets, observation references and outcomes.
Retain intent-before-dispatch and receipt-after-outcome checkpoints. After an
outage, uncertain steps are reobserved, not replayed or counted as discovery.
Resume follows the existing explicit resume rules and requires fresh same-world
readiness; it never resets budgets. Owner stop/takeover cancels the continuation.

Policies and future model proposals cannot change the host's success predicate,
permissions or scope. A model escape hatch receives only the objective, compact
coverage/frontier summaries, relevant discoveries/failures, current readiness
and remaining budget. It can propose supported steps; it cannot invent a route,
new sensing capability, supplies or success evidence.

## Panel and player commands

Use the existing profiles and global/native permissions for starting, changing
and stopping expeditions. Location-only access does not imply travel authority.
The panel should show the authorized boundary, return anchor, known/observed/
visited coverage, frontier choices, current leg, failed approaches, discoveries,
return allowance and the reason for turning back or needing a plan. Planned
lines, travelled segments and uncertain routes need distinct presentation.

A useful eventual request is “Tas, explore around the meadow camp and return.”
Questions such as “what did you discover?” and “how much remains unknown?” are
read-only. These are proposed command semantics, not commands supported today.

## Rollout and evidence

1. Establish explored/observed/visited coverage and route provenance first.
2. Validate frontier selection and blocked-approach memory using recorded scenes.
3. Implement preparation, one travel/survey leg and return, with supplied gear.
4. Expose a typed persistent exploration goal and earn its return/discovery proof.
5. Add panel controls/overlays and a bounded autonomous benchmark with cleanup.
6. Broaden to requested resources/biomes, planned resupply and larger expeditions.
   Boats, portals, camp building and cross-biome survival each need separate work.

Replay scenarios include repeated/stale frames, shared-map-only knowledge,
incomplete scans, stacked floors, water/cliffs, unavailable supplies, declining
health, blocked or changed return routes, no useful frontier, budget exhaustion,
checkpoint failure, uncertain restart, world changes, stop and owner takeover.
Verify that failures lead to a materially different allowed choice or an honest
partial/needs-plan result, rather than an unbounded identical retry.

The first live benchmark needs no staged enemy or player. It starts at an
observed safe anchor, selects naturally available nearby frontier ground,
records new evidence and verifies supported arrival on return. Lack of a safe
frontier is a valid bounded diagnostic, not discovery success. Keep full traces
in retained evidence and short summaries on tickets; leave the panel available
through test cleanup as in the existing supervisors.

## Draft work breakdown

The existing coverage ticket stays in its current parent; the new epic reuses
it through dependencies rather than creating another coverage implementation.
All new work below explicitly depends on both M2 and M3 and remains draft:

- TKT-01M66PMAPGXNJHB49E7WMGEBYS — Select exploration frontiers and remember failed approaches.
- TKT-01M66PMAQSBK2TH076X7M5FGAM — Add bounded expedition preparation, survey and return routines.
- TKT-01M66PMASYEM3Z9KJMB2979CEM — Compose persistent exploration goals with discovery and return proof.
- TKT-01M66PMAV9P2Z8QMR9N90T0TH1 — Show and control exploration progress in the panel.
