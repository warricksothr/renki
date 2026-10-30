# Self-defense development

Targeted stump harvesting now completes from raised home through collection,
delivery and return. Its aiming, contact checks, normal attack input and
cancellation provide useful foundations for combat. A moving creature also
requires facing, blocking, stamina management and a decision to disengage.

The first explicit bounded melee task is implemented. Its staged game check
is pending; combat remains disabled by default. Automatic self-defense and
owner guarding below are subsequent work.

## Current attack task

An explicit `attack` accepts `entity` and optional `style` (`melee` or `auto`,
which currently selects melee). Native preparation is available with `kite=true`
and optional preferred `at` ground. This build accepts only one unstarred hostile
Greyling, within 30 m, starting at 80% health or higher. Players, tame creatures,
other species, ranged style and unusable weapons are refused. Combat and
movement permissions are both required.

The task chooses a carried melee weapon using character damage rather than
harvest damage. Normal game controls supply aiming, swings, blocks and dodges.
Attack, block and dodge costs use the installed game's methods/modifiers;
blocking reserves the conservative full stamina drain, not an assumed parry.
Attacks preserve stamina for defense plus 10 for escape. Dodges require a new
observed attack window and a supported, clear 5 m corridor. A missing roll
corridor selects block when affordable. No timed-parry guarantee is made.
Ground need not be flat: dodge corridors follow support every 0.5 m, allowing
up to 30 degrees of inclination. A change between adjacent segments is sharp
when it exceeds 15 degrees and its height deviates by more than 0.15 m from
the previous segment's continuation. This accepts mild terrain mesh joins
while rejecting abrupt ledges and ridges. Dry footing and body clearance
are still required.

An explicit target outside 12 m first enters `closing`: supported walking
within 30 m of the starting position, at most 12 seconds, with at least
0.5 m of distance progress every 4 seconds. Health, stamina and nearby-threat
checks apply throughout. When the target first comes within 12 m, the bot's
position anchors the fighting leash and starts the encounter budgets.
An encounter has a 12 m leash, 45 s work limit and 12 s no-progress limit.
Low health/stamina, lost equipment, another nearby creature or unsafe scenery
selects a bounded ordinary flee. That escape stays part of the attack task;
its final result reports the failed encounter and the escape outcome. The
controller recognizes matching combat/task IDs and lets healthy closing, melee or an
active fallback flee run, while stale/unsafe attack state still allows retreat.
Only observed target death succeeds; unloading reports target_lost. Stop,
replacement, timeout and session cancellation clear held inputs and pending
attack/dodge queues. An already-started swing or roll follows normal animation.

The panel shows stamina and combat mode/reason. Use `just vm combat` to read
the plugin gate; changing it requires a game restart. An owner must separately
allow attack in controller settings. Local target acquisition runs only inside
an explicitly requested native goal. Automatic self-defense, job resumption,
ranged combat and owner guarding remain future work.

## What is already available

- State includes health, stamina, equipped items and attacking/blocking flags.
- Perception identifies creatures, tame status, health fraction and stars.
- Damage events identify Tas's actual attacker. Retreat already uses those
  events when a remotely simulated creature's target is unavailable.
- Normal movement, aiming and held primary attack are wired through Driver.
- The task runner cancels replaced work and enforces the dead-man rule.
- AllowCombat defaults off. Explicit melee has matching Go/C# protocol
  arguments, plugin implementation and controller dispatch support.

Two existing behaviors matter for the design. Retreat now flees on a confirmed
hit at any health, so an independent attack loop would be canceled repeatedly.
The explicit melee integration now coordinates matching active encounters;
automatic fight selection still needs the shared policy described below.
Also, the game can report wildlife such as Deer as hostile. Hostility alone
must never start autonomous combat.

## First slice: one nearby attacker

Implement the plugin attack task before enabling automatic self-defense.
Start with ordinary melee against one unstarred Greyling, close to Tas, with
fresh evidence that it is targeting or recently damaged him. A nearby creature
that is merely hostile is insufficient. Keep players and tamed creatures
excluded regardless of PvP settings.

Use a usable carried melee weapon and its actual combat damage, attack range,
durability and stamina costs. Harvest damage is not a measure of weapon
effectiveness against creatures. Do not estimate victory from health fraction
and stars alone: they do not describe a creature's absolute strength.

At each physics step, face the moving target, approach a supported strike
stance, and use normal game attack/block controls. Spend stamina only while
preserving a reserve for defense and escape. Block observable attack windows;
dodge only with known timing, enough stamina and a supported escape direction.
When timing cannot be observed reliably for a remote creature, do not invent
an attack animation or promise a timed parry. Verify this behavior with the
installed game's methods and multiplayer observations before enabling it.

Bound pursuit to the encounter's starting area and time budget. Recheck target
identity, hostility, tame state, nearby characters and structures before
swinging. PvP protection does not protect buildings from incidental swings.
Unloaded or unseen targets are lost, not proven dead. Confirm success from a
live death observation and clear all held inputs on every exit path.

## One fight-or-retreat decision

Coordinate defense with the existing retreat watcher and body ownership.
Emergency retreat takes precedence over combat; combat interrupts harvesting
and hauling. Owner stop/stay-out-of-it clears combat intent immediately, so a
remembered hit cannot restart an attack. Stopping attacks must preserve the
existing escape behavior unless the owner explicitly disables that too.

The initial eligibility policy should require high health (start conservatively
at 80%), a known low-risk attacker, a usable weapon, sufficient stamina and a
clear supported stance. Low health, an additional engaged attacker, a stronger
or unknown enemy, broken equipment, unsafe surroundings, or a bounded lack of
progress selects retreat. Test the transitions with recorded observations and
actual equipment costs; tune thresholds from evidence rather than claiming
that these conditions guarantee victory.

Preserve world/connection scoping, fresh-hit retry handling and active-flee
protection. After combat, leave interrupted jobs paused until their target,
route and required items are revalidated. Stale continuations must not resume
after an owner order, reconnect or world change.

## Native kiting and encounter preparation

`attack` with `kite=true` delegates one complete local encounter to C#.
`entity` is optional in this mode; omission selects the nearest eligible single
ordinary Greyling. Optional `at: [x,y,z]` supplies preferred fighting ground
within 12 m. Without it, the routine surveys the starting point and rings at
6 and 12 m. The second pass rotates by 22.5 degrees to cover new centers within
the same 34-check budget; the starting center is retried once. Each candidate
must have dry supported terrain, clear body routes
within a 6 m radius, no nearby buildings or bystanders, and a usable approach.
Four full diameters sample support every 0.5 m, including across the center.
They use the same 30-degree grade and angle/height roughness checks as
dodging, allowing gradual hillsides while rejecting steep ground and sharp
crests, dips or ledges at that sampling resolution. The installed game's
player sliding threshold is 38 degrees; 30 leaves margin for combat.
Terrain projection uses the game's terrain-only raycast, not the nearby mesh
vertex returned by `Heightmap.GetWorldHeight`. Rejected candidates emit
`combat/ground_rejected` traces with their location, attempt and reason. Profile
failures include the exact segment, supported point, current/previous
inclination and height residual. Physical failures distinguish support, body casts/overlaps, cached
bounds, remembered walls and water; collider names/centers accompany live hits.
Detailed physical diagnostics are enabled only for this survey.
It moves to the checked ground before searching. Preparation is capped at
25 seconds; local enemy acquisition waits at most 8 seconds once positioned.

One body task owns survey, position, search, draw, kite, hold, melee and escape.
The world-scoped controller `combat` behavior submits one such goal and waits;
editing/stopping uses the existing engine cancellation rules. The Python trial
runner also submits one goal, without selecting enemies or issuing movement.
No background combat behavior is installed or enabled automatically.

Acquisition is restricted to loaded ordinary hostile Greylings within 30 m
of both the bot and its starting area, with clearance from other characters.
Kiting confirms actual target movement toward Tas on half-second samples;
it does not require a replicated AI targeting flag or force the creature's AI.
It draws toward 8 m separation, leads toward the area once pursuit is observed,
waits if the enemy lags, walks normally and sprints only below 5 m separation
with at least 30 stamina. Below 30 stamina it holds when at least 6 m clear;
below 10, or too close to preserve the reserve, it aborts.

The lure has a 40 second budget and a 40 m travel bound, with explicit loss
of pursuit and progress limits. It routes sideways when the destination lies
past the enemy, checks the actual route and aborts if a close step would cross
into the enemy. Fighting ground is rechecked every second and before melee.
When Tas is within 2 m of its center and the following target is within 6 m,
it hands directly to the existing melee task under the same attack ID.
Only observed target death succeeds. Additional creatures, bystanders,
changed ground, lost equipment, low health/stamina or owner stop end the attempt.
A close-threat abort uses a bounded 30 second flee; a distant lost follower
ends the attempt without starting another long chase. Melee retains its own
existing budgets and fallback rules.

Policy tests cover pacing, pursuit evidence, bounds, cancellation and retreat
coordination. Successful live attraction, kiting and melee handoff remain to
be observed. The isolated test world remains optional later tooling.

## Owner protection and the web view

Owner defense follows self-defense. Current damage events observe Tas, not
every owner's damage, and remote AI targeting is incomplete. Add explicit,
world-scoped attacker/victim evidence where the game exposes it; keep unknown
targets unknown. Only then enable guard-me behavior against creatures actually
attacking the selected owner.

Expose defense mode, current threat, fight/retreat reason, health/stamina and
interrupted job in the web view, with stay-out-of-it and retreat controls. Keep
the plugin AllowCombat gate and the owner-controlled behavior setting separate.
Existing waypoints/zones can later describe preferred refuges or guard areas;
a storage/cooking/harvest zone is not evidence that a location is safe.

Ranged attacks, multiple-enemy tactics, high-tier enemies and proactive owner
guarding follow the verified melee slice. Combat remains off by default while
these foundations are developed.

## Verification

Use deterministic policy tests for unrelated wildlife, known attackers,
healthy-fight/unsafe-retreat transitions, stamina exhaustion, multiple threats,
broken weapons, stale worlds and owner-stop suppression. Exercise controller
arbitration so defense and retreat cannot repeatedly replace each other.

Plugin checks must cover friendly targets, target loss versus confirmed death,
weapon reach, supported dodge movement, disabled permissions, cancellation and
clearing attack/block/dodge inputs. Include dead-man and reconnect behavior.

Use `just vm combat-trial` for one automatic normal-controls encounter when a
safe ordinary Greyling is available nearby. It restarts with combat enabled,
checks food, health and stamina, submits one native kiting goal, records
task/state/perception/audit evidence, and always logs
out and stops the game, restoring the combat config to false. It aborts on
lost body ownership, another nearby creature, health below 70%, escape stamina
below 10 or travel beyond 40 m. It never requests owner clearance. No suitable
encounter produces an explicit aborted result, not a passing combat test.
The runner does not spawn enemies, reset the character, scout unseen terrain,
or prove return-to-shelter. Native kiting above prepares natural encounters
without owner staging. It cannot guarantee that
a suitable enemy is available; isolated-world staging/reset remains optional
tooling for guaranteed fixture availability.
Verify defending and disengagement from actual evidence; do not deliberately
injure Tas to cross a health threshold. Record incomplete game checks on the
tickets. Use just check for retained, bounded diagnostics; just check ci and
committed just check review remain the landing gates.

Tracking: TKT-01M62N25826TRP761PYVVC7RKY (Plugin: attack task with blocking
and dodging), TKT-01M62N25ADM0Z968D974G0CHP2 (Controller: guard the owner
and defend itself), and TKT-01M62N256MPAJ6YJC11XJS09RS (Plugin: flee task,
and retreating when health is low).
