# Roadmap

Where renki is going, in milestones, and how we know each one is finished.
The tickets carry the work. This page carries the order, the dates and the
exit goals, so it changes only when a plan does.

Written 2026-11-01. The dates and goals are also on each epic, as its due date and its definition of done. Re-read and re-dated every Sunday (see [Cadence](#cadence)).

## Where we are

Work started on 2026-10-28. In five days 153 pull requests merged.

- **M1 (join, follow, stop, status, chat)** closed in the game on 2026-10-28.
- **The panel** is mostly built: tailnet login, live map, camera, atlas layers, settings, stop, home and logout. Its epic is still open for bookkeeping and for controlling behaviour instances.
- **M2 and M3** have a lot of working parts in the game, such as harvest, delivery receipts, home unloading, danger zones and native melee. But not one of their epic criteria is ticked.
- **Most effort since 10-31 went into the food and household track** (TKT-01M6864352, Pressure-driven work, storage and maintenance goals). That track is useful, but it is not on the M2 or M3 criteria.
- **The live trial is the bottleneck.** `just vm gather-trial`, the M2 benchmark, has never passed. On 10-31 one run felled and split a tree, then a falling log killed the bot, and another stopped at a startup injury. A flee from the raised base on 10-31 left the bot standing still.
- **The 2026-11-01 code review** (TKT-01M6DCF2AB, Code review 2026-11-01: robustness, trust and gate gaps) found no broken design. It did find gaps in containment, durability, trust and the CI gate that matter before the bot runs unattended.

The pace that matters for dates is not PRs a day. It is *passing live trials a day*. That rate has been roughly zero to two for the benchmark trials, so the dates below give each milestone one week of trials.

## Milestones

| | Milestone | Outcome | Epics | Due |
|---|---|---|---|---|
| M1 | Join and chat | Done 2026-10-28 | TKT-01M61T36B8 | done |
| M2 | Hauls and gathers wood | The wood benchmark in Nimah's words | TKT-01M62MZN7Q, the template part of TKT-01M62VXYQB | 2026-11-08 |
| M3 | Survives and stands with its owner | Retreats, fights when allowed, avoids where it was hurt | TKT-01M62N2557 | 2026-11-15 |
| M4 | Safe to leave running | 24 h unattended without a crash, a leak or a lost store, gated by CI | TKT-01M6DCF2AB, close-out of TKT-01M628X106 | 2026-11-22 |
| M5 | Runs the base | Owner-editable behaviours, fuel, tending, chest assignments, pressure-driven work | TKT-01M62VXYQB (remainder), TKT-01M62VXZ2F, TKT-01M6864352 | 2026-12-06 |
| M6 | Agent with tools | The LLM looks before it acts, wakes on events within a budget, respects autonomy levels | TKT-01M62VXZA3 | 2026-12-20 |
| M7 | Explores and maps | Bounded expeditions that return with new coverage, and a portable atlas | TKT-01M66PFAYA, TKT-01M66Q5SFW | 2027-01-17 |

**Unscheduled:** terrain shaping, paths and construction (TKT-01M6CEA0K3), which the owner filed on 2026-11-01 as future capability work. Schedule it at a weekly review once M5 closes, or sooner if the owner promotes it.

M3 and M4 overlap on purpose. M4's work is controller, CI and tooling, and it does not compete with M3 for the game client. Its *soak* needs the VM and runs after the M3 trials each day.

M5 to M7 are ordered by dependency. Owner-editable behaviours come before the agent can drive them. The agent's trust fixes (TKT-01M6DCFSME) come before it gets more tools. Exploration waits on M2 and M3, as its epic already says.

## SMART exit goals

Each goal is specific, has a number and a named evidence source, builds on something that already works, ties to the milestone's outcome, and has the milestone's date unless it says otherwise. A goal is met when the evidence is linked in a note on the ticket named in the goal and that ticket's criterion is ticked. "Run" means a supervised `just vm …` trial that writes `.checks/<trial>-<ts>/summary.json`. The bounded trials need no owner staging (docs/TESTING.md).

### M2: hauls and gathers wood (due 2026-11-08)

| # | Goal | Measure | Evidence | Ticket |
|---|---|---|---|---|
| 2.1 | The autonomous gather loop works end to end | `just vm gather-trial` passes 3 runs in a row: fell, split, collect, deposit gains into a home chest, return. Each run under 15 min, no death, no drowning | three `summary.json` paths | TKT-01M62N12JW |
| 2.2 | Owner-haul scene | In 2 owner sessions of at least 30 min, at least 3 trips each: the bot picks up at least 80 % of the wood, resin and seeds dropped within 30 m of the owner. It delivers when 80 % full or 60 s after the pile is gone, then comes back | a haul report computed from the audit (TKT-01M6DDFTDD) | TKT-01M62N12HJ |
| 2.3 | "Go chop wood" picks a sensible site | 3 of 3 chat requests go to the last observed owner work site if there is one within 300 m, else the nearest known trees, and finish 2.1's loop | audit window per run | TKT-01M62N12JW, TKT-01M62N12G9 |
| 2.4 | Honest failures | For no chest, chest full and no axe, the bot says why in chat within 10 s and stops. One offline test and one live run each | test names and audit lines | TKT-01M62N12JW, TKT-01M62N12C9 |
| 2.5 | M2 tickets tell the truth | Every M2 child is done, or has a note saying what is left and is moved to a later milestone. No M2 ticket is in-progress without a note in the last 3 days | `git ticket list --milestone M2` | TKT-01M62MZN7Q |

### M3: survives and stands with its owner (due 2026-11-15)

| # | Goal | Measure | Evidence | Ticket |
|---|---|---|---|---|
| 3.1 | Retreat instead of standing still | In 3 of 3 runs where HP falls below the retreat threshold with a threat within 15 m, `flee` is dispatched within 1 s and the bot reaches its owner or home alive. One run must start on the raised base that failed on 10-31 | `combat-trial` summaries plus audit windows | TKT-01M62N256M |
| 3.2 | Fight when allowed, stop when told | With `AllowCombat` on, 3 of 3 `just vm combat-trial` runs end in an observed Greyling death. A chat `stop` ends combat within 2 s in at least 1 of them. With it off, no attack is ever sent | summaries, plus an offline test for the off case | TKT-01M62N2582 |
| 3.3 | Defend the owner | 2 of 2 escort or guard runs engage a creature that targets the owner within 10 m, and stop on "stay out of it" | audit windows | TKT-01M62N25AD |
| 3.4 | Avoid where it was hurt | A replay test shows routes preferring to avoid known danger zones. One live trip detours around a zone recorded earlier that day | test name plus an audit window | TKT-01M62EP0PD |
| 3.5 | Containment for the fights | The review's M3 safety tickets are done: TKT-01M6DCFS3K (done), TKT-01M6DCFSEG (stow and marker safety) and TKT-01M6DCFSG0 (goal lifecycle) | ticket status | TKT-01M6DCF2AB |

The disposable test world (TKT-01M656NE4C) stays optional. The owner chose native encounter work over isolated staging on 2026-10-30.

### M4: safe to leave running (due 2026-11-22)

| # | Goal | Measure | Evidence | Ticket |
|---|---|---|---|---|
| 4.1 | The gate runs itself | `just ci` runs on every pull request, compiles Renki.Plugin (or fails), and has real lint for Go, panel, plugin and scripts | the workflow file and a red run on a planted error | TKT-01M6DCFST1 |
| 4.2 | A 24 h soak | `just vm soak` (TKT-01M6DDFTF6) runs 24 h with the bot idle-working at home and fails on any of these: a controller crash or restart; more than 100 open TCP connections; audit growth over 2 GB a day; a store write error; or a plugin `internal` failure | soak `summary.json` | TKT-01M6DDFTF6 |
| 4.3 | Stores survive a hard stop | 3 forced controller kills during saves leave memory.json, goals.json and settings readable, with no lost home | test plus a soak note | TKT-01M6DCFSHH |
| 4.4 | Only owners command it | Owners bind to player IDs. Lines from other players cannot steer an owner's turn, as shown by a test that replays the review's injection scenario | tests | TKT-01M6DCFSME |
| 4.5 | The link heals itself | After a 20 s controller pause the plugin reconnects and accepts tasks again within 10 s | bridge test plus one live pause | TKT-01M6DCFS8Y |
| 4.6 | Review closed | Every child of TKT-01M6DCF2AB is done, or has a note saying why it stands. The panel epic's criteria are ticked or moved | ticket status | TKT-01M6DCF2AB, TKT-01M628X106 |

### M5 to M7: direction, not yet SMART

These get measurable goals at the weekly review that follows the previous milestone's close, from evidence we do not have yet. For now:

- **M5, runs the base.** The base-upkeep epic's three criteria become goals measured over 2 h home soaks: fires stay lit, no food burns, assigned items reach their chests. An owner clones and edits a behaviour from chat and from the panel.
- **M6, agent with tools.** Run the voice-model comparison spike (TKT-01M630V80T) first, so we choose a model on data. Goals will bound tokens per hour, wake latency, and the share of proposals an owner rejects.
- **M7, explores and maps.** The goals in docs/EXPLORATION.md. Each expedition reports new observed coverage in m² and returns to its anchor within budget, and the atlas exports and imports one portable layer.

## How the goals are measured

- **Bounded trials** (`just vm gather-trial`, `combat-trial`, `harvest-trial`, `goal-trial`, `resupply-food`, `escort-check`, `route-benchmark`) write a `summary.json`. They always restore combat off and log out or stop. A goal counts *consecutive* passes, so a flaky pass does not close a milestone.
- **Owner sessions** cannot be supervised automatically. They are measured from the audit log with a report recipe, so the owner plays and the numbers come afterwards.
- **Offline tests** cover the failure branches (no chest, combat off, injection) that are slow or unsafe to stage live.

## Cadence

- **Every Sunday:** compare each open milestone against its goals and record the result as a note on its epic. If a date will be missed, re-date it here with one line saying why. Do not let it slip silently.
- **When a milestone closes:** tick its epic, close or move its children, and write the next milestone's SMART goals.
- **New work** lands in `draft`. It joins a milestone when a person promotes it, and is otherwise left out of the dates above.

## Risks

- **Live trials depend on the production world.** A death or a lost item costs real progress. Every trial already refuses to run without its safety preconditions, and M4's durability work protects the stores.
- **Work in flight drifts from the criteria.** The household-food track shows how much effort can go into useful work that is not on the milestone. At the weekly review, ask whether the active tickets serve the open milestone's goals.
- **Stale claims.** Several tickets still carry claims from 10-29 sessions that have ended. Release or re-claim them at the first weekly review.
