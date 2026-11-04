# Handoff

Replace this page at each handoff; tickets retain decisions and evidence.
Current work of 2026-11-04 by `agent:codex/portal-deploy`.

## Owner decisions and current task

The owner selected native fire interactions for hearths. That fireplace slice is
implemented, deployed and live-tested through the rebuilt half-wall opening.
TKT-01M62VXZ4 (Plugin: add fuel and ore to machines and take their output) is
review/released: criteria2/3/4 earned; both-task coverage and take_from remain
unticked. Do not describe cooking-output collection or other machines as shipped.

The broader objective remains arrival at saved home[-204,-467], claiming a bed
and mapping the new base's stacked storage, navigation and double doors. Tas
arrived through the old-base portal, owns the new bed and has mapped20/23 selected
storage entities. Keep Tas and private video online, Loot disabled and automatic
logout disabled. No other agent is active; preserve older ticket claims.

## Runtime and final safe state

Branch `t3/renki-portal-travel-startup`, PR179, builds on main501ca8d. PRs176/177/178
are merged;178 is unwatched. Plugin/controller1c1d36c is installed. Later evidence
commits do not require another deployment. Full gate passed in227s,
.checks/20261104T180339Z-7nnsAy.log; committed review had no findings,
.checks/20261104T180805Z-6zOnNj.log. Deployment preflight, installed checksums and
receipt passed; durable data retained (.checks/20261104T180836Z-KWYTZz.log).
`just check vm online` restored panel, private output and Tas in78s
(.checks/20261104T180939Z-ISCenW.log).

Final observed Tasleroch UID827096491 is connected to DizzysHellheim, idle upstairs
at[-202.99,32.97,-467.42],grounded/sheltered/near_fire,109.5/109.5HP,injurySeq0.
Three active meals: Sausages,BoarJerky,QueensJam;2 carried each,1 Wood and4 Stone.
Maximum-health decay is not injury. Bed1274003559:489 remains Mine=true,Owned=true.
Home Wood Door1:14823 was explicitly closed by t-5 and freshly confirmed closed.
Storage double doors1:15335/1:15334 remain observed open; no automatic closure
claim is earned. Home rules remain idleMinutes0,afterDeath stay,afterOutage false.
Evidence .checks/hearth-final-home-status.json,.checks/hearth-final-home-map.json,
.checks/hearth-two-wood-after-inventory.json,.checks/task-e5g1xg9_.json.

VM console session2 works. Release the lease at handback; take/renew it before
live work as agent:codex/portal-deploy. Keep the game online. After RDP use
C:\renki\release-rdp.ps1. The private stream's final /api/video result is verified
gameplay,scope1:DizzysHellheim:827096491 (.checks/hearth-posttest-video.json).
Only private local output is configured, not an external public broadcast.

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
y32 and lower chests y30.97,with identical empty inventory before/after. Gaps
1:15433,1:15438,1:15414 remain unknown. Pukeberries1:15420 are not sustaining food.
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
