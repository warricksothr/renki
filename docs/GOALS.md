# Persistent goals and scripted recovery

A goal owns an objective across individual routine failures. A native task can
correctly refuse unsafe ground or lose a target while the objective remains
useful. Repeating the same task indefinitely is not recovery. The goal records
what happened, waits for relevant observations, selects an allowed alternative,
and eventually asks for a new plan when its recovery budget is exhausted.

## Layers and authority

Native C# routines own motion, aiming, combat, harvesting and immediate safety.
Go behavior runners compose those bounded tasks. The existing behavior engine
keeps exclusive body ownership, checks capabilities/permissions and honors new
orders. A persistent goal is an additional runner registered with that engine;
it composes existing runners and preserves a durable objective/checkpoint.
Starlark selects the next runner or bounded wait from a compact host snapshot.
It never controls Unity inputs or sends tasks directly.

The host owns success evidence, world/area limits, deadlines, attempt/task
budgets, permission checks and cancellation. Scripts cannot relax these. An
owner stop cancels the goal, including pending retries. Another body order
cancels the current goal continuation rather than allowing it to resume behind
the owner. A controller outage or disconnect suspends the objective. This first
slice requires explicit resume after reconnect, with fresh observations in the
original world and an idle body before a new attempt. Death and a different world require explicit resumption/replanning.

## Durable record and execution

`goals.json` beside `memory.json` stores a bounded set of records: identity,
objective/parameters, immutable policy source and SHA-256, explicit JSON state,
status, deadlines and remaining budgets, last outcomes and bounded history.
Definitions and checkpoints are independent of transient interpreter stacks.

The host saves an intent checkpoint before dispatching a step, and saves its
outcome before advancing policy. There is no atomic transaction between disk
and the game. After a crash, a pending intent is **uncertain**: do not replay
it or infer success. Reobserve/recover; if proof was lost, request a new plan.
A failed checkpoint write stops progression before another game action.
The behavior engine's cancellation hook updates the durable goal, including
when no next policy call is possible. Restart does not reset attempt budgets.

Policy runs are short, synchronous decisions. Each invocation receives a fresh
JSON snapshot, a JSON checkpoint and the previous structured outcome. It returns
one directive: run a named supported behavior with validated parameters, wait
for at most 30 seconds, request a new plan, or request completion. A completion
request succeeds only when the host's objective predicate has earned evidence.
Child runners and their internal native tasks remain bounded and cancellable.

## Starlark contract

The initial entry point is `step(context, state, outcome)`. Inputs are ordinary
Starlark JSON values. Output is a JSON-shaped dictionary containing `action`,
optional `template`/`params`, `state`, `waitS` and a short human `reason`.
The first version accepts decision trees, local assignments, JSON literals,
comparisons, indexing, `len` and dictionary `get`. Loops, comprehensions,
recursion, nested functions, arithmetic/allocation operators and other calls
are rejected before evaluation. This also prevents a single list multiplication
from bypassing interpreter step limits. Keys/indexes must be scalar literals;
comparisons require a scalar literal operand so collection expansion/comparison
cannot hide unbounded work inside one instruction. No module loader, filesystem, network, environment, process or game APIs are exposed. Source,
checkpoint/output size, evaluation steps and execution time are bounded. Policy
globals are rebuilt and frozen; persistence is exclusively the returned state.
A source hash identifies every decision. Policy revisions require explicit
validation and do not reset the original goal's host budgets.

## Initial combat-testing goal

The first objective is `combat_trial`: observe one native successful encounter
against an eligible ordinary Greyling while Tas remains healthy, within an
explicit world/area. Only the existing `combat` runner and bounded observation
waits are exposed initially. Fresh, observed targets within 30 m use native
survey/scouting/kiting; direct melee is an alternative only for a close confirmed attacker.
When fresh perception contains no live unsupported threat or unchanged eligible
target, the host can instead offer native local discovery. C# surveys ground,
scouts and selects the enemy under its existing bounds. An unchanged search
failure waits for materially changed observations rather than repeating on
advancing timestamps. Discovery success requires the final native task receipt
to bind its selected entity as well as report observed death.
The script can retry after fresh, materially changed observations, reacquire
another eligible target or wait for readiness. It cannot turn a refused task,
unloaded target or missing perception into a kill. The host retains the exact
successful attack task/target receipt as completion evidence.

The first goal has a ten-minute deadline, six attempts/native tasks, a
three-consecutive-failure limit and 120 seconds of total observation/recovery
waits. Waits are at most 30 seconds each. Budgets persist through resume. A
2 m position bucket, target identity/targeting state and readiness buckets
distinguish changed observations; advancing timestamps alone never enables a
retry. Host work scope is a 60–250 m sphere; new panel/trial goals explicitly use
200 m. Discovery receives up to 60 m of search radius, reduced by distance
from the goal center and a 52 m reserve for local combat/escape. Named-target
encounters keep the existing local envelope. The host never passes a search
radius below 12 m or expands an existing goal scope. Emergency native retreats may
leave that work sphere; the next attempt then requires a new plan. Recovery
never silently expands the authorized area or species. Native retreats are outcomes to inspect; low
health/stamina waits for recovery. Missing food/equipment, permission failures,
unsafe persistent conditions and exhausted budgets become `needs_plan` with a
compact diagnostic. There is no assumption that every world state offers a
safe fight or that reaching needs-plan proves melee works.

## Model escape hatch and generated patterns

The first version records `needs_plan` but does not call a model automatically.
Its context contains the objective, policy hash, compact current observations,
remaining budgets, distinct attempted recoveries and recent outcomes. Full
perception/audit streams stay in retained files. A later planner can propose a
revised script or supported behavior sequence using that bounded context.

Proposals must pass source/schema/capability validation and replay tests before
activation. Keep the prior policy/version and rollback on regression. A planner
cannot grant combat permission, alter a goal's scope, reset cancellation, invent
success evidence or create an unlimited retry loop. Native frame control remains
in C#. Human input is a last resort for a real missing resource or decision.

## Operator view and validation

The authenticated panel lists the objective, status, current step, attempts,
policy version, recent recovery decisions and needs-plan reason. Start/resume
and stop are explicit actions; stop ends the entire goal. Retained records remain
visible while the game is offline. The VM supervisor submits one persistent goal
and monitors it across failed child tasks instead of terminating at the first
routine failure. Cleanup cancels unfinished test goals, saves/logs out Tas,
stops the game, restores combat off and leaves the panel available.

Replay coverage must include recovery after failure, false/uncertain success,
restart with pending intent, preserved budgets, changed/world-stale observations,
owner takeover before streamed state catches up, permission failure, checkpoint
save failure, policy limits and shutdown. A live run exercises the orchestration;
only observed successful melee can satisfy the combat objective. M2 gathering
and hauling reuse the same contract later with inventory deltas/deposit receipts
as their host predicates, rather than trusting a script's claim of completion.

## Current API and testing

`POST /api/goals` uses panel authentication and `X-Renki-Act: 1`:

```json
{"action":"start","spec":{"kind":"combat_trial","world":"DizzysHellheim","center":[184,82,-289],"radius":200}}
```

It returns the durable record with its generated `id`. Subsequent commands are
`{"action":"resume","id":"..."}` and `{"action":"stop","id":"..."}`.
`GET /api/goals` lists records even offline. The panel button creates a goal
around the current bot position; it grants no combat permissions. Resume never
resets budgets, revives owner cancellation, or replays an uncertain intent.
At most 64 records are retained in this first slice; reaching the limit refuses
new goals instead of silently discarding evidence.

Policies are bundled and immutable per goal in this slice; there is no policy
upload/revision endpoint yet. A lost in-flight receipt requires a new goal and
fresh authorization rather than treating the old action as unfinished work to
repeat. Automatic reconnect continuation and model-generated revisions are
future additions. Goal state survives the interruption now, while continuation
is deliberately explicit.

[Authorable procedures and durable decisions](PROCEDURES.md) records the
follow-on research: general multistep child execution, a host-owned candidate
ledger, versioned user/model-authored scripts, replay and promotion into native
routines. The current runtime extracts one attack from a combat child; it does
not yet preserve arbitrary multistep children. Starlark remains the recommended
execution language for those extensions.

Use `just check controller test-packages ./internal/goal ./internal/app ./internal/behaviour`
for focused replay coverage and `just check vm goal-trial`
for a bounded supervised live run. The supervisor watches the goal through
individual native failures and cancels an unfinished goal during cleanup.

## Proposed exploration follow-on

After M2 and M3, [exploration goals](EXPLORATION.md) will extend this host with
coverage/frontier evidence, expedition preparation, bounded travel/survey and
return routines. The proposal is tracked as draft work; `combat_trial` remains
the only implemented goal kind. Map knowledge, observed coverage, visited ground
and revalidated routes are distinct evidence, and discovery plus verified return
will be required for the initial bounded land expedition.
