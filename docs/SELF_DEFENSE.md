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
which currently selects melee). This build accepts only one unstarred hostile
Greyling, within 12 m, starting at 80% health or higher. Players, tame creatures,
other species, ranged style and unusable weapons are refused. Combat and
movement permissions are both required.

The task chooses a carried melee weapon using character damage rather than
harvest damage. Normal game controls supply aiming, swings, blocks and dodges.
Attack, block and dodge costs use the installed game's methods/modifiers;
blocking reserves the conservative full stamina drain, not an assumed parry.
Attacks preserve stamina for defense plus 10 for escape. Dodges require a new
observed attack window and a supported, clear 5 m corridor. A missing roll
corridor selects block when affordable. No timed-parry guarantee is made.

An encounter has a 12 m leash, 45 s work limit and 12 s no-progress limit.
Low health/stamina, lost equipment, another nearby creature or unsafe scenery
selects a bounded ordinary flee. That escape stays part of the attack task;
its final result reports the failed encounter and the escape outcome. The
controller recognizes matching combat/task IDs and lets healthy melee or an
active fallback flee run, while stale/unsafe attack state still allows retreat.
Only observed target death succeeds; unloading reports target_lost. Stop,
replacement, timeout and session cancellation clear held inputs and pending
attack/dodge queues. An already-started swing or roll follows normal animation.

The panel shows stamina and combat mode/reason. Use `just vm combat` to read
the plugin gate; changing it requires a game restart. An owner must separately
allow attack in controller settings. No autonomous attacker selection, job
resumption, ranged combat or owner-guard behavior is installed by this slice.

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

The first live check is one staged unstarred Greyling in open ground, with
food and a usable weapon, an owner ready to clear the encounter, and exact
task/audit evidence. Verify defending, then disengagement and return to shelter;
do not deliberately injure Tas to cross a health threshold. Record incomplete
game checks on the tickets. Use just check for retained, bounded diagnostics;
just check ci and committed just check review remain the landing gates.

Tracking: TKT-01M62N25826TRP761PYVVC7RKY (Plugin: attack task with blocking
and dodging), TKT-01M62N25ADM0Z968D974G0CHP2 (Controller: guard the owner
and defend itself), and TKT-01M62N256MPAJ6YJC11XJS09RS (Plugin: flee task,
and retreating when health is low).
