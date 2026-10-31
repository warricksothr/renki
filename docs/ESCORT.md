# Escorting a player

`Tas, escort me` or `Tas, guard me` starts one native, persistent `escort`
task. Ordinary `follow me` retains its existing behavior. `stop`, `wait here`,
`stay out of it`, another body order, logout or connection loss cancels escort.
It does not restart after reconnect or death.

The task follows the speaker at 3.5 m, matching pace. It uses dry, hostile-aware
travel and the same wall, doorway, upper-floor and fallen-log navigation checks
as other safe travel. An invisible leader is waited for until they enter loaded range; the task
remains active until stopped/replaced or the session ends. Threat observation
and escape continue while waiting. It does not infer a distant location.
Failed following is retried with a short pause; there is no blind walk toward
an old player sighting.

The protected group is the leader and the configured owners/operators captured
when the command starts. Changes to that group take effect on the next escort
command. Companions and guests are not automatically protected. Nearby means
within 12 m of the leader, with the threatening creature within 16 m of Tas.
The low-level task API accepts `protect` explicitly for trusted tooling; chat
constructs it from the same configured commanders that may issue orders.

## Collection detours

After the leader holds position for two seconds, Tas may collect ordinary
materials within 5 m of himself and 6 m of the leader. Wood, stone, flint,
berries, mushrooms and ordinary animal/creature material drops are supported.
He checks ward access, room for the complete yield and a 10% carry-weight
reserve. Known random/extra-yield or theft-provoking pickables are excluded.
Chests, tombstones, valuables, tools, mining and tree felling are excluded.

A detour yields to danger, leader movement, loss of sight, capacity or a 12 s
limit. Failed/contested targets have a 60 s per-entity retry delay in a bounded
cache. The existing pickup/interaction task reports the actual outcome; an
item disappearing is not proof that Tas received it. Once the child ends he
returns to following, without another controller job taking ownership.

## Defense and escape

Proximity or `hostile=true` alone never starts a fight. Tas needs an observed
AI target identifying a protected player, or a recent actual hit against
himself. Valheim's synchronized AI flag says only that a creature has a target;
it does **not** identify the victim. A remotely owned attacker can therefore
remain unconfirmed. An unknown target within 6 m may justify escape, never an
attack on behalf of an inferred victim.

He fights at most one eligible unstarred Greyling, wild Boar or Neck, with at
least 30 HP, 80% health, 35 stamina, active food, armour and a usable melee
weapon. These are conservative admission limits, not a claim that every such
fight is winnable. Existing melee checks, survival reserves, pursuit/work limits,
block/dodge, death observation and escape remain authoritative. Loss of the
leader, a second threat, unsafe footing, no progress or insufficient reserves
chooses escape and then attempts to rejoin. After an encounter there is a
15 s combat retry delay. Tas never attacks players or tame creatures.

Only named protected players whose PvP protection is known may be exempted
from the melee bystander guard. Unknown/PvP-vulnerable players, other creatures,
structures and nearby falling/rolling tree hazards still prevent swings.

The plugin's `AllowCombat` and `AllowGather` gates remain authoritative. The
session must allow `escort`, `follow`, `flee`, `attack`, `pickup` and `interact` for the default
routine. Setting `defend:false` or `collect:false` in the task args disables the
corresponding feature and its child permission requirement. A wrapper never
grants a withheld capability. Emergency controller retreat can still replace
unsafe combat; a matching native escort escape is allowed to finish.

## Diagnostics and validation

`state.escort` reports the task ID, leader, mode, entity and reason. The panel's
status chip and agent status API show this same state. Phase transitions emit
`escort` traces. `state.combat` continues to report the nested melee encounter
under the same body task ID. Controller arbitration checks both IDs and target
identity before allowing that encounter to own retreat.

Use `just check plugin test-core EscortTests` for admission, leash, collection,
PvP, nested-permission and runner cancellation regressions. Protocol fixtures
round-trip through both implementations. `just check controller test-packages
./internal/protocol ./internal/chat ./internal/retreat ./internal/app` covers
commands/status and exact-task retreat arbitration. `just check ci` is the full
gate. Keep live follow, collection, owner-defense and escape evidence on
TKT-01M62N25A — Controller: guard the owner and defend itself. Automated policy
checks do not establish live combat success.

`just vm escort-check Nimah --seconds 90 --leave-running` starts a permitted
escort in an already joined, recovered game and retains observations under
`.checks/escort-*`. Without `--leave-running`, it stops only its own task after
the observation window. It never replaces a newer human order during cleanup.
It logs out on a measured survival floor (below 15 HP or 40% health), and does
not claim combat success merely because following stayed active. It does not
start the game or enable combat/gather permissions.

For a critically injured saved character, `just vm recovery true` enables an
opt-in native startup guard on the next game launch. It holds body inputs even
before the bridge connects, rejects body orders rather than queuing them,
permits instant food use, and checks nearby hostile
creatures and injury each physics step and rendered update. After two seconds
of initialization, recovery requires shelter, dry grounded footing and active
food; an unsafe condition logs out and suppresses automatic rejoining. It
releases once health reaches at least 25 HP and 95% of the current maximum.
It runs only once per plugin lifetime, remains off by default, and does not
govern ordinary combat after recovery. `just vm recovery false` disables it on
the next launch. Keep an external health observer running throughout a live
recovery check and retain the outcome; these policy checks do not establish
that a particular saved location is safe to rejoin.
