# Cooking one carried ingredient

Owners and configured operators can say `Tas cook food` (or use `cook food`
in the panel). Tas selects carried raw boar meat, deer meat or a neck tail,
and a freshly observed, empty cooking station within 12 m in three
dimensions. The order cooks **one input**, watches it and collects its actual
output, including the game's possible cooking bonus. It replaces the current
body order, as other explicit commands do. The separate [food goal](FOOD.md)
composes this primitive with storage resupply and escort maintenance.

The low-level equivalent is `interact` with
`{"entity":"<current station id>","cook":"RawMeat"}`; `DeerMeat` and `NeckTail`
are also accepted. Allow a 90 s timeout for an ordinary nearby station. `cook`
cannot combine with sleep, repair, alternate use or hold. Native `AllowGather`
and `AllowMovement`, and session `interact` and `pickup` permissions, are required.
Plain interaction with a cooking station remains refused.

This first slice supports ordinary fire-backed stations with direct item use,
not ovens, cauldrons or stations using fuel switches. Remote station simulation
is supported through the game's ordinary owner-mediated interactions. An unowned
station may be acquired by normal ingredient insertion; the task never claims
ownership itself or displaces another peer's simulation. Native cooking checks the actual
recipe, ward access, sight/reach, active food, 80% health, dry footing, nearby
hostiles and space/weight for the complete possible yield. Missing ingredients,
busy stations and a dead fire produce a refusal with the missing prerequisite.
Station and fire reach use the player's configured normal interaction range
(3.5 m on Tas in the live check; 5 m in the source defaults) and retain solid obstruction checks. A
currently visible/reachable station does not require an extra approach walk.
Otherwise cooking measures up to 48 nearby position seeds, one per physics
tick, including support above and below raised floors. It requires dry support,
standing-body clearance, sight/reach to the station and any fire needing fuel,
and clearance from burning areas (including the unlit supporting fire's bounds).
Only then does ordinary dry, hostile-aware navigation walk to a candidate.
At most four walks, 12 seconds each and one minute total, are permitted.
Actual sight/reach is rechecked after movement; candidate geometry never proves
successful use. Failed approaches and rejection counts appear in cooking traces.
The station needs a working fire, or one carried `Wood` for its unique empty
supporting fire. Before inserting food, the native routine matches the station's
fire-check points against the fire's actual burning-area bounds (including
inactive box/sphere areas), verifies sight/reach/access and uses that wood
normally. It requires the inventory decrease, synchronized fuel increase and
actual station heat before continuing. It never adds a second wood on an
uncertain receipt or to an already fueled but blocked fire. Automatic ingredient
withdrawal is composed by the food goal. Continued fire tending and bulk cooking
remain follow-up work.

The station's normal `UseItem` action consumes one carried ingredient. Success
requires the matching inventory decrease and a single matching cooking slot
whose recorded producer is this peer. Uncertain insertion is not retried.
When that slot is cooked, normal station use removes it once. A scoped observer
records the normal outgoing removal RPC's requested yield and simulating peer.
Locally simulated output has a direct `SpawnItem` witness (`direct_spawn`).
Remote output uses a separate correlation receipt (`remote_correlation`): only
new item IDs absent before removal, created by the addressed peer, born within
three seconds of the request (100 ms clock tolerance) and observed within 2 m
of the normal spawn origin qualify. Prefab, one-item stack and the recorded
crafter, when enabled by the station, must match. Old/reloaded drops, mismatched
creators and merges cannot replace missing output. Excess matching candidates
are ambiguous and fail. The full requested yield and empty synchronized slot
must arrive; remote delivery also waits three seconds before completion to
observe competing candidates. Removal acknowledgment has an eight-second limit.

Remote correlation is evidence of delivery, not a direct witness of the remote
spawn call. Vanilla drops carry no station ID. A simultaneous same-owner,
same-prefab drop at the same origin/time can still be indistinguishable,
especially when the station does not record a crafter. The trace and proof tool
retain the evidence mode rather than claiming exact remote causality. Slow
replication or an early merge may conservatively fail a real delivery.

No inventory or world state is written to create food. Normal pickup of the
identified output instances must produce matched per-drop inventory gains;
remote drops use the game's normal ownership-request RPC and wait up to five
seconds for its grant before manual pickup. This installs no deferred pickup
callback that could continue after cancellation. Pickup does not substitute
another drop after a merge. The task completes only after the corresponding full
inventory increase. An unrelated gain of the same prefab is insufficient. The food
keeper waits to eat this output until the native receipt completes.

A foreign occupied slot, burnt/changed/removed input,
unconfirmed output, unrelated food gains, blocked pickup or work deadline ends
the task without claiming success. State transitions and input/output receipts
appear as `cooking` traces, with the task/station/slot, output IDs and evidence mode.
Other players can still take a batch or its drops; a disappearing item alone
does not prove delivery.

Stop, replacement, timeout, death or bridge loss ends the composition at once.
Already inserted food remains on the station and may burn; already removed
output remains in the world if not collected. Cancellation never performs a
new interaction and reconnect never resumes a partial batch. A future tending
routine must reconcile these partial outcomes before attempting another batch.

Perception exposes optional `fire: {fuel, fuelMax, lit}` and
`cooking: {heated, supported, slots:[{index,item,state,elapsedS}]}` on current
structures. Empty slots are explicit; absent state means unknown. These are
live observations, not cached geometry or remembered guarantees. Cooking state
is `empty`, `cooking`, `done`, `burnt` or `unknown`; elapsed time comes from the
station's synchronized state. Other machine/fermenter/beehive state is pending.

Use `just check plugin test-core CookingTests` and
`just check controller test-packages ./internal/protocol ./internal/chat` for
focused checks; `just check ci` is the full gate. Live proof requires matched
input, slot, direct-spawn or remote-correlation and inventory receipts. Policy tests alone do not
establish successful cooking on the server. Track remaining broader tending
on TKT-01M62VXZ6 — Controller: tend template: cooking, fermenters, beehives, ore.

`just check vm cooking-check` attempts one bounded real batch in an already
joined, healthy game. `--search-home` may obtain one ingredient through matched
native chest/inventory receipts, checking the finite set of unique visible
chests within 32 m of known home; Tas must already be within 32 m of home.
Remembered contents rank candidates but never prove a transfer. Missing hints
lead to another observed chest, with each chest/item pair tried once and the
whole proof limited to ten minutes. `--join` starts once from offline only if
the opt-in native recovery guard is already enabled. The supervisor waits for
its hold flag, avoiding a comparison against rounded recovery thresholds.
Admission requires 80% health and active food. Actual injury, unsafe water,
scope change or the survival floor (20 HP/40% health) aborts. The lower ongoing
fraction permits a newly eaten meal to increase maximum HP before regeneration;
native cooking still requires 80% health throughout its body work. The proof
can first use `--refresh-food` to withdraw one ready meal near known home,
verify the transfer and inventory receipt, and wait for active food with at
least two minutes remaining and recovery to 80% health. Only this bounded
near-home preparation permits absent active food; injury, health, water and
scope checks still apply. It does not relax native cooking admission. The proof
retains full data in `.checks/cooking-*` and prints a compact result. Once
admitted it always logs out/stops and leaves the panel available. It does not
change combat settings, replace an initial body order or resume escort.
Use `just check vm test-cooking-check` for supervisor policy checks.
