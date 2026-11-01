# Work trips through ordinary danger

A work goal should retain its objective and verified progress while Tas
handles an encounter. The controller owns the durable job and its receipts;
C# owns immediate walking, escape and combat under the current body task.
Human commands, reconnects and world changes revoke that continuation.

## Staged implementation

1. **Recover travel interruptions.** Opted-in dry travel can pause, escape on
   checked ground, observe a quiet period and replan to the original target.
   A distant targeting flag alone does not terminate the household episode.
   Keep one task identity and its existing deadline throughout.
2. **Recover work interruptions.** Extend the same policy to gathering,
   storage and cooking. Reconcile any in-flight product/transfer receipt
   before deciding what to resume. Replace the blanket any-injury abort with
   explicit damage/reserve budgets backed by encounter tests. Prioritize
   recovery and storage when nourishment, equipment, capacity or health drops.
3. **Plan with exposure and escape options.** Separate physical constraints
   from assessed danger. Rank routes by creature capability, numbers, actual
   pursuit, expected exposure and available retreat paths. Require readiness
   for any accepted exposure; stale knowledge cannot establish a clear route.
   Remember failed approaches with geometry/threat revisions and expiry.
   Reconsider a patch or route when observations change, without blind retries.
4. **Defend when capable.** Reuse the existing melee/escort policies and
   explicitly validate ordinary Necks and Boars. Preserve player/tame exclusion,
   structure protection, combat permissions, stamina reserves and death proof.
   A new fight cannot compete with an active escape or another body owner.

Travel now includes a narrow defensive slice and a measured damage allowance
for food goals. Stationary gathering, storage and cooking still need encounter
continuation. Hostile route clearance is unchanged, and the retained seven-threat
coastal bottleneck still has no complete physical route proof.

## First native slice

`move_to` can opt into `recoverThreats` together with `dry` and
`avoidHostiles`. The native task samples loaded creatures at 4 Hz, excluding
players, tamed creatures and passive AnimalAI. Close hostiles or a nearby
engaged creature can trigger escape; proximity never grants combat permission.
Loaded threat sampling covers 64 m to match route clearance observations.

One task owns its walking and escape children. It allows at most three
recovery attempts, each bounded to 30 seconds including observation, and
requires five continuous quiet seconds before resuming. Escape targets 35 m
clearance, beyond the 30 m known-pursuit trigger. A fresh threat during
observation consumes another attempt. Escape uses current physical support,
water, obstacle and hostile-corridor checks. Resumption reconstructs the route
from the actual position and revalidates the original target. The enclosing
task deadline is never extended.

A refused route can instead hold for changed conditions under that same
attempt budget. It waits at least five quiet seconds and replans only after
an observed change to local water level, nearby hostile positions, indexed
geometry or the body's position. Samples run at 4 Hz with coarse spatial
quantization; a change requests another physical search, never movement
permission. Thirty unchanged seconds, unsafe reserves/footing or exhausted
lifetime attempts produce a failure that retains the original route reason.
The dry shore fallback also validates each half-metre step through the same
support, water, body and hostile checks before steering toward a shore hint.

Unsafe reserves/footing, a failed escape or an exhausted budget return a
precise failure. Cancellation clears child inputs. Mode transitions report
the reason, threat identity and attempt count in traces. A matching native
travel task may handle close threats without the food guard cancelling
the entire goal. Missing ownership/freshness never grants an exemption.

## Travel defense and food progress

`defendThreats:true` additionally requires `recoverThreats` and attack permission
from both plugin safety and the controller session. Escape alone grants no combat
authority. Personal resupply and household food travel both opt into fed recovery;
defense is requested only when the session permits attack.

The initial automatic capability is one unstarred hostile Greyling within16m,
confirmed targeting Tas or recently observed attacking him. No second loaded
hostile within30m may be present. Admission requires food, a usable melee weapon,
at least2 armour,40 stamina,30HP and80% health, on grounded dry unencumbered footing.
This reuses ordinary melee's live approach, bystander, structure and strike checks;
it does not establish automatic Neck, Boar, starred or group capability.

Defense lasts at most12s, preserving the remainder of the same30s attempt for
withdrawal/observation. A new threat, changed victim evidence, failed stance,
lost target, insufficient reserves or melee retreat cancels that child and uses
checked escape. Neither transition resets the original task deadline or lifetime
three-attempt budget. Only exact observed Greyling death permits quiet observation
before replanning the original destination. Disappearance is not death proof.
`state.journey` reports the owning task ID, mode, attempt count and reason. The
controller's emergency retreat loop yields to that matching native owner while
reserves remain safe; stale, refused or unrelated metadata does not suppress it.

Food goals retain a single cumulative observed damage allowance of10HP from
their first admitted fed travel, requiring at least30HP,65% current maximum
health and no more than10HP below initial travel health. Session injury metadata
reports cumulative accepted damage independently of healing. The injury sequence
is a wire event marker, not a hit count. Unknown damage amounts or death invalidate
the allowance; reconnect clears it and revokes the old body lease. The original
goal baseline persists across travel tasks. Permission to absorb damage is limited
to the exact running native travel intent and reported journey identity.

At a completed body boundary, reconcile the exact native action and inventory,
container or cooking receipt before applying threat/injury guards to further work.
This preserves verified progress without replaying an uncertain transfer or
inventing a collection. Only verified owned travel can acknowledge bounded injury
for the next step. Stationary injury still ends the run after receipt accounting.
The food supervisor checks the same budget/identity and accepts a completed travel
between polls only with its prior intent, same-run successful history and the
controller's acknowledged injury marker.

## Validation and remaining work

Deterministic checks must cover escape/quiet/resume, renewed pursuit,
exhausted attempts/time, unsafe reserves, cancellation/ownership changes,
protocol opt-in and unchanged production receipts. Live evidence must show
the same native task identity across recovery and actual subsequent progress.
An escape receipt alone is not home arrival or production proof.

The field benchmark is a real encounter followed by resumed collection and
verified storage. A sheltered local return passed on c9c084e; the subsequent
personal food trip stopped on a Neck, and a later mixed-threat return had no
checked initial heading. Tas remains offline there without carried meals.
Before another live work trip, readiness and a physically checked exit or a
validated mixed-threat capability still need resolution. Tests
must retain failures and distinguish recoverable encounters from emergencies;
the current slice does not prove live combat/resumed production or interrupted
stationary-work resumption.
