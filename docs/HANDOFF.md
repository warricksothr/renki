# Handoff

Replace this page at each handoff; tickets retain decisions and evidence.
Current work of 2026-11-04 by `agent:codex/portal-deploy`.

## Objective and owner decisions

Bring Tas to saved home [-204.39,32.97,-467.43], claim an observed free bed,
and map the area and individual storage access. The base has vertically stacked
chests/barrels, complex navigation and more double doors. The owner says the
old-base SpawnCamp portal leads directly to the new base and cannot supply food
at the old base. The owner also requested starting OBS/private streaming with
Tas. Keep Loot disabled. Keep the game online except required deployment
restarts; automatic logout remains disabled. Do not take other sessions' claims.

## Runtime and live failure

PRs #176/#177/#178 merged; installed runtime is main 501ca8d with verified
preflight/checksums/receipt. #178 is unwatched. The reboot/desktop blocker was
resolved by the owner; interactive session 2 started successfully through
just check vm start (.checks/20261104T155916Z-waASAP.log). Tasleroch is connected
to DizzysHellheim, world UID 827096491. Lease belongs to this worktree as
agent:codex/portal-deploy; renew during work and release at handback.

Four old-base chests were freshly inspected with normal read-only controls.
No ready food was present; pantry 1:75501 contains raw fish, superseding its
historical mushroom contents. Outdoor 1:75525 has trophies/Ectoplasm;
1:75526 and 1:77215 are empty. The northern berry approach was refused due to
Elite clearance. A southern mushroom attempt ended in Tas's death. Greydwarf
2675073813:273 approached while the agent inspected collection code; the agent
should have abandoned the source immediately. Bank retreat t-18/t-20 and
t-23/t-25 failed bearings_exhausted. Hits at 16:12:35.238Z and 16:12:41.296Z,
death at 16:12:41.322Z near [-11.08,30.8,44.77]; no food collected. Owner informed.
Exact audit .checks/audit-window-CReSWs.jsonl; flee and survey tickets retain it.

After respawn at the old bed, automatic home travel was stopped. Persisted
home.idleMinutes=0 and home.afterDeath=stay prevent another unsupplied 500m trip;
afterOutage=false retained. Restore only after safe supplied staging.
RecoveryGuard/AutoLogout=false, combat/gather allowed. Loot, Relocation mushrooms
and Relocation food search disabled. Empty pack, no meals. Two bounded shelter
returns failed without injury. Latest observation at 16:24Z: idle, grounded,
25/25HP at [-43.39,33.73,8.53]; non-targeting Boar ~30m away. Both old double
gates were observed open after these failed returns; earlier closed observations
do not prove restoration. Refresh before acting; do not repeat unsafe foraging.

## Current implementation

Branch t3/renki-portal-travel-startup builds on 501ca8d. Tickets:

- TKT-01M6AZ7DJYGTWDR9G9ZP0MA20M — Traverse known portals with verified arrivals:
  owner-selected, promoted/claimed/in-progress. Adds explicit interact portalTo
  with advertised feature gating, exact-link native walking admission, bounded
  hostile-avoiding dry approach, native cargo/access checks and observed teleport.
  Success requires grounded non-swimming 3D proximity to the loaded expected exit,
  including vertical tolerance. Task/audit receipt separates actual arrival from
  observed return-link metadata. Durable map route planning and repeatable native
  return tests remain unearned.
- TKT-01M6KNVKQZGF26CSJ5T49BH5MV — Start private OBS streaming alongside Tas:
  owner-selected, claimed/in-progress. just vm online restores panel and existing
  OBS/relay/capture/watchdog tasks, verifies private output, then starts the game.
  Stream remains independent of game/controller lifecycle. just streaming
  windows-adapter-update replaces the adapter without reinstalling scenes or
  credentials; windows-online restores installed tasks. Active private output is
  retained on repeat startup. Only loopback private RTMP output is configured.
- TKT-01M6FV6E40SBSMJEHT227XKRK2 — Home survey: familiarize with an area and
  update its places: claimed/in-progress, live criteria 5/7 unchecked. Portal
  transport, new-home food, free bed and bounded survey still need live evidence.

Source portal freshly observed: SpawnCamp 1:166576 at [-50.26,34.09,13.84],
connected to 1:166579; no target coordinates delivered. Owner identifies the
new-base destination. Refresh the exact native link before ordered crossing.
Do not fabricate leader evidence, edit tags, call teleport directly or bypass
ordinary trigger guards globally.

Focused protocol/session/OBS checks and portal/fixture core tests passed.
The complete just check ci gate passed in 178s
(.checks/20261104T162852Z-rZ5dey.log); commit then just check review before
push/deployment. Deployment requires clean committed source. Native startup,
repeat stream startup, portal crossing and new-home work are pending.

## Remaining checks and older claims

Follow-portal ticket TKT-01M6FT1ZVW remains review/released: live leader round trip
criterion 5 is unearned. Loot panel ticket TKT-01M6EYEMTX remains review/released:
owner-in-game reach-over-time criterion unearned. Loot stays disabled.

TKT-01M62PRDS (Claim a bed near home, or tell owner none) and
TKT-01M62N256 (Plugin: flee task, and retreating when health is low) retain prior
session claims. Four new-base beds were historically owned; inspect again and
never take another player's bed. Native search-chests and double-door tickets
remain review with live criteria; a final closed state alone does not prove both
leaves were restored. OBS freshness/standby ticket TKT-01M67DWF retains its prior
claim and separate native checks. Do not infer those checks from active output.

Earlier deployment-preflight/activity tracking tickets are done. Unselected
passive gathering/needs-ledger/agent-list drafts stay draft. Twelve unknown saved
goal instances warned on startup; do not delete without owner direction. Prior
in-progress claims remain owned; no other agent is actively using the VM.
