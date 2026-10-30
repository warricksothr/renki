# Local native policies and choreography: research proposal

Design for review, 2026-10-30. The owner asked whether the C# plugin should
support more granular agent-authored behavior, including a dance. This document
recommends an experiment and adoption threshold, not an enabled interpreter,
new protocol task or expanded permissions. It complements long-horizon
[procedures](PROCEDURES.md) and their [agent tools](PROCEDURE-TOOLS.md).

Draft follow-up: TKT-01M670QPEAJNPZ952E3R5VWWVF — Evaluate bounded native policies
and choreography. It waits on M2/M3 and does not authorize live operations.

## Recommendation

Support dynamic local behavior first through parameterized native routines and
a small declarative, bounded sequence/statechart executor. The author should
choose a few supported actions and guarded transitions; C# should implement
their physical execution and safety. A finite dance can be data, without Lua,
JavaScript, C# compilation or a second general scripting language.

Keep an interpreter as a measured alternative. Adopt one only when at least
two useful, representative local behaviors cannot be expressed cleanly with
the finite action/guard format, and a pinned implementation passes compatibility,
resource, cancellation and diagnostic experiments. More expressive code should
solve a demonstrated authoring problem rather than replace difficult native
navigation or combat work. M2/M3 remain the priority; a successful dance is
neither harvesting evidence nor proof of melee readiness.

## Repository findings

These statements come from the checkout, rather than upstream assumptions:

- `Renki.Core` and `Renki.Plugin` target `netstandard2.1`. The plugin references
  BepInEx.Core 5.4.21 and builds against the installed game's Unity/Valheim
  assemblies. Those project targets do not establish the exact deployed Unity,
  Mono, platform or JIT/AOT version; inventory the deployment before choosing a
  package. Do not assume modern Unity `Awaitable` APIs exist in this game.
- `RenkiPlugin.Update` receives bridge messages, checks the dead-man condition
  and publishes observations. `Driver.OnFixedStep` calls `TaskRunner.FixedTick`
  in the patched `PlayerController.FixedUpdate`. `GameTaskHost` explicitly runs
  on Unity's main thread.
- `IBodyTask` has `Tick(double dt)` and `Cancel()`. `TaskRunner` holds one body
  task, admits supported/config/session-permitted tasks and handles replacement,
  stop, death, session end and timeout. Instant tasks may run alongside it.
  A task timeout is checked between ticks; it cannot interrupt a stuck Tick.
- `Driver` feeds ordinary game controls. It contains movement, look and attack
  flags, then clears one-shot inputs. `Stand` clears movement/attack flags;
  `ClearCombat` additionally clears look and queued combat inputs. New local
  actions must define their cleanup explicitly, not assume `Stand` resets every
  temporary effect or animation.
- Existing `Walker` supplies path following, support/water checks and bounded
  recovery. Movement variants use different native policy implementations;
  granting a script access to Driver would bypass important task-local checks.
- `SafetyConfig` gates movement/chat/gather/combat/raw controls separately.
  `emote` uses the chat gate; `look_at` uses movement. The native gate, advertised
  task capabilities and controller `allowedTasks` are distinct constraints.
  A new parent task cannot use its own admission to grant all child actions.

There is no local script runtime, sequence task, universal native danger
preemption wrapper or durable child-interpreter checkpoint protocol today.
Those must not appear as existing capabilities in discovery or documentation.

## Options and evidence

| Option | Advantages | Cost and limits | Initial decision |
| --- | --- | --- | --- |
| Parameterized C# routines | Existing architecture, static contracts, efficient ticks, core tests | New combinations may need code/deploy | Prefer for navigation, harvesting and combat mechanics. |
| Declarative bounded sequence/statechart | Agent-editable data, easy validation, explicit phases, no user callbacks | Maintain a deliberately small action/guard vocabulary | Prototype for choreography and local noncombat composition. |
| MoonSharp Lua | Managed C#, Unity-oriented, coroutine instruction yielding | Lua 5.2-oriented dialect, module/CLR sandbox work, allocation limits need evidence, native callbacks still uninterruptible | Candidate if finite composition is demonstrably inadequate. |
| Jint JavaScript | Managed interpreter; documented .NET Standard support and resource constraints | Larger language surface, version-dependent APIs/dependencies, cooperative host-call limits | Compare a pinned release only if there is a concrete reason to prefer JS. |
| Native Lua/another embedded VM | Mature ecosystem | Native binaries/ABI deployment and interop complexity; no durable-state advantage | No present justification. |
| Arbitrary C# via Roslyn | Full host ecosystem | In-process arbitrary CLR code is not an acceptable capability boundary; deployment/JIT complexity | Do not expose as a user/model policy mechanism. |
| Custom general bytecode VM | Can bound instructions/actions | A compiler, verifier and runtime become a substantial maintenance project | A tiny validated action IR is sufficient initially; do not build a general VM. |

[MoonSharp's own documentation](https://www.moonsharp.org/about.html) describes
managed .NET/Mono/Unity support and Lua 5.2 compatibility. Its
[sandbox guide](https://www.moonsharp.org/sandbox.html) explicitly warns that the
default module preset exposes system facilities. Start with no modules, no
automatic CLR object registration and no filesystem loader, then add only
necessary pure operations. Even `Preset_HardSandbox` is not proof of a CPU or
memory bound. Its [coroutine guide](https://www.moonsharp.org/coroutines.html)
provides `AutoYieldCounter` but explains restrictions around yielding across
CLR calls and warns about atomicity with automatic yielding.

[Jint's current project documentation](https://github.com/sebastienros/jint)
lists .NET Standard 2.0/2.1 support and bounded execution facilities, while
stating that constraints are cooperative and cannot preempt arbitrary host
callbacks. Current main-branch API examples are research leads, not a selected
release or guaranteed API on the deployed Valheim runtime. Verify package
version, dependencies, license and actual deployed compatibility in the spike.

[Unity's .NET and thread-safety guidance](https://docs.unity3d.com/2022.3/Documentation/Manual/overview-of-dot-net-in-unity.html)
and [execution-order documentation](https://docs.unity3d.com/Manual/ExecutionOrder.html)
provide general engine context. The actual checkout's main-thread task path is
the decisive integration constraint. Documentation for a newer Unity release
does not prove API compatibility with the installed game.

## Local execution contract

Treat a local policy as a bounded native task, not an independent body owner.
Go procedures select a named, pinned native task and receive one structured
outcome plus compact local trace. The plugin alone advances its substeps at
physics rate. A local sequence cannot dispatch bridge tasks recursively, grant
itself combat, override `Walker`, or continue after its parent is cancelled.

The initial executor belongs in `Renki.Core` as a pure state machine over a
small value-type observation/clock and typed intents. `Renki.Plugin` captures
observations on the main thread and validates/applies those intents through
approved native primitives. Expose no Player/GameObject/Transform, Driver,
reflection, file, network, console, module-loader or arbitrary CLR methods to
authored policies. Do not hide expensive raycasts, pathfinding or scans inside
apparently cheap getters; prepare bounded snapshots and expose freshness and
availability explicitly.

Suggested first input fields: sequence tick/elapsed time, valid ownership,
same world/character, grounded/dry state, observed local threat flag, HP/stamina
buckets, current primitive status/reason and measured displacement. Native
sensors compute them; a script does not discover arbitrary entities itself.
Unavailable danger/surface evidence blocks motion instead of becoming "clear."

First action vocabulary: stand, bounded face-relative-heading, one catalogued
emote, wait and a small walk offset through the existing native navigation
facade. Defer attack, jump/dodge, inventory mutation, interaction and raw control
flags; each would need its own validated parameter, safety and evidence contract.
Even movement offsets need supported arrival/floor validation, not projection
of a raised-floor start onto terrain. A choreography request does not authorize
leaving a work area, traversing water or operating doors it otherwise cannot use.

For the declarative format, bound source bytes, parse depth, number of states,
transitions and actions, maximum duration, cumulative movement and per-action
range. Validate finite acyclic sequences first; later cycles need explicit
host-owned iteration caps. Expressions are a fixed set of typed predicates and
scalar comparisons, not injected code strings. Check all reachable actions and
guards before admission. A failed guard ends or selects an explicitly declared
bounded alternative; it never silently enables a lower-level control path.

An interpreter, if adopted, should follow the same snapshot-to-intent facade
and explicit state contract. Do not execute arbitrary host callbacks from a
physics tick. Pure evaluation could run on a worker with bounded copied inputs;
only a timely result for the exact task/epoch/tick is eligible for main-thread
application. Discard stale results. An in-process worker cannot reliably stop
arbitrary blocked managed/native code; hostile broader code requires a separate
process. A separate process adds latency and is usually unsuitable for a
physics-rate control loop. This is another reason to prefer the finite executor.

## Budgets and preemption

Suggested experiment targets, not current settings or guaranteed limits:

- At most 16 states/32 transitions, 8 KiB authored data, ten seconds duration,
  two metres total commanded walking and a one-metre radius from the anchor.
- At most one action transition and one emitted intent per physics tick; excess
  elapsed time does not execute a backlog of several moves in one tick.
- Target zero steady-state executor allocation after admission; preallocate
  bounded buffers and report admission/transition telemetry allocation separately.
- Measure incremental p50/p95/p99 and worst-case time on the actual bot VM.
  Initial target: p99 below 0.25 ms per executor tick and no observed tick above
  1 ms in the declared corpus. Calibrate against physics interval/baseline load.
  These measurements are release thresholds, not preemptive hard deadlines.

Hard structural bounds must make the executor's own work finite. A stopwatch
can detect an overrun after a call returns; it cannot keep Unity responsive
while that call blocks. Interpreter instruction/time budgets likewise do not
bound a long native call. Expensive native primitives keep their own bounded
incremental execution and cancellation, rather than running synchronously to
completion inside a scripting callback.

Before evaluating/applying each tick, check parent ownership, stop/replacement,
session/dead-man state, death/world change, resource freshness, local danger and
physical constraints. Exact threat readiness and priority rules require a
specific native contract; they are not already supplied by TaskRunner globally.
On refusal, timeout or cancellation, neutralize all owned inputs and temporary
look/animation state that can safely be cancelled. Do not restore stale prior
controls. Return task/epoch, phase, reason and evidence of last applied action.

Danger must preempt optional choreography immediately at the next guarded tick;
an existing permitted native safety routine may then own the body. Do not resume
the dance automatically when danger disappears. Existing owner orders, explicit
stop and session revocation remain decisive. Cleanup and cancellation latency
must be measured at the Driver application boundary, including queued inputs.

## Example: a six-second dance

The following is proposed authored data, not an accepted protocol payload.
Durations are maxima for the phases, not promised animation timings.

```json
{
  "formatVersion": 1,
  "id": "greeting_shuffle",
  "contract": "local_sequence.v1",
  "maxDurationMs": 6000,
  "anchorRadiusM": 1,
  "maxTravelM": 2,
  "requires": ["movement", "emote"],
  "guards": ["living", "dry_supported_surface", "no_observed_threat", "parent_owns_body"],
  "steps": [
    {"id": "greet", "action": "emote", "name": "wave", "maxMs": 1200},
    {"id": "left", "action": "walk_offset", "rightM": -0.4, "forwardM": 0, "maxMs": 1000},
    {"id": "face_left", "action": "face_relative", "yawDeg": -30, "maxMs": 600},
    {"id": "right", "action": "walk_offset", "rightM": 0.4, "forwardM": 0, "maxMs": 1000},
    {"id": "face_front", "action": "face_relative", "yawDeg": 0, "maxMs": 600},
    {"id": "settle", "action": "stand", "maxMs": 500}
  ],
  "onFailure": "stop_and_report"
}
```

Define offsets/headings relative to the captured start anchor and heading, not
the progressively rotated body. This prevents frame-dependent coordinate drift.
Check the entire swept local path and each native step, including elevation,
support, nearby people, obstacles, stamina and water. Validate `wave` against the
deployed emote catalogue; reject unknown emotes rather than invoke an arbitrary
animation method. Static maximum times sum below the overall deadline; actual
completion requires native observations and may be earlier or fail.

The initial safe experiment can omit all walking and use stand/look/emote only.
The moving variant needs the validated local-motion adapter. A game API accepting
an emote command proves command acceptance, not that a viewer saw the entire
animation. Separate commanded phase completion from observed displacement,
heading and any available animation confirmation. Do not market an unobserved
animation as host-verified successful choreography.

Example final diagnostic for an interrupted dance:

```json
{"taskId":"t-9","controllerRunId":"run_44","localPolicyVersion":"lp_shuffle_2","status":"cancelled","phase":"left","reason":"danger_preempted","lastAppliedTick":183,"observedTravelM":0.2,"inputsCleared":true,"traceRef":"ev_local_9","coverage":{"phaseTrace":"complete","animationConfirmation":"unavailable"}}
```

A tactical alternative is “if the current safe native approach is refused,
report a suggested alternate local offset.” It still needs a verified motion
adapter. Scripts cannot make an unreachable approach safe merely by choosing
more granular controls. Native combat and harvesting parameters should remain
bounded routine parameters until their core mechanics are established.

## Agent diagnosis and tuning

Reuse the proposed procedure tools with a `local_policy` artifact kind and
plugin capability/version information. Discovery advertises implemented action
and guard schemas, actual measured limits and whether an operation requires a
game connection. No separate implicit script upload or raw-control endpoint.

The agent needs: current phase and guard values, first rejected intent, refusal
layer, snapshot age, primitive/native reason, measured displacement, duration,
per-tick timing/allocation counters and compact transition history. Retain
source node IDs, policy hash/version, parent goal/decision, task/run/epoch and
evidence refs. Report trace coverage per class: retained, unavailable, truncated
or expired. An empty map must not mean both "zero failures" and "not collected."

A current M3 observation illustrates why: a bounded scout reported no eligible
Greyling with zero observed movement. Exact task traces contained 24 rejected
frontiers: 21 `no_complete_navmesh_route`, three `wet_or_unloaded_terrain`.
The compact stdout projection printed an empty scouting object despite those
retained traces; the full summary retained reasons/examples under
`encounters[].ground.scouting`. This was an incomplete diagnostic projection,
not lost underlying evidence. A separate constructor/start-mode trace lacked
task identity because it ran before the current task was assigned; rejected
frontier traces were correctly task-stamped.
These are observations from the root agent's 2026-10-30 live trial, not a new
benchmark run by this research. The local retained evidence was
`.checks/scout-task-reconciliation-check.json`; ignored evidence is not a
portable fixture until captured/redacted through an explicit scenario bundle.
The attack ticket owns the implementation follow-up.

Correct diagnosis is “no tested scout frontier was admitted; enemy presence was
not established,” with native rejection counts and coverage gaps. It is not
“no enemy exists” or automatically a policy defect. A high-level policy might
choose an already permitted travel-to-search-start phase. A raised-floor
navigation inconsistency instead needs a native regression fixture and code fix;
Lua cannot repair missing sensor or navigation capability. Preserve those two
remedies separately in agent feedback.

An offline local-policy replay supplies recorded snapshots and virtual ticks to
the pure executor. Test action/guard transitions, deadline bounds and cleanup
intents. A different move changes later physical state: stop causal replay at
the first divergence, just as in Go procedures. Simulated terrain/primitive
outcomes are labeled synthetic. Unity admission and actual motion need separate
bounded in-game evidence; no model call belongs in a FixedTick path.

## Version replacement, crash and promotion

Compile/parse/validate outside the physics tick and pin an immutable policy
version plus capability contract for each admitted native task. Do not replace
instructions under an executing coroutine. Tuning creates a new version for
the next task, or explicitly cancels the old one and starts a separately
authorized task after a fresh readiness check. Epochs invalidate late results.
Rollback changes future selection and cannot rewind movement or emotes.

Do not persist interpreter stacks. Short local actions are ephemeral and stop
on session loss/restart. The durable Go goal retains the pending intent and
applies its existing uncertain-action rules; it never replays a half-finished
dance or claims completion from its last phase. A new dance can start from a
fresh anchor after authorization. Any future resumable local task needs a typed
checkpoint/proof contract and per-phase idempotency analysis first.

Promote learned useful behavior into reusable parameterized native routines
when its contract and limits stabilize, measurements justify it, varied adverse
fixtures pass and live evidence confirms the physical behavior. Keep the
reference artifact and compare decisions/cleanup before replacement. Learning
can propose parameters or sequences; it cannot expand approved action bounds,
disable refusal guards or train against fabricated success. Cosmetic preference
feedback and physical safety/evidence metrics remain separate.

## Experiment and adoption gate

1. Inventory the deployed Unity/Mono/BepInEx/runtime, packages and target OS.
   Baseline existing task tick time/allocation and cancellation behavior.
2. Specify a pure finite sequence executor and action/guard contracts. Build
   stand/look/emote first, then a guarded sub-metre motion adapter only if
   needed. Implement ordinary Core tests before any game-dependent wrapper.
3. Use the same corpus for parameterized native code, finite authored data and
   a pinned interpreter prototype: valid dance, long loop, large allocation,
   malformed/deep input, unknown action, stale snapshot, refused movement,
   cancellation at every phase, owner replacement, world/death/session changes,
   delayed worker result and a deliberately slow host callback.
4. Benchmark admission separately from steady ticks and transition telemetry.
   Verify no queued input remains after cancellation, no late result applies,
   no forbidden CLR/system API is reachable and state remains bounded. A slow
   host callback must demonstrate the preemption limitation, not pass because a
   timeout eventually reports it. Reject any architecture needing that callback
   inside an untrusted tick.
5. Conduct a separately authorized safe stationary live canary, then bounded
   walking only after surface/path validation. Preserve full evidence and
   restore the prior native capability configuration during cleanup.
6. Adopt an interpreter only if the two representative expression gaps exist,
   it materially improves agent authoring/diagnosis, all bounds/compatibility
   checks pass on the actual VM and its maintenance/security cost is acceptable.
   Otherwise ship parameterized routines plus declarative data and retain the
   decision. No general interpreter is a valid experiment outcome.

Use existing `just check plugin test-core <class-or-test-name>` for pure checks
and `just check ci` for the repository gate. Add any repeated policy benchmark
or corpus runner as a plugin just recipe before using it. Game-dependent plugin
builds and bounded live trials must report unavailable local game assemblies
honestly; a passing Core suite alone cannot validate Unity integration.
