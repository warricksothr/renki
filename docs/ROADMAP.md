# Roadmap

Where renki is going, in milestones, and how we know each one is finished.
The tickets carry the work. This page carries the order, the dates, the exit
goals and the rules for running the work in parallel, so it changes only when
a plan does.

Written 2026-11-01 and hardened on 2026-11-02, when every milestone was groomed
so that one agent can pick up one ticket and finish it alone. The goals are also
on each epic, as its definition of done. Re-read and re-dated every Sunday (see
[Cadence](#cadence)).

## Where we are

Work started on 2026-10-28. In five days 153 pull requests merged.

- **M1 (join, follow, stop, status, chat)** closed in the game on 2026-10-28.
- **The panel** is mostly built: tailnet login, live map, camera, atlas layers, settings, stop, home and logout. Its epic is still open for bookkeeping and for controlling behaviour instances.
- **M2 and M3** have a lot of working parts in the game, such as harvest, delivery receipts, home unloading, danger zones and native melee. But not one of their epic criteria is ticked.
- **Most effort since 10-31 went into the food and household track** (TKT-01M6864352, Pressure-driven work, storage and maintenance goals). That track is useful, but it is on no M2 or M3 goal, so its tickets now sit in M5.
- **The live trial is the bottleneck.** `just vm gather-trial`, the M2 benchmark, has never passed. On 10-31 one run felled and split a tree, then a falling log killed the bot, and another stopped at a startup injury. A flee from the raised base on 10-31 left the bot standing still.
- **The 2026-11-01 code review** (TKT-01M6DCF2AB, Code review 2026-11-01: robustness, trust and gate gaps) found no broken design. It did find gaps in containment, durability, trust and the CI gate that matter before the bot runs unattended.

The pace that matters for dates is not PRs a day. It is *passing live trials a day*. That rate has been roughly zero to two for the benchmark trials, so the dates below give each milestone one week of trials.

## Milestones

| | Milestone | Outcome | Epics | Due |
|---|---|---|---|---|
| M1 | Join and chat | Done 2026-10-28 | TKT-01M61T36B8 | done |
| M2 | Hauls and gathers wood | The wood benchmark in Nimah's words | TKT-01M62MZN7Q | 2026-11-08 |
| M3 | Survives and stands with its owner | Retreats, fights when allowed, defends its owner, avoids where it was hurt | TKT-01M62N2557 | 2026-11-15 |
| M4 | Safe to leave running | 24 h unattended without a crash, a leak or a lost store, gated by CI | TKT-01M6DCF2AB, close-out of TKT-01M628X106 | 2026-11-22 |
| M5 | Runs the base | Fires lit, food not burnt, items in their chests, the pantry stocked, behaviours the owner can edit | TKT-01M62VXZ2F, TKT-01M6864352, TKT-01M62VXYQB | 2026-12-06 |
| M6 | Agent with tools | The LLM looks before it acts, wakes on events within a budget, respects autonomy levels | TKT-01M62VXZA3 | 2026-12-20 |
| M7 | Explores and maps | Bounded expeditions that return with new coverage, and a portable atlas layer | TKT-01M66PFAYA, TKT-01M66Q5SFW | 2027-01-17 |

| Milestone | Open tickets | Offline | Live |
|---|---|---|---|
| M2 | 23 | 9 | 14 |
| M3 | 22 | 14 | 8 |
| M4 | 36 | 29 | 7 |
| M5 | 29 | 13 | 16 |
| M6 | 16 | 15 | 1 |
| M7 | 26 | 19 | 7 |
| Unscheduled | 21 | 20 | 1 |

**Unscheduled** holds work no goal needs: terrain shaping and construction (TKT-01M6CEA0K3), procedure authoring (TKT-01M6701PBV, TKT-01M6701ZJS), archery, hunting the owner's killer, revenge on players, group peeling, portal travel, the look-ahead and occupancy-grid spikes, GeoPackage export, the Valheim Plus migration and the late-door bug. Schedule them at a weekly review, or sooner if the owner promotes one.

M3 and M4 overlap on purpose. M4's work is controller, CI and tooling, and it does not compete with M3 for the game client. Its *soak* needs the VM and runs after the M3 trials.

M5 to M7 are ordered for focus, not because every ticket requires it:

- M5 and M6 share chest memory and the behaviour engine, and M6's worked example needs stock targets built on M5's chest memory.
- M6's tools wait on the chat-trust fix TKT-01M6DCFSME, which is in M4.
- M7 depends only on M2 and M3. Its offline work, coverage and export, can run beside M5 and M6 (see [Conflict hotspots](#conflict-hotspots)).

## SMART exit goals

Each goal is specific, has a number and a named evidence source, builds on something that already works, ties to the milestone's outcome, and has the milestone's date unless it says otherwise. A goal is met when the evidence is linked in a note on the ticket named in the goal and that ticket's criterion is ticked. "Run" means a supervised `just vm …` trial that writes `.checks/<trial>-<ts>/summary.json`. The bounded trials need no owner staging (docs/TESTING.md).

Goals marked **(proposed)** have thresholds the owner has not confirmed yet. See [Decisions for the owner](#decisions-for-the-owner).

### M2: hauls and gathers wood (due 2026-11-08)

| # | Goal | Measure | Evidence | Ticket |
|---|---|---|---|---|
| 2.1 | The autonomous gather loop works end to end | `just vm gather-trial` passes 3 runs in a row: fell, split, collect, deposit gains into a home chest, return. Each `summary.json` records a duration under 15 min, no death and no swimming | three `summary.json` paths | TKT-01M62N12JW |
| 2.2 | Owner-haul scene | In 2 owner sessions of at least 30 min, with the owner inside the bot's perception radius for at least 90 % of each, at least 3 trips each. The bot picks up at least 80 % of the wood, resin and seeds seen within 30 m of the owner, not counting drops the owner placed on purpose (the loot template's deliberate-drop rule), delivers at 80 % full or 60 s after the pile is gone, then comes back | the haul report (TKT-01M6DDFTDD) | TKT-01M62N12HJ |
| 2.3 | A wood request picks a sensible site **(proposed)** | 3 of 3 requests, by chat or the panel's wood action, go to the most recent owner work site within 150 m of home, else the nearest safe known trees within 150 m of home, and finish 2.1's loop. At least one run starts with a recorded work site | the wood-request trial's summaries | TKT-01M6EFHE80, TKT-01M62N12G9 |
| 2.4 | Honest failures **(proposed)** | With no axe, the bot says why in chat within 10 s and sends no further body task, in one live run. No chest and chest full are covered by offline tests | trial summary, test names | TKT-01M6EFHE80, TKT-01M62N12C9 |
| 2.5 | M2 tickets tell the truth | Every M2 child is done, or has a note saying what is left and is moved to a later milestone. No M2 ticket is in progress without a note in the last 3 days | `git ticket list --milestone M2` | TKT-01M62MZN7Q |

2.3 changed from 300 m because the site planner refuses trees more than 150 m from home, and a trial cannot type in-game chat. 2.4 changed because staging "no chest" or "chest full" live would mean emptying or filling the owner's real chests.

### M3: survives and stands with its owner (due 2026-11-15)

| # | Goal | Measure | Evidence | Ticket |
|---|---|---|---|---|
| 3.1 | Retreat instead of standing still **(proposed)** | Across every live trial from 11-03 to 11-15, at least 3 consecutive low-HP episodes with a threat within 15 m in which `flee` is dispatched within 1.5 s and the bot reaches its owner or home alive, and none that fails. One starts on the raised base that failed on 10-31 | `just vm retreat-evidence` (TKT-01M6EFHEEN) | TKT-01M62N256M, TKT-01M6EFHED7 |
| 3.2 | Fight when allowed, stop when told | With `AllowCombat` on, 3 of 3 `just vm combat-trial` runs that find a Greyling end in its observed death. An owner stop, by chat or the panel, ends combat within 2 s in at least one. Runs that find no Greyling do not count and do not break the streak. With combat off, no attack is ever sent | summaries with stop timing (TKT-01M6EFHEG5), and `MeleeTests` for the off case | TKT-01M62N2582, TKT-01M6EFHEBC |
| 3.3 | Defend the owner | 2 of 2 escort or guard runs engage a confirmed attacker of a protected player within 10 m, and stop on "stay out of it" with no further attack | `just vm escort-check` with defence triggers and timing (TKT-01M6EFHEHM, TKT-01M6EHP58N) | TKT-01M62N25AD |
| 3.4 | Avoid where it was hurt | A replay test shows routes home detouring around remembered danger zones. One live trip detours around a zone still scoring at least 1 | test name plus an audit window | TKT-01M6EFHEK3, TKT-01M6EFHS5E |
| 3.5 | Containment for the fights | TKT-01M6DCFSEG (stow and marker safety) and TKT-01M6DCFSG0 (goal lifecycle, three children) are done | ticket status | TKT-01M6DCF2AB |

3.1 counts natural episodes because the trials refuse to injure the bot on purpose. If fewer than 3 occur by 11-12, the owner decides between funding the isolated world (TKT-01M656NE4C) and accepting a slip. 3.3 says "confirmed attacker" because a remotely owned creature may not reveal whom it targets (docs/ESCORT.md). A confirmed attacker is one whose observed AI target is a protected player other than the bot. The plugin must first report that trigger (TKT-01M6EHP58N).

### M4: safe to leave running (due 2026-11-22)

| # | Goal | Measure | Evidence | Ticket |
|---|---|---|---|---|
| 4.1 | The gate runs itself | `just ci` runs on every pull request. It lints Go, the panel, Renki.Core and the Python scripts, and either compiles Renki.Plugin or fails loudly wherever it was skipped, depending on the owner's choice of plugin build | the workflow file and a red run on a planted error | TKT-01M6DCFST1 and its four children |
| 4.2 | A 24 h soak | `just vm soak` runs 24 h with the bot idle-working at home. It fails on a controller crash or restart, more than 100 open TCP connections, audit growth over 2 GB a day, a store write error, or a plugin `internal` failure | soak `summary.json` | TKT-01M6DDFTF6, TKT-01M6EFGYV6 |
| 4.3 | Stores survive a hard stop | A crash test plus 3 live controller kills during saves, before the soak window, leave memory.json, goals.json and settings readable, with no lost home | test name and a note | TKT-01M6DCFSHH, TKT-01M6EFGYV6 |
| 4.4 | Only owners command it | Owners bind to player IDs. Lines from other players cannot steer an owner's turn, as shown by a test that replays the review's injection scenario | tests | TKT-01M6DCFSME, TKT-01M6EFGYX3, TKT-01M6EFGYYY |
| 4.5 | The link heals itself | After a 20 s controller pause the plugin reconnects and accepts tasks within 10 s of resume | bridge test plus one live pause | TKT-01M6DCFS8Y, TKT-01M6EFGYNM |
| 4.6 | Review closed | Every child of TKT-01M6DCF2AB is done, or has a note saying why it stands, which includes a move to a later milestone. The panel epic's criteria are ticked or moved | ticket status | TKT-01M6DCF2AB, TKT-01M628X106 |

### M5: runs the base (due 2026-12-06)

The evidence for 5.1 to 5.3 and 5.6 comes from two 2 h home soaks (TKT-01M6EFM4G7) scored by `just vm upkeep-report` (TKT-01M6EFK453). The soaks run on top of M4's soak supervisor.

| # | Goal | Measure | Evidence | Ticket |
|---|---|---|---|---|
| 5.1 | Fires stay lit **(proposed)** | In 2 consecutive 2 h home soaks with the fuel instance on, every fire within 32 m of home is lit in at least 98 % of night samples taken after the first 120 s of each night, and in every sample where a cooking slot within 3 m holds food. A fire at fuel 0 within the first 120 s after dusk passes, because the bot can only start stoking once it sees night and may have to fetch fuel from a chest; any night sample at fuel 0 after that, or any sample at fuel 0 with food cooking over it, fails. Fires may run dry by day. At least one matched fuel receipt per soak. Stored fuel never drops below the reserve. The bot is within 32 m of home for at least 80 % of each soak | upkeep report, fires | TKT-01M62VXZ5P |
| 5.2 | No food burns | In the same soaks, 0 slots go from cooking or done to burnt. In 1 owner session of at least 30 min with at least 5 raw items left cooking, 5 of 5 come off before burning and reach a chest with matched receipts | upkeep report, cooking | TKT-01M62VXZ6Z |
| 5.3 | Assigned items reach their chests | With at least 3 assignments and a loose-items chest, the end-of-soak snapshots show 0 units of an assigned item outside its chests (don't-touch chests excepted) and none in the pack above `keep`. A full chest gets one chat line within 10 s naming the item and count, at most every 5 min | upkeep report, assignments | TKT-01M62VXZ84, TKT-01M6EFKDSV |
| 5.4 | Clone and edit from chat | Offline chat tests for list, start, stop, enable, disable, clone, set, and a non-owner refusal. Live: 3 of 3 clone, set and start sequences reply within 2 s and end in a matched deposit | test names plus audit windows | TKT-01M62VXYV3, TKT-01M6EFKDNH |
| 5.5 | Clone and edit from the panel | Panel scenarios cover list, clone, editing one field, start, stop, and a rejected edit that shows the field and reason. Live: one panel clone appears in `GET /api/behaviours` as changed by `panel` and runs once with a matched deposit | `just check panel ci` plus an audit window | TKT-01M6EFK472, TKT-01M6EFKDNH |
| 5.6 | Pantry stays stocked | In the 5.1 soaks, every chest inspection after the first completed household episode shows each food target at or above its minimum. 0 episodes end with an uncertain action. 0 deaths | upkeep report, pantry | TKT-01M6AT4EFW |
| 5.7 | Work continues under pressure | `just vm gather-trial --gather-cycles 2` passes 2 runs in a row. Each cycle fells a whole tree, makes a matched home deposit, records why it went home, repairs the axe if it is under 90 %, and returns within 12 m of the same site. Each run takes under 30 min with no death | two `summary.json` | TKT-01M6EFKWBX, TKT-01M6EFKWED |
| 5.8 | Honest tool readiness | With no usable axe, a chat line within 10 s names the axe, the station and each missing material with its count (offline tests for no axe, broken axe, no station, shortfall and stored axe). One live flint axe craft with exact counts before and after | tests plus an audit window | TKT-01M6EFKWGA, TKT-01M6EFKWJ6, TKT-01M64T4WGZ |
| 5.9 | M5 tickets tell the truth | Every M5 ticket is done, or has a note saying what is left and is moved | `git ticket list --milestone M5` | the three M5 epics |

Smelters, kilns, fermenters and beehives are out of these goals until the owner says whether the base has them.

### M6: agent with tools (due 2026-12-20)

Measured by `just vm agent-report` (TKT-01M6EFC7R8), which reads new `agent wake`, `agent tool` and `agent proposal` audit records. The live evidence is gathered in one ticket (TKT-01M6EFK122), so the implementation tickets stay offline.

| # | Goal | Measure | Evidence | Ticket |
|---|---|---|---|---|
| 6.0 | Trust before tools | Before the tool registry merges, an injection replay test cannot steer an owner's turn | test name | TKT-01M6DCFSME (M4) |
| 6.1 | Model chosen on data | 3 runs of at least 20 scenarios against haiku-5-5 and the local gemma. The default model has tool accuracy of at least 90 %, no false promises, and p95 reply latency of at most 10 s | `.checks/voice-eval-*/summary.json` plus a comparison note | TKT-01M630V80T |
| 6.2 | Looks before it acts, and hears the result | In the look-first scenarios a query precedes the first action in at least 80 % of runs. Every action call returns accepted or refused and has an `agent tool` record | voice-eval summary, unit tests | TKT-01M62VXZAV, TKT-01M6EFCK6H |
| 6.3 | Never promises what it cannot do | 0 false promises in the scenario set and in the live owner sessions | voice-eval, agent report | TKT-01M630V829 |
| 6.4 | Wakes within a budget **(proposed)** | Over a 4 h wake soak, woken turns average at most 4,000 tokens an hour, no clock hour exceeds 8,000, at most 6 woken turns an hour, and no danger event wakes the agent | agent report | TKT-01M62VXZBN, TKT-01M6EFG477 |
| 6.5 | Wakes promptly | Over the same soak, event to turn start p95 at most 7 s, event to first chat line p95 at most 20 s | agent report | TKT-01M62VXZBN |
| 6.6 | Respects autonomy | At ask, no self-started action is dispatched without an owner's yes. At tell, every one is announced. Pins refuse the agent at every level | agent report and tests | TKT-01M62VXZCV, TKT-01M62VXZE3 |
| 6.7 | Proposals worth making **(proposed)** | Over at least 2 owner sessions totalling at least 2 h with at least 10 answered proposals, owners reject at most 30 %. Lapsed proposals are reported separately | agent report | TKT-01M62VXZCV, TKT-01M6EFK122 |
| 6.8 | Worked example end to end | A stock-low event leads to a query, an ask, a yes, gathering until the target, and a report of the ending. It passes once at ask and once at tell | audit windows, agent report | TKT-01M62VXZE3, TKT-01M6EFK122 |
| 6.9 | M6 tickets tell the truth | Every child is done, or has a note saying what is left and is moved | `git ticket list --milestone M6` | TKT-01M62VXZA3 |

### M7: explores and maps (due 2027-01-17)

Coverage is measured in atlas samples: one sample is a 4 m cell, 16 m². "New" means first received during the expedition, from this bot, inside the authorized circle. That needs a first-seen time in the atlas (TKT-01M6EFQN74), because today it keeps only the latest observation.

| # | Goal | Measure | Evidence | Ticket |
|---|---|---|---|---|
| 7.1 | The kinds of coverage stay apart | Offline tests: the measured-coverage query counts 0 m² for re-confirmed, resent, minimap-only and visited-only ground, and visited ground keeps its height on raised floors | `just check controller test-packages ./internal/atlas ./internal/app` | TKT-01M6EFQN74, TKT-01M6EFQN91, TKT-01M6EFQNBJ |
| 7.2 | Frontier choice is honest in replay | At least 8 recorded scenes each return ranked candidates or "no safe frontier". No failed approach is retried without new evidence | `just check controller test-packages ./internal/explore` | TKT-01M6EFQNMW, TKT-01M6EFQNR1 |
| 7.3 | First bounded expedition **(proposed)** | `just vm expedition-trial` passes 3 runs in a row, radius at most 150 m, combat off. Each run records at least 2,000 m² (125 samples) of new terrain inside the circle and a successful return, ending within 3 m across and 1.75 m vertically of the pinned anchor. Elapsed time at most 10 min, work at most 5 min, travel at most 600 m, no death | three `summary.json` | TKT-01M6EFQPB8 |
| 7.4 | Stop and partial results are honest | A run with `--stop-after 60`, an owner stop through the panel, is idle within 2 s, its goal cancelled and no task following. A scene with no frontier ends partial with no outbound leg | `summary.json` plus a test name | TKT-01M6EFQPB8, TKT-01M6EFQP43 |
| 7.5 | The goal host survives the replay list | All 15 scenarios in docs/EXPLORATION.md are named tests. Each ends in a different allowed choice, a partial result or `needs_plan`, and budgets never reset | `just check controller test-packages ./internal/goal` | TKT-01M6EFQP5X |
| 7.6 | The panel shows the expedition | Fixture tests draw boundary, anchor, coverage, trail, frontier and budget, with planned and travelled lines styled differently. A map snapshot from one 7.3 run is linked | `just check panel ci`, `just vm map-snapshot` | TKT-01M6EFQPDB, TKT-01M6EFQPB8 |
| 7.7 | One portable layer round-trips | A 512 × 512 m terrain and biome export from the VM atlas, previewed on import, matches 100 % of samples (height to 0.1 m, biome, water, time, source). Negative coordinates and holes survive, the hash verifies, a 1-byte change is refused, nothing is marked fresh, and no player, waypoint or container data is included | export and import-preview recipes, test names | TKT-01M6EFQNDF, TKT-01M6EFQNFN |
| 7.8 | M7 tickets tell the truth | Every child is done, or has a note and is moved | `git ticket list --milestone M7` | the two M7 epics |

## Working in parallel

The milestones are cut into tickets that each fit one pull request for one agent. This section is how to hand them out.

### Definition of ready

A ticket can go to an agent when:

1. Its description states the outcome and why it matters to its milestone.
2. `### Files` names the files or packages to start in.
3. `### Verification` names the `just` recipes that prove it offline, and, where live evidence is needed, the `just vm …` recipe and what in its summary counts.
4. Each acceptance criterion can be checked by someone else.
5. It fits one pull request, about a day of work. Larger work is an umbrella that depends on its children.
6. Dependencies are recorded with `git ticket link`, so `git ticket ready` tells the truth.
7. It carries the `live` label if any criterion needs the game client on the VM.
8. Its status is `ready`. Only a person promotes a draft (AGENTS.md).

### Two lanes

- **Offline** tickets need only the repository and `just ci`. Any number run at once, each in its own worktree.
- **Live** tickets (label `live`) need the one Windows VM and its one game client to deploy, run a trial or check something in the game. Reading from the VM (audit windows, the report recipes, map snapshots, read-only panel requests) changes nothing there, needs no lease, and does not make a ticket live. They run one at a time, under the VM lease (TKT-01M6EF5DTG, VM: exclusive lease for deploys and live trials). Every live ticket depends on it. Until it lands, one person decides who uses the VM.

The usual shape is offline code first, then a short live session for evidence. Many tickets split that way on purpose, so the code can merge while the VM is busy.

### Handing a ticket to an agent

Give the agent the ticket ID and nothing it can read for itself:

> Work TKT-… (title). Read AGENTS.md and the ticket, then claim it, plan it, implement it in your own worktree, pass `just check ci` and `just review`, and open a pull request against main that names the ticket. Do not touch the VM unless the ticket is `live` and you hold the lease. Tick only the criteria you met; leave a note for anything you could not.

One agent per ticket. An agent that finds a ticket too big files children and stops, rather than landing half of it.

### Wave 1: offline tickets with no open dependencies

These can all start as soon as they are promoted. Each is in its own files except where [Conflict hotspots](#conflict-hotspots) says otherwise.

| Milestone | Tickets |
|---|---|
| M2 | TKT-01M6EF5DTG (VM lease, first of all), TKT-01M6DDFTDD (haul report), TKT-01M6EFHE67 (drop piles), TKT-01M6EFHE2N (loot template) |
| M3 | TKT-01M638V1XC, TKT-01M6DCFSEG (both already ready), TKT-01M638V1VY, TKT-01M6EFHEBC, TKT-01M6EFHED7, TKT-01M6EFHEEN, TKT-01M6EFHEHM, TKT-01M6EFHEK3, TKT-01M6EFHS9D, TKT-01M6EFHSCK, TKT-01M6EFHSFV |
| M4 | TKT-01M6EFGJY8 (CI workflow, early), TKT-01M6EFGJZK (after decision 4), TKT-01M6EFGK0R, TKT-01M6EFGK2M, TKT-01M6DCFSHH, TKT-01M6DCFSNT, TKT-01M6DCFS8Y, TKT-01M6DDBX5T, TKT-01M6DCFSME, TKT-01M6EFGYX3 (after decision 5), TKT-01M6EFGYYY, TKT-01M6DCFSK1, TKT-01M6DCFSAR, TKT-01M6DCFSQ6, TKT-01M6DCFSRN, TKT-01M6DCFSVE, TKT-01M6DDBX7K, TKT-01M6DCFS5D, TKT-01M6DCFSCK |
| M5 | TKT-01M62VXYV3 (chat behaviours), TKT-01M6EFK453 (upkeep report), TKT-01M6EFK472 (panel behaviours) |
| M6 | TKT-01M630V829, TKT-01M630V83M, TKT-01M69V7YHK, TKT-01M6EFC7R8 (agent report) |
| M7 | TKT-01M6EFQN74 (first-seen coverage), TKT-01M6EFQN91 (walked ground), TKT-01M6EFQNDF (layer export) |

Priority inside the wave: the VM lease, the CI workflow, and the M2/M3 tickets that sit on a critical path below. The M5 to M7 tickets in the wave fill spare capacity and should not hold up review of the near milestones.

### Critical paths

- **M2.** Goal 2.1: the lease, then repeated `gather-trial` runs on TKT-01M62N12JW. Goal 2.2 is longer: the lease, then drop piles and the loot template, then deliver-when-full (TKT-01M6EFHE4E), then integration on TKT-01M62N12HJ, then two owner sessions. It runs past 11-08 unless the owner can play by about 11-06.
- **M3.** Goal 3.1: the lease, then the flee fix TKT-01M6EFHED7, a deploy, and then waiting for three natural low-HP episodes, which nobody can schedule. Second chain: TKT-01M6EFHEBC, then TKT-01M6EFHEG5 and TKT-01M6EFHEHM, plus the escort trigger TKT-01M6EHP58N (live, plugin), then combat trials and two owner escort sessions.
- **M4.** The lease and the soak supervisor TKT-01M6DDFTF6, then the 24 h run TKT-01M6EFGYV6. That run also waits on TKT-01M6DCFSHH, TKT-01M6DCFSNT, TKT-01M6DCFS8Y and TKT-01M6DDBX5T, which run in parallel. The soak window has to start by 2026-11-20.
- **M5.** Machine add_to (TKT-01M62VXZ4D), then the fuel template (TKT-01M62VXZ5P), then the two home soaks (TKT-01M6EFM4G7). A second chain is about as long: stow and chat behaviours, then chest assignments (TKT-01M62VXZ84), then idle sorting (TKT-01M6EFKDSV). Realistically the soak window is the week of 11-30.
- **M6.** The trust fix TKT-01M6DCFSME and the model spike TKT-01M630V80T, then the tool registry (TKT-01M62VXZAV), wake bus (TKT-01M62VXZBN), autonomy (TKT-01M62VXZCV), behaviour tools (TKT-01M62VXZE3) and live evidence (TKT-01M6EFK122). Goal 6.8 also needs the stock-low event (TKT-01M62VXZF8, in M6), which waits on chest memory (TKT-01M62VXYWJ).
- **M7.** M2 and M3 close, then readiness (TKT-01M6EFQNT4) and the goal spec (TKT-01M6EFQP11), then the live survey leg (TKT-01M6EFQNWT), the return (TKT-01M6EFQNYK), the goal runner (TKT-01M6EFQP43), replays (TKT-01M6EFQP5X) and three expedition passes (TKT-01M6EFQPB8). Frontier ranking and the trial supervisor run alongside.

### Conflict hotspots

Work that touches these together should go one after another, or be coordinated in the pull request.

- **`internal/goal/store.go` and `runtime.go`:** goal lifecycle (TKT-01M6EFHSCK, TKT-01M6EFHSFV), food acquisition (TKT-01M69R5A75), pressure goals (TKT-01M6864352) and M7's goal kind (TKT-01M6EFQP11, TKT-01M6EFQP43).
- **`internal/chat/brain.go`:** the three chat-trust tickets (TKT-01M6DCFSME, TKT-01M6EFGYX3, TKT-01M6EFGYYY), then the M6 registry.
- **Plugin task files** (`TaskArgs.cs`, `BodyTasks.cs`, `KiteTask.cs`): M2/M3 movement, M5 machine tasks and M7's survey task.
- **Atlas schema:** TKT-01M6EFQN74, TKT-01M6EFQN91 and TKT-01M6EFQNBJ each add a migration, so they land in order.
- **Lint rollout:** TKT-01M6EFGK0R touches most Go and panel files. Land it in a quiet window, or with suppressions that later tickets remove.
- **The VM:** the M4 24 h soak, M5's 2 h soaks, M6's 4 h wake soak and M7's expedition trials all need it. Book them in the weekly review.

## How the goals are measured

- **Bounded trials** (`just vm gather-trial`, `combat-trial`, `harvest-trial`, `goal-trial`, `resupply-food`, `escort-check`, `route-benchmark`, and later `soak` and `expedition-trial`) write a `summary.json`. They always restore combat off and log out or stop. A goal counts *consecutive* passes, so a flaky pass does not close a milestone.
- **Reports from the audit log** score what cannot be supervised: the haul report (M2), retreat evidence (M3), the upkeep report (M5) and the agent report (M6). The owner plays and the numbers come afterwards.
- **Offline tests** cover the failure branches (no chest, combat off, injection) that are slow or unsafe to stage live.

Every report recipe above is a ticket of its own, so a goal never waits on tooling that nobody owns.

## Decisions for the owner

These block work or set a goal's numbers. They are listed in the order they block. Milestone moves and dependency changes the 2026-11-02 planning pass already applied are noted on each ticket; revert any you disagree with.

1. **Promote the wave.** Start with the VM lease TKT-01M6EF5DTG, then the wave 1 tickets above. Agents cannot claim drafts.
2. **Release the stale claims** from the 10-29 to 10-31 sessions on in-progress M2, M3 and panel tickets, so new agents can take them. Close TKT-01M628X134 and TKT-01M628X14M, which are done.
3. **Confirm the Behaviours epic move** (TKT-01M62VXYQB). Already applied: it is in M5, where goals 5.4 and 5.5 carry its owner-editing outcome, and M2 no longer depends on it, because its M2 part is the same work as TKT-01M62N12HJ.
4. **CI:** GitHub Actions or not, where CI installs `git-ticket`, and how CI builds Renki.Plugin (stub assemblies, a self-hosted runner, or required only at package and deploy time). This sets goal 4.1's final wording.
5. **Owner binding** for 4.4: learn the player ID on first sight and confirm it in the panel, or require IDs in config.
6. **Owner time:** two haul sessions of 30 min or more by about 11-06 (2.2), two escort sessions before 11-15 (3.3), a 30 min cooking session (5.2), and 2 h of play for M6 (6.7).
7. **VM windows:** 24 h from about 11-20 (4.2), about 6 h in the week of 11-30 (M5), 4 h for the M6 wake soak.
8. **Thresholds marked (proposed):** 2.3, 2.4, 3.1, 5.1, 6.4, 6.7 and 7.3.
9. **Scope calls:** which machines the base has (M5), whether the bot may spend shared materials to craft tools, whether waking stays off by default until M6's goals are met, whether 4 m loaded-terrain samples count as "observed" for M7, and whether M7's offline tickets may start before M2 and M3 close.
10. **For the live checks:** name a chest the bot may fill for the chest-full check, and say whether you will stage the lake walk for the drowning bug.

## Cadence

- **Every Sunday:** compare each open milestone against its goals and record the result as a note on its epic. If a date will be missed, re-date it here with one line saying why. Do not let it slip silently. Book the VM windows for the week, and promote the next wave.
- **When a milestone closes:** tick its epic, close or move its children, and revisit the next milestone's proposed thresholds with the evidence now available.
- **New work** lands in `draft`. It joins a milestone when a person promotes it, and is otherwise left out of the dates above.

## Risks

- **Live trials depend on the production world.** A death or a lost item costs real progress. Every trial already refuses to run without its safety preconditions, and M4's durability work protects the stores.
- **The VM is the real bottleneck.** About 50 offline tickets can run at once. The live work cannot, and the M4 to M7 soaks alone need about 36 h of VM time. The lease makes that visible, and the weekly review books it.
- **M2's owner-haul goal is at risk for 11-08.** It needs four pull requests, a deploy and two owner sessions. If it slips, re-date M2 rather than closing on 2.1 alone.
- **Natural episodes for 3.1 may not happen.** See the fallback under the M3 goals.
- **Merge pressure.** Wave 1 is wide. Review capacity, not agent capacity, sets the pace, so keep the near milestones first.
- **Work in flight drifts from the criteria.** At the weekly review, ask whether the active tickets serve the open milestone's goals.
