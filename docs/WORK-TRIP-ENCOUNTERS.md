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

The initial implementation is stage 1 only. It does not enable fighting,
change hostile clearance radii, permit injury, or establish that the retained
seven-threat coastal bottleneck has a physically viable route.

## First native slice

`move_to` can opt into `recoverThreats` together with `dry` and
`avoidHostiles`. The native task samples loaded creatures at 4 Hz, excluding
players, tamed creatures and passive AnimalAI. Close hostiles or a nearby
engaged creature can trigger escape; proximity never grants combat permission.

One task owns its walking and escape children. It allows at most three
recovery attempts, each bounded to 30 seconds including observation, and
requires five continuous quiet seconds before resuming. Escape targets 35 m
clearance, beyond the 30 m known-pursuit trigger. A fresh threat during
observation consumes another attempt. Escape uses current physical support,
water, obstacle and hostile-corridor checks. Resumption reconstructs the route
from the actual position and revalidates the original target. The enclosing
task deadline is never extended.

Unsafe reserves/footing, a failed escape or an exhausted budget return a
precise failure. Cancellation clears child inputs. Mode transitions report
the reason, threat identity and attempt count in traces. A matching native
travel task may handle close threats without the household guard cancelling
the entire goal. Stationary work and actual injury retain their existing
guards until stage 2; missing ownership/freshness never grants an exemption.

## Validation and remaining work

Deterministic checks must cover escape/quiet/resume, renewed pursuit,
exhausted attempts/time, unsafe reserves, cancellation/ownership changes,
protocol opt-in and unchanged production receipts. Live evidence must show
the same native task identity across recovery and actual subsequent progress.
An escape receipt alone is not home arrival or production proof.

The field benchmark is a real encounter followed by resumed collection and
verified storage. Before that can run from Tas's current offline location,
the recorded coastal return still needs a physically checked route. Tests
must retain failures and distinguish recoverable encounters from emergencies;
later stages will update the current supervisor's blanket injury/logout rules.
