# Handoff

Replace this page at each handoff; tickets retain decisions and evidence.
Current work of 2026-11-04 by `agent:codex/harvest-zones`.

## Owner decision and completed work

The owner reported wood markers in water and requested protection for camp's
wood and stone, choosing the saved **60 m new home boundary**. This is implemented,
deployed and verified on TKT-01M6KXKRTK0CXTWC4KV2YGC5TX (Protect base resources
and reject underwater harvest sites), done/released with all four criteria earned.
Every saved base circle in the current world protects resource roots at all heights.
New home remains center[-204.39,32.97,-467.43],radius60; its definition was not edited.

Both survey and background discovery reject camp wood/stone/ore sites and require
measured dry roots. Static FirTree_oldLog objects are logs rather than standing
wood sites. Invalid automatic resource markers are pruned without suppressing
future safe sites; owner edits and saved base role bindings survive. Gather planning
checks camp exclusion, and every application harvest dispatch resolves a fresh
current-world target and supplies authoritative saved circles. Native harvesting
checks the actual root against those circles and current water before work and
every tick. The advertised harvest_exclusions_v1 feature refuses unsafe fallback
to older plugins. Existing collision, structure/fall, sight and ward guards remain.
The circles are a dispatch-time snapshot: stop active harvesting before changing
a base boundary. No cutting/mining trial was performed or is needed to repeat the
safe refusal checks. Native checks have unit coverage and a real plugin build.

The broader owner goal remains arrival at saved home, bed ownership and mapping
stacked storage/navigation/double doors. Prior arrivals, bed claim, storage survey,
hearth tending, clothing and repairs are retained below with their honest gaps.
Keep Tas and private video online; Loot and automatic logout remain disabled.
No other agent is active. Preserve older ticket claims and unselected drafts.

## Installed runtime, validation and live proof

Branch `t3/renki-portal-travel-startup`,PR179,builds on main501ca8d. Plugin/controller
**e5a7ec2** is installed; later evidence-only commits need no deployment.
Full just check ci passed in186s (.checks/20261104T184737Z-zO9WFA.log), including the
actual Valheim plugin build. Committed just check review 7c67dfa..HEAD had no findings
(.checks/20261104T185106Z-nKwmJQ.log). Initial CI had an existing retreat-test timeout;
five focused race runs passed before the successful complete gate. See the ticket.

just check vm install passed preflight,installed checksums and deployment receipt;
durable data retained (.checks/20261104T185151Z-08PSEM.log). just check vm online
restored panel,existing private output and Tas in68s
(.checks/20261104T185225Z-khRSDb.log). VM console session2 still works; after RDP use
C:\renki\release-rdp.ps1. VM lease was released; take it again as the current actor
before live work. Do not stop the healthy game/private video at handback.

Live comparison removed exactly `new home wood 1` through `6` and
`new home stone 1`. All11 owner places and the saved base definition were unchanged.
No automatic wood/stone/ore marker remains in camp across successive captures.
Both static logs1:11828 and1:14180 now report kind log and measured water above their
Y25.29/26.03 roots. Unobserved global wood/stone markers outside saved bases remain
last-known hints; missing fresh water evidence alone does not delete them.
Evidence .checks/harvest-zones-{installed,final}-map.json and
.checks/harvest-zones-live-summary.json.

Bounded collect=false requests for camp Beech1 1:14179 and Rock_3_frac 1:14273 both
returned HTTP refusal: protected saved base(60 m radius). Their just check vm task
commands intentionally exit nonzero: .checks/20261104T185410Z-2H1WDQ.log and
.checks/20261104T185419Z-2BNsXB.log. Exact audit18:54:05Z..18:54:30Z
.checks/audit-window-H9ZyqE.jsonl contains no outgoing harvest task,only normal
BoarJerky maintenance. This proves controller refusal before the bridge,not native
destructive work. Wood1,Stone4 and axe100/100 are unchanged; injurySeq0.

## Final safe state and food

Latest captured Tasleroch UID827096491 is connected to DizzysHellheim,idle at
[-201.81,32.97,-468.07],near saved home,grounded/sheltered/near_fire,
101.2/101.2HP,injurySeq0. That home position was already observed before installation;
do not attribute it to the refusal tests. Equipped quality1 AxeFlint100/100,
HelmetLeather400/400,ArmorLeatherChest400/400,ArmorLeatherLegs400/400.
Active DeerStew,BoarJerky,QueensJam;carried2 each. Wood1,Stone4.
Private /api/video reports enabled gameplay,scope1:DizzysHellheim:827096491.
Final evidence .checks/harvest-zones-final-{status,inventory,map}.json and
.checks/harvest-zones-installed-video.json. Current runtime and food counts
supersede historical endpoints/counts below. Bed1274003559:489 remains the saved
owned spawn; actual sleep/respawn remains untested.

Food was replenished through ordinary known storage interaction: t-29 reached
the upper food barrel approach; t-30 moved3 BoarJerky and3 QueensJam but could not
supply missing Sausages (.checks/task-gzu65aif.json). Fresh barrel1:15597 contents
showed DeerStew,then t-32 withdrew3 (.checks/task-s33_5u2w.json). t-34 successfully
returned to the eastern warehouse workbench (.checks/task-mpihgx1t.json); a later
fresh capture found Tas near home before installation. Do not invent a cause for
that later position. Fresh food barrel contents remain finite; cached stock is
not a guarantee. No unsafe foraging or camp harvesting was done.

## Clothing, axe and layered warehouse access

Earlier owner operation: west forge1:15324[-165.67,30.31,-473.28],east
workbench1:15887[-158.49,30.30,-476.39]. These are separate from the basic bench
near Tas's bed. t-21 returned to the warehouse. t-22 failed ordinary station use
from[-159.27,30.97,-474.67],approximately2.004m away in3D: the existing approach
accepts planar2m arrival while the game requires strict3D distance below its2m
use radius. Ordinary closer move t-24 reached[-159.00,30.97,-475.07],then t-25
repaired all three leather pieces to400/400 with normal repair-button receipts.
Game station identity/type and minimum-level eligibility checks remain intact;
exact levels are not exposed and were not invented. t-26 re-equipped the full
flint axe after crafting UI use. No injury or runtime change. Leave Tas online
at this sheltered eastern workbench for the owner's next instruction.

This supersedes the worn-armor/end-at-home state in the earlier gear history
below. Receipts .checks/task-w1lvycaq.json,.checks/task-gyvdqj9w.json,
.checks/task-5imt_75g.json,.checks/task-skao5zl2.json,.checks/task-lu4nsmup.json.
Draft TKT-01M6KX627 (Repair: approach inside the station's native use radius)
records the captured mismatch and closer-standing control; do not promote it
without owner selection. There is no source fix yet. Native repair success is
verified, not forge repair, automated station selection or explicit level telemetry.

TKT-01M6KWJMP is done/released. The owner identified a possible flint axe in a
chest by the central doorway. Fresh t-8 inspection found it in1:15433
[-170.71,30.97,-465.33],alongside other gear; cached contents were stale.
This inspection succeeded without moving from[-169.69,31.03,-467.8]. t-9 withdrew
one full AxeFlint and one HelmetLeather. t-12 withdrew one ArmorLeatherChest and
one ArmorLeatherLegs from upper barrel1:15599[-165.1,32,-465]. t-10/13/14/15
successfully equipped each; fresh inventory verified the equipped flags.

Ordinary t-16 returned home. Optional t-17 repair at workbench1:15063 returned
"repaired0/no more repairable gear"; observed durability did not improve. Do not
claim repairs or invent a station-level/recipe cause. At that earlier point armor remained usable but
partly worn, especially trousers236.1/400; the warehouse repair above supersedes it. The normal station interaction
holstered the axe; t-19 returned to the sleeping approach and t-20 re-equipped
AxeFlint. That earlier inventory confirmed all four equipped, axe100/100 and injurySeq0.
No tree cutting, combat trial, crafting or material spending was ordered.
Receipts .checks/task-907sf08y.json,.checks/task-6ohrx01c.json,
.checks/task-sfonphhi.json,.checks/task-o1ck2k4w.json,.checks/task-qx71k4bg.json,
.checks/task-bvuz1tdv.json,.checks/task-9x5dijfh.json,.checks/task-bezmf5ok.json,
.checks/task-exgh0oqr.json,.checks/task-7wfd6f1m.json,.checks/task-2g3wugrq.json.

The owner asked about inventorying two/three layers in the warehouse. Existing
search_chests reads full container contents and updates memory, while the chat
command asks for an item, e.g. "Tas, search nearby chests for flint axe within60m".
Each layer retains its own entity/height. UseReach checks eye-relative3D sight
and reach; InteractionApproach can measure alternate supported standing positions.
Lower chest and upper barrel access have native proof. This does not establish
every third-layer access:1:15414 and1:15438 remain previously unread candidates.
The routine reports failed/unchecked observed candidates and bounds radius,
time,travel and chest count; undiscovered storage remains unknown. Future stack
changes need fresh receipts. Preserve the older equip implementation claim
TKT-01M62N12D (Plugin: equip and unequip items); today's operation did not test
its explicit unequip task or earn its combined criterion.

## Hearth support and exact live evidence

`add_to {entity,item,count}` supports ordinary finite refillable Fireplace fuel,
including hearths,torches,braziers. Count is an explicit integer1–16. The plugin's
new default-off Safety.AllowStations is enabled on this VM, alongside existing
AllowGather/AllowMovement. Existing allowedTasks were preserved with add_to
appended before launch. `just vm stations` reads the flag; changes apply after
restart. No ownership claiming, fire toggling, fireworks or direct inventory/ZDO
writes are used. Each ordinary UseItem is followed by exactly one pack decrement
and a synchronized fuel increase before another use. An uncertain unit stops
without retry; cancellation/reconnect does not replay it. Cooking shares the
single-unit receipt while retaining its empty-fire and heat requirements.

Tending requires ordinary ward access, live sight/reach, food, safe health,
grounded dry footing clear of flames/smoke and no nearby hostile. Approaches are
bounded using existing supported-floor interaction movement. A fueled but unlit
fire refuses more fuel; an empty fire may receive one unit and then stops if
still unlit. Fuel verification does not establish heat or ventilation.

Rebuilt hearth1274003559:2382 at[-203.66,30.37,-470.56]: new-session t-1 rejected
Stone as invalid_target before use. Ordinary t-2 reached the lower room via the
north staircase/door at[-205.54,30.26,-466.87]. t-3 added exactly2 Wood through the
half-wall opening from[-205.54,30.26,-466.89],without extra approach movement:
pack3->2,fuel1.69052184->2.69052172; pack2->1,fuel2.69052172->3.69052172.
Native success: added2/2 Wood,fuel3.691/20,lit=True. Fresh next perception confirms
fuel1.7->3.7/20 and inventory Wood3->1,Stone4 unchanged. InjurySeq remained0.
This proves actual use through the opening and short safe tending, not sustained
smoke safety in every weather state. t-4 returned upstairs without injury.

Exact task .checks/task-v5io_9bo.json; full audit window18:11:50Z..18:12:05Z
.checks/audit-window-VvrnUU.jsonl. Before/after
.checks/hearth-two-wood-{before,after}-{map,inventory}.json and after-status.json.
Negative case .checks/task-xy4fy2om.json; lower-room and return receipts
.checks/task-05kfvpse.json,.checks/task-zbsqagkt.json. No need to repeat the same
successful test or consume more fuel without a new reason.

## Food and storage

Mapped upper food barrel1:15597 at[-165.89,32,-466.76] remains stocked with ready
meals. CarrotSoup was absent on this visit; cached storage is not a guarantee.
Material barrel1:15599 at[-165.1,32,-465] supplied2 Wood. Resupply first triggered
automatic flee from Boar1946562314:4067: t-36 reached safety34m away with injurySeq0.
Subsequent ordinary withdrawals supplied3 Sausages,3 BoarJerky and3 QueensJam;
automatic maintenance ate one each. Ordinary t-44 returned home without injury.
The boar was later observed north of home around[-199.94,30.46,-450.21]. Refresh
threats before another exposed trip; no unsafe foraging or spawned supplies.
Receipts .checks/task-55w4xxwr.json,.checks/task-kxuqqjfa.json,
.checks/task-nm3uajxl.json,.checks/task-_2k3r8b3.json and
.checks/hearth-resupply-flee.json. Four Stone were already carried before this
slice; do not attribute their acquisition to an ordered transfer.

TKT-01M6H2AB (Search nearby or listed chests for requested items) is done.
Its read-only survey inspected20/23 selected containers, including upper barrels
y32 and lower chests y30.97,with identical empty inventory before/after. Original gaps
1:15438,1:15414 remain unknown;1:15433 was successfully refreshed today. Pukeberries1:15420 are not sustaining food.
Evidence .checks/newbase-search-final.json,.checks/newbase-storage-map4.json,
.checks/newbase-inventory-before.json,.checks/newbase-search-inventory-after.json.

## Other delivered work and remaining gaps

TKT-01M6KNVKQZ (Start private OBS streaming alongside Tas) is done.
`just check vm online` restores panel,existing OBS/relay/capture/watchdog tasks,
verifies private output,then starts Tas. `just check streaming windows-online`
restores video independently. Repeated startup retains active output; the pinned
OBS33 unclean-shutdown dialog has a narrow verified recovery. Do not reconfigure
scenes/credentials or restart healthy output. See docs/WINDOWS-VIDEO.md.

TKT-01M6AZ7D (Traverse known portals with verified arrivals) remains in-progress.
Native outward t-5 crossed SpawnCamp1:166576->1:166579 and verified grounded dry
arrival[-168.53,30.97,-458.92],observedReturnTo1:166576. Evidence
.checks/task-pm_oi5en.json. Repeatable round trips and durable atlas portal routes
remain unearned. The earlier t-5 is from a different controller session than the
latest manual door closure.

Tas's new bed1274003559:489 at[-207.83,32.99,-468.26] was claimed natively with
"Spawn point set" after the owner fixed an exposed predecessor. Saved place
`tas bed approach`[-204.89,32.97,-467.49] is bound as new home.sleeping. Four other
beds belong to others and were left alone. Actual sleep and respawn after death
are not tested. Preserve TKT-01M62PRDS's older claim and unticked criterion2.
Owner-requested TKT-01M6KS61A (Build and claim a bed through native placement)
remains draft; filing was not implementation selection.

TKT-01M6FV6E (Home survey: familiarize with an area and update its places) remains
in-progress. Survey9efba35d2e5246771e077c4d reached bound storage via t-1,35.485m,
then its return t-2 failed after11.979m at[-177.89,30.97,-471.19]. Level deck y30.97
under changing water30.155->30.275 triggered dry bank departure at0.9m clearance;
its stronger1.1m recovery threshold was not regained within3m. Do not repeat the
unchanged dry survey or relax water/support/hostile guards. Scope retained
690/704 measured samples,14 unknown; separate supervised ordinary t-12 returned
home and recorded71 route points. This does not earn survey return criteria5/7.
Source correction83a4743 already lets an exact bound role within60m reach the
existing bounded native dry walker when terrain under raised decking is wet.
Audit .checks/audit-window-IxX5Fs.jsonl,.checks/survey-return-audit-summary.txt,
.checks/task-gecesv1a.json. Supported deck/wave handling is a future implementation.

TKT-01M6H243 (Close doors Tas opened after passing through the doorway) remains
review/released with native criterion4 unearned. Manual home-door closure today
is not automatic double-door restoration. TKT-01M6FT1Z (Follow: go through the
portal the leader just used) and TKT-01M6EYEMT (Loot panel) remain review/released
with owner-in-game criteria unearned. Do not take over older flee/bed claims,
promote unselected drafts or delete the twelve unknown saved goals warned at
startup. Next work needs the owner's selection.
