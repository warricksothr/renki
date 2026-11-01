# World events and group encounters

Tas observes Valheim's `RandEventSystem` instead of guessing that a crowd is a
raid. State, panel status and the shared map expose `worldEvents`:

- Missing means unavailable knowledge, including offline, old plugins, menu,
  malformed geometry or a state stream older than three seconds.
- `{}` is a fresh observation of no random or local event. It says nothing
  about surviving monsters.
- `randomEvent` carries game identity, localized notice, centre, radius,
  observed elapsed/duration seconds, possible empty-area timer pause and Tas's
  geometric membership. Membership matches the game's strict horizontal
  radius and excludes positions above Y3000.
- `localEvent` describes the event currently affecting Tas. Its kind is
  `random` or `local`. Forced, boss and zone notices get no invented bounds.

The danger layer draws the random-event boundary, includes it in hover and
exports the same knowledge for diagnostic tools. Current events do not persist
in saved terrain or survive disconnect as live knowledge. State audit samples
retain observations. Elapsed time is not a reliable end countdown: the server
can pause the timer while nobody occupies the area. Logout need not expire it.

Creature `eventCreature` reports event-spawn provenance, not membership in the
current event. `targetingKnown` is true only when this peer owns the creature's
simulation. Then `targetingMe` describes its target and `huntingPlayers` its
effective player-hunt mode. Remote target absence remains unknown. Hunt mode
can sustain pursuit beyond ordinary sensing/distance assumptions. These fields
are observations, not combat authorization.

## Household response implemented

The pantry adapter settles pending native actions and inventory receipts first.
Existing injury and nearby-hostile guards retain priority. With unavailable
event knowledge, a local event, or a random event covering home or Tas,
household work defers only at actual home shelter with no known pursuer. Tas
may eat carried food there. Exposed positions need an escape plan; no return
through an event or group fight is commanded by this change.

Waiting shares the existing 240-second aggregate observation/recovery budget
and goal deadline, checking every two seconds. Repeated calls before wake do
not spend another step. Fresh observed clear allows reassessment, including
lingering-hostile checks. Separate personal resupply remains available. Event
bounds are not yet general native route exclusions: a distant area covering
neither home nor Tas does not block household work.

## Native peeling design: next combat increment

Group size should drive preparation and escape decisions rather than imply
combat is permanently impossible. Current native melee still supports only
ordinary Greylings, Boars and Necks under single-target safety gates.
**Greydwarf combat and group peeling are not implemented by this change.**

A persistent goal should request one bounded native peel episode. Movement,
pursuit sampling and emergency disengagement belong in C#, not controller
puppeteering. The episode follows these phases:

1. **Prepare:** verify supported enemies, weapon/armour durability, food,
   health, attack/defence/escape stamina and loot capacity. Survey gently graded
   fighting ground and an escape corridor away from bystanders, buildings and
   water. Account for event bounds and continuous spawning.
2. **Disengage and select:** track the whole group by stable entity IDs. A
   group already targeting Tas first requires escape and pursuit assessment.
   Choose a peripheral target using exposure and likely separation. Being
   nearest does not prove isolation.
3. **Measure pursuit:** accumulate movement and authoritative target evidence
   for every pursuer. Ownership loss, lost visibility or unloading means
   uncertainty, not disengagement. Event creatures in player-hunt mode may
   follow the entire route. Bound escape/reposition attempts when no separation
   forms.
4. **Confirm isolation:** require a quiet interval with exactly one eligible
   pursuer near fighting ground, competitors outside an approach-time margin,
   and no unaccounted group member able to arrive during combat. Recheck escape
   and bystanders. A second pursuer, spawn, lost observation or low readiness
   cancels isolation. Distances and the interval need measured calibration.
5. **Fight one and reassess:** hand off to bounded melee, require observed
   death, and escape if enemies converge. Then remeasure the group, health,
   stamina, food, durability and loot pressure. Recover/resupply or request
   another episode. Do not queue nine attacks.

Expose phase, target, tracked pursuers, unknown/lost members, hunt mode,
isolation evidence, escape destination and refusal reason in status/traces.
Goal checkpoints retain completed kills and remaining work. Never replay an
uncertain attack or retain isolation across task/session boundaries.

Deterministic checks must cover one follower separating, the whole pack
following, a delayed second pursuer, ownership loss, continuous spawning,
timer pause, blocked escape, low stamina and verified kill followed by resupply.
Field checks progress from one supported creature to two and then larger mixed
groups after preparation and enemy-specific combat pass. Find natural enemies
through bounded native search; manual staging/rescue does not earn a pass.

## Evidence and remaining checks

Shared Go/C# fixtures cover random, clear and local-only events, creature
provenance and remote target uncertainty. Geometry tests cover boundary,
altitude and malformed inputs. Controller checks cover scope, freshness,
bounded shelter waits, lingering/pursuing threats and receipt settlement.
The panel `world-events` UI scenario uses a simulated event and checks offline
export removes current-event knowledge.

The nine attackers in `.checks/food-20261101T111255Z/before-map.json` were six
Greydwarfs and three Greylings targeting Tas. That older capture has no event
metadata and cannot establish a raid. Native event start/end observations,
inside/outside membership, paused timer and ownership transfer still require
field verification. These checks do not include logging into that crowd.
