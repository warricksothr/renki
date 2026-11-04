# Handoff

Replace this page at each handoff; tickets retain decisions and evidence.
Current work of 2026-11-04 by `agent:codex/portal-deploy`.

## Objective and owner decisions

Bring Tas to saved home [-204.39,32.97,-467.43], claim an observed free bed,
and map the area and individual storage access. The new base has vertically
stacked chests/barrels, complex navigation and double doors. The owner said
the old-base SpawnCamp portal leads here and requested OBS/private streaming
alongside Tas. Keep Loot disabled and the game online. Automatic logout remains
disabled. No other agent is active, but preserve older ticket claims.

## Runtime and current position

PRs #176/#177/#178 are merged; #178 is unwatched. This branch
`t3/renki-portal-travel-startup` builds on main501ca8d. Plugin/controller bdd70e7
was installed after full CI and committed review; OBS recovery correction4cc25b8
and survey controller83a4743 also passed full CI and committed review.
Controller-only deployment83a4743 retained Valheim, passed preflight/checksums
and verified the deployment receipt. Latest gate:222s,
.checks/20261104T165535Z-FOYm2i.log. Review: no findings,
.checks/20261104T165928Z-Ml5zXw.log. Deployment:
.checks/20261104T165942Z-50whAH.log. Do not redeploy unchanged code.

At17:04Z Tasleroch is connected to DizzysHellheim, UID827096491, idle at
[-203.11,32.97,-467.58], grounded/sheltered,115.7/115.7HP,injurySeq0.
Three active meals: Sausages, BoarJerky, CarrotSoup; carried reserves include
sausage/jerky, but no soup remains. .checks/final-home-observe.json retains
status/inspection; .checks/final-home-behaviours.json verifies Loot disabled.
Home rules remain idleMinutes0,afterDeath stay,afterOutage false after the earlier
unsupplied old-base death. Do not restore autonomous travel without safe staging.
VM console session2 works. Lease is released at handback; take/renew it before
live work as agent:codex/portal-deploy. Keep Tas online at handback.

## OBS delivered

TKT-01M6KNVKQZ — Start private OBS streaming alongside Tas is done.
`just check vm online` restores panel, existing OBS/relay/capture/watchdog tasks,
verifies active private output, then starts Tas. Repeated startup retains output
and fresh verified scene. `just check streaming windows-online` restores video
independently; `just check streaming windows-adapter-update` replaces only the
adapter binary without scenes/credential reconfiguration. Keep the desktop signed
in; after RDP use C:\renki\release-rdp.ps1. See docs/WINDOWS-VIDEO.md.

OBS33 unclean-shutdown startup was native-tested and recovered narrowly:
archive stale dedicated run markers only while stopped, retaining crash logs;
restart only the dedicated process whose sole current log message proves the
shutdown dialog and whose control port is closed. Other states time out.
Final independent video startup passed in9s,
.checks/20261104T165401Z-wMi5vr.log, retaining gameplay. Latest17:04Z
GET /api/video reports verified gameplay, scope1:DizzysHellheim:827096491
(.checks/final-home-video.json). Only private local output is configured;
no external broadcast destination. No need to restart an already healthy stream.
Older OBS freshness/standby ticket TKT-01M67DWF retains its separate claim/checks.

## Portal, food and stacked storage evidence

TKT-01M6AZ7D — Traverse known portals with verified arrivals remains in-progress.
Explicit portalTo pins observed native source/exit IDs and keeps tag editing
separate. Bounded dry approach, native cargo/access/link checks and actual loaded
exit arrival checks do not bypass the ordinary trigger guard globally.
Actual outward crossing t-5: SpawnCamp1:166576 ->1:166579, grounded/dry arrival
[-168.53,30.97,-458.92], observedReturnTo1:166576.
.checks/task-pm_oi5en.json proves one outward traversal; durable map route
transitions and repeatable return/restoration remain unearned.

TKT-01M6H2AB — Search nearby or listed chests for requested items is done.
Read-only search checked20/23 selected containers, including upper barrels y32
and lower chests y30.97. Inventory remained exactly empty with the same seq112
before/after. Gaps1:15433,1:15438,1:15414 remain unknown. Upper barrel1:15597
has ready meals;1:15598 has berries/mushrooms. Pukeberries1:15420 were not treated
as sustaining food. Only after the read-only evidence, separate t-30 withdrew
3 Sausages,3 BoarJerky,2 CarrotSoup; food maintenance consumed meals normally.
Evidence .checks/newbase-inventory-before.json,
.checks/newbase-search-inventory-after.json,.checks/newbase-search-final.json,
.checks/newbase-storage-map4.json,.checks/task-v8wtntcx.json.

## Survey and remaining navigation work

TKT-01M6FV6E — Home survey: familiarize with an area and update its places remains
in-progress. Saved selected base new home: radius60m,duration600s,travel800m,
return home and storage new home storage west[-169.13,30.97,-467.75].
Sleeping remains unbound. Four freshly observed beds1:14774,1:15016,1:14773,
loft1:15382 are all owned and none mine. The owner was asked asynchronously to
make a bed available; no bed interaction was attempted. Do not claim another
player's bed or invent a sleeping approach.

At17:18Z the same four owned beds remained freshly observed. The lower-area
bed941891633:991 at[-202,30.5,-467] exists only as an older map marker, not a
fresh entity. The owner thinks they moved it and will place a new bed; keep Tas
sheltered while waiting, then refresh perception before selecting any bed.
Owner-requested TKT-01M6KS61A0T2VCT0RMQ1WF47CK (Build and claim a bed through
native placement) is filed as draft under the construction epic, dependent on
the native building foundation. Filing is not implementation selection.

At17:25Z the owner placed new bed1274003559:168 at[-208.09,32.99,-468.03],
freshly observed owned=false. Native interact t-19 approached it but failed
refused: "The bed is too exposed"; it remains unowned. Tas is sheltered at
[-205.25,32.97,-467.77], uninjured. Receipt .checks/task-f0oetsx5.json and
.checks/bed-claimed-map.json. The owner has pending asynchronous questions to
adjust the bed's shelter and identify the home fire they want lit. Only the
already-lit storage campfire1:15355 around40m away is currently observed; do not
assume it serves the new bed. The installed runtime supports fueling only as
part of cooking; ordinary interact refuses fireplaces. Do not use cooking as
a disguised fire-lighting action or bypass native interaction rules. Standalone
fueling remains drafted on TKT-01M62VXZ4 (Plugin: add fuel and ore to machines
and take their output); no implementation was selected in this turn.

Read-only t-20 inspected adjacent home chest1:15452 successfully, with zero
travel and no transfers; it is empty. Tas also has an empty pack and two meals
after the last soup expired; no carried Wood. Evidence .checks/task-gp6xuik9.json,
.checks/home-bed-container-memory.json,.checks/bed-shelter-wait-status.json.
Fresh observations show the storage double doors now closed; no evidence
attributes their closure to Tas, so automatic cleanup remains unearned.
Do not retry the unchanged exposed bed or bind it as Tas's sleeping place.

First goal38cb6628bc116daff1b9323a returned partial without travel,690/704
terrain samples measured,14 unknown,52 refreshed,17 places added/refreshed,
9 old markers protected. Wet ground under the deck wrongly prevented dispatch
to the bound storage role. Correction83a4743 allows an exact bound role within60m
to reach the existing dry bounded native walker without fabricating floor heights.

Retried goal9efba35d2e5246771e077c4d actually reached storage via t-1:
35.485m travel,final[-170.06,31.04,-467.82]. Return t-2 failed after11.979m at
[-177.89,30.97,-471.19]. The actual failure is wave-sensitive dry clearance:
level deck/support y30.97, water30.155 rising to30.275. WaitingMargin0.9 triggered
bank departure; its stronger1.1m recovery threshold was not regained on the level
deck, and its3m bound refused distance_limit. Further terrain legs refused and
the agent stopped the survey. Retained coverage690/704 measured,598 refreshed,
14 unknown;7 new places,9 protected. Goal is cancelled, not a verified return.
Do not repeat the same unchanged survey or relax water/support/hostile guards.
Next implementation is to handle supported deck travel under changing wave
clearance, then verify both directions with bounded receipts.

Separate supervised ordinary native move t-12 succeeded home and recorded71
route points. It does not earn the survey return or separated-building criterion.
Exact audit .checks/audit-window-IxX5Fs.jsonl; bounded summary
.checks/survey-return-audit-summary.txt; checkpoint .checks/continuation-observe.json;
recovery receipt .checks/task-gecesv1a.json. Native trails are in full audit even
where the goal checkpoint's trail arrays are null. Survey criteria5/7 stay unticked.

TKT-01M6H243 — Close doors Tas opened after passing through the doorway remains
review/released with native criterion4 unearned. Storage leaves1:15335/1:15334
were opened but cleanup ran after Tas was8m away; fresh17:03Z perception confirms
both remain open. Explicit t-14 closed home door1:14823 and fresh perception
verified closed (.checks/task-zg0w2zau.json,.checks/home-after-close-map.json).
This manual home-door closure does not prove automatic double-door restoration.

## Older claims and boundaries

TKT-01M62PRDS (Claim a bed near home, or tell owner none), TKT-01M62N256
(Plugin: flee task, and retreating when health is low) and other earlier tickets
retain their claims. Old-base mushroom foraging ended in death before portal
travel; do not repeat it. Exact old pursuit audit .checks/audit-window-CReSWs.jsonl.
TKT-01M6FT1Z (Follow: go through the portal the leader just used) and TKT-01M6EYEMT
(Loot panel) stay review/released with their owner-in-game criteria unearned.
Unselected passive-gathering/needs-ledger/agent-list drafts remain draft.
Twelve unknown saved goal instances warned on startup; do not delete them.
