# Preparing replaceable movement steps

Status: proposed design, not an implemented movement queue. Tracked in
TKT-01M674C36AET3YJMR7AFQZB6AD — Design cancellable look-ahead between movement
steps. The existing procedure/native-policy research in draft PR #92 remains
separate. Start with native scouting, where one task already owns the controls.

## What caused the observed pauses

The retained persistent scouting trial on installed `b536179` covered 227.9 m
across three native attempts without damage. Its audit is
`.checks/audit-window-0hjdQA.jsonl`; its bounded summary is
`.checks/combat-trial-20261030T192119356202Z/summary.json`.

`KiteTask.Scout` calls `Scouting.Decide` every 0.5 seconds. A rejected candidate
returns `Wait`, clears `walking`, and leaves the next candidate for the next
half-second decision. This makes rejection streaks visible as stationary waits:

| Native attempt | Rejected candidates before next accepted route | First rejection to next route |
| --- | ---: | ---: |
| t-1 | 4 | 2.003 s |
| t-2 | 8 | 4.005 s |
| t-3 | 14 | 7.020 s |

These are measured event-to-event delays, not estimates of CPU cost. In the
same trace, failed-result to next controller dispatch was 0.091 s and 0.164 s.
The one recorded local-route search reported 27.4 ms total and a 20.07 ms worst
slice. None of these numbers establishes a typical cost across other terrain.
The two-second initial search observation window in `Scouting.Decide` is
intentional. It is distinct from candidate rejection and route loading.

At ordinary leg boundaries, the half-second decision cadence can also let the
Walker reach its endpoint before the next decision. State frames can help
locate these waits but do not reveal exact control-idle time. Instrument native
arrival/ready/commit times before claiming a latency improvement. The present
trace does not justify removing checks or increasing planning work without a
budget.

## One provisional slot

The current task retains ownership of movement. Give it one replaceable slot:
`empty -> preparing -> ready -> committed`, with `discarded` possible from
preparing or ready. Replacing the slot cancels its prior generation. There is
no FIFO of future commands and no autonomous execution after the task ends.
Preparation creates data; only a validated commit may change controls.

A prepared step carries:

- World UID, connection/session epoch, native task ID and task generation.
- Owner/control revision; goal/policy generation if a controller goal supplied it.
- Current-leg generation and predicted grounded endpoint/start region.
- Candidate index, target identity, intent (outbound/return), route and support
  samples, relevant obstacle/tile revisions and observation timestamps.
- Creation time, expiry, planning work used, and the reason it was discarded.

Keep one candidate search cursor rather than repeatedly starting at candidate
zero. Preparation must not increment completed-leg counts, reset travel/time
budgets, consume a success receipt, or update the active Walker's stuck state.
Candidate evaluation counts against the existing finite candidate/work budget;
cancellation must not turn the slot into unlimited retries.

## Prepare while the current leg finishes

Prepare the next short scouting leg near the end of the active leg, initially
within roughly two metres or one second of predicted arrival. Evaluate from
the predicted endpoint using a separate read-only planning state. Continue
following the current checked leg while preparation runs. If several candidates
are rejected, continue their evaluation across render/physics slices instead
of paying a half-second observation delay per rejection.

Unity physics, navmesh queries and scene object access stay on the game thread.
Use incremental work with a shared planning allowance: initially at most eight
probes and a two-millisecond soft slice, yielding to current-route validation,
stop/retreat and control updates. A single engine call can exceed a soft time
allowance, so record actual worst slice/call cost and lower probe counts when
needed. Speculation must not double the current planner's frame budget.

Return travel, stamina and health take precedence over outbound speculation.
An unloaded future tile leaves the slot unready; it does not authorize a
straight-line fallback. If nothing is ready at arrival, stop safely, finish
bounded planning and report why the handoff waited.

## Commit and displacement

On actual arrival, require matching authority, world/session/task/leg and target
identities, fresh observations, unchanged relevant geometry, and an unexpired
slot. Check the actual grounded pose against the prepared start region and
validate the connector from that pose to the first route segment. Recheck
current support, immediate clearance, characters, water, stamina and retreat
conditions using existing policies. If the checks pass, install the route and
steer within the same tick. Reuse route data through a dedicated adoption path;
calling today's `Walker.SetGoal` blindly would throw it away and plan again.

Stop, disconnect, world change, owner takeover, retreat, a new enemy, target
replacement or changed geometry invalidates the slot immediately. Stop and
retreat never wait for preparation to finish. A delayed planner result must
compare its generation before publishing, so it cannot resurrect a discarded
step. A new urgent action may replace the proposed step, but still needs its
own current safety checks.

Smoothness comes first from avoiding empty control ticks between validated
legs. Any later heading/velocity blending must stay inside the checked corridor,
respect sharp turns and floor changes, and preserve tasks that intentionally
stop at an interaction point. Preparing an attack, chest mutation or item use
does not authorize executing it early. Controller procedures may later prepare
arguments, but dispatch must retain existing leases and verified prerequisites.

## Implementation and verification order

1. Add compact transition events: arrival, preparation started/ready, candidate
   rejection, committed/displaced, wait reason, plan age, probes and slice cost.
   Retain full events in audit; summarize counts, delay percentiles and a few
   examples through a `just` tool rather than printing the whole trace.
2. Add a pure native slot/generation policy with tests for supersession, stale
   results, expiry, changed target/scope/authority and bounded work.
3. Integrate candidate preparation into scouting only; keep current movement
   and safety checks authoritative. Test a rejection streak, unloaded terrain,
   connector changes, structure changes and immediate stop/retreat while ready.
4. Verify on an autonomous route without owner staging. Compare arrival-to-next
   steer and idle time, frame cost, rejected/wasted speculation, travel and all
   stop/retreat outcomes. Report intentional holds separately. Require no stale
   step execution and no authority or safety regression before enabling by
   default; then consider ordinary waypoint chains and procedure handoffs.

No live smooth-handoff result is claimed yet. This design explains the observed
candidate waits and gives the next implementation a measurable target.
