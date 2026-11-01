# Watching and explaining behavior

Design and first implementation, 2026-11-01. This document distinguishes existing inspection
from proposed tooling. It complements [Goals](GOALS.md),
[Procedures](PROCEDURES.md) and the proposed [procedure tool contract](PROCEDURE-TOOLS.md).
It does not introduce another planner or interpreter.

## Current surface

The authenticated panel already shows durable goals, status/reasons, attempts,
policy hash, recent task outcomes and food/return progress. Native combat status
includes the current mode and optional peel phase, tracked-member count,
pursuer/unknown IDs, quiet interval and planned escape endpoint. The activity
feed has task/chat history and a Plugin decisions filter. The shared map shows
perception, terrain, paths, obstacles and current world-event knowledge. Local
video shows motion; diagnostic camera and map snapshots retain selected views.

These are separate observations. There is no unified causal decision graph,
recorded decision replay UI, or procedure authoring/publishing API yet. Recent
goal summaries trim history; the audit is not a permanent scenario archive.

## Available first inspection slice

The panel's **Decisions** link opens a read-only decision inspector. It and
`just vm inspect [GOAL]` consume authenticated `GET /api/inspection`, optionally
`?goal=ID`. Omitting the ID selects the current lease-backed task's goal when
known, otherwise the newest saved goal. Choosing a goal or phase only changes
the view; it never resumes a goal, issues an order or acquires the body.

The `inspection-v1` response includes controller run, connection and world scope;
state arrival age/sequence; actual retained dispatch provenance; a compact saved
goal with budgets; typed guard observations/sources; phase descriptions and
selected possible transitions; and at most 12 retained intent/outcome/inventory
receipt rows. Goal choices use a lightweight index, and only one full checkpoint
is read. The response excludes policy source, full checkpoints and raw task
origins (which may contain private chat).

Ownership requires the current exact automation lease, matching behavior episode,
current-run pending intent and task identity. A newly dispatched owner order
supersedes the older streamed task immediately in the API, with `awaiting_state`
until its native observation arrives. Native recovery takes precedence. A saved
running goal by itself never establishes body ownership. Offline or more than
3-second-old world observations withhold native phases. Pursuit predicates expire
at 1 second. The panel also expires observations locally and invalidates live
presentation on scope changes or failed refreshes; it polls once per second while
visible. Food checkpoints remain explicitly saved evidence offline.

For peeling, necessary pursuit predicates can pass without granting permission
to fight. Full native preparation, independent native action permission,
competitor clearance and current escape-route validity stay **unknown** where
telemetry does not export them. A planned escape endpoint is not a checked
corridor receipt. Food search progress is a ledger of candidates, search entries
and verified receipts; one negative chest result cannot prove all storage empty.

This increment has a bounded evidence list, **not a causal timeline or replay**.
Only outcomes with recorded run/task/time receive scoped task-audit links. Old
records without a run or time stay unscoped; inventory receipt timestamps are
not invented. Combat and return intents now record run/time like food intents.
The graph describes possible transitions, never completed historical visits;
accepted legacy food phases are identified separately from current execution.
Map/video links open live views. Retained synchronized map/camera evidence and
material guard/decision archives remain subsequent work.

Verification uses `just check controller test-packages ./internal/inspection
./internal/session ./internal/app ./internal/goal`, `just check panel test`, and
`just check panel ui-test --scenario inspection-peel,inspection-interruption,inspection-pantry-offline
--viewport all --port 0`. UI scenarios use a clearly synthetic response to test
presentation; Go tests establish projection/provenance behavior. Neither proves
a live peel, kill, return or pantry deposit.

## One inspectable execution contract

Retain the existing authority layers:

1. A durable **goal** owns the objective, scope, progress and remaining budgets.
2. A **procedure** chooses bounded behaviors from host-authorized candidates.
   Restricted Starlark expresses named phases and guarded transitions.
3. A Go **behavior/adapter** coordinates supported native tasks and receipts.
4. A C# **routine** owns motion, timing-sensitive action and immediate safety.

A higher layer cannot override a lower safety refusal. Native recovery,
retreat or an owner command can own the body instead; the display must identify
the actual owner and interruption cause rather than infer ownership from a
goal's requested action.

Give each procedure/routine a versioned inspection description: intent,
prerequisites, required observations and freshness, phase names, legal
transitions and guards, allowed effects, success evidence, interruption and
recovery rules, and limits. Phase identifiers come from execution. The graph
is a projection of that contract; editing the diagram must not silently create
a second implementation of its transition rules. Initially use read-only
descriptions for hand-written native routines. A later visual procedure editor
can compile to the same restricted policy contract with node/source maps.

Keep implementation support, authorization and readiness distinct. For example,
an unsupported Greydwarf, disabled combat permission, and low stamina are three
different blockers with different remedies. Explain uncertainty explicitly:
an unloaded group member is not a confirmed disengaged enemy.

## Operator workspace

| View | Questions answered | Proposed content |
| --- | --- | --- |
| Current decision | What is Tas trying to achieve, and who controls him? | Goal → procedure phase → behavior → native task/phase; target; reason code/text; observed age; interruption owner; next check; remaining time/travel/attempts. |
| Phase graph | Where can this routine go next? | Active phase, completed transitions, waiting guards, escape/recovery edges and terminal outcomes. Distinguish planned edges from executed ones without relying on color alone. |
| Guard inspector | Why act, wait or refuse? | Actual value, required condition, pass/fail/unknown/stale, observation time/source and evidence reference. Show the authoritative refusing layer. |
| Decision timeline | What changed, and why did Tas change course? | Observation → decision → persisted intent → native action → matched receipt → checkpoint. Link interruptions and retries to their causes. |
| Map overlays | Where is the plan taking him? | Selected target/group members, pursuit trails with age, survey candidates/rejection reasons, selected ground, escape corridor, route, event bounds and observed obstacles. Label planned, measured and uncertain geometry separately. |
| Video and checkpoints | Does observed movement match the reported state? | Live video beside the map; timestamped diagnostic images linked to decisions. Historical scrub selects matching retained evidence and clearly labels gaps. |
| Evidence and comparison | Did it work, and did a revision improve it? | Kill/transfer/return receipts, candidate ledger, budget use, missing evidence and version comparison over captured scenarios. |

The default view should answer the current-decision questions in a few lines.
Expand guards and history on demand. Keep stop/cancel visible. A paused viewer
only pauses inspection; pausing or cancelling Tas is a separate authorized body
operation. Offline views retain historical intent and evidence with their ages,
never live-looking enemies, guards or pursuit certainty.

For peeling, a useful explanation would be: “Leading Greyling A toward surveyed
ground; B is still pursuing, so isolation has not started. Waiting for fresh
non-pursuit evidence and 24 m clearance for 3 continuous seconds. Escape remains
available; 18 s work budget remains.” This is illustrative text, not a field
observation. A lost member changes the guard to unknown; it cannot produce a
green isolation indicator. A native task reporting success still needs the
host's observed-death and requested return evidence before goal success.

## Shared data and agent tools

The panel and agent tools should consume the same compact explanation service.
Build toward the diagnosis, trace, evidence and capture contracts in
PROCEDURE-TOOLS.md rather than making agents scrape DOM text or read an entire
status/audit log. Include:

- Controller run, connection, world, goal, policy/routine version, decision,
  intent and native task identities. A task ID alone is not globally unique.
- Monotonic event sequence and explicit causal references. Wall-clock time is
  useful for display, but cannot alone order simultaneous events or join video.
- Stable phase/reason/guard identifiers, actual/required values, observation
  provenance and freshness, chosen action, allowed alternatives considered,
  and recorded rejection codes. Do not invent alternatives retrospectively.
- Actual body ownership, interruption/recovery cause, success receipts and
  remaining budgets. Model interpretations are labeled hypotheses, separately
  from facts reported by the host or native routine.
- Bounded pages and evidence references, not repeated embedded histories.
  Report retention gaps, redaction and truncation explicitly. An expired
  observation is not an empty observation.

Record decision/phase transitions and material guard changes; do not serialize
every physics tick. Native safety still evaluates at its required rate. A
rate-limited live snapshot can summarize progress between transitions. Pin
small scenario bundles before audit expiry when they explain a defect.

Existing video is not a frame-synchronized replay recorder. Correlate each
stream/frame using capture time and run/connection scope; expose lag and missing
frames. Use timestamped map/diagnostic checkpoints first. Continuous video
recording and scrub require a separate storage/retention design. Images support
diagnosis; they do not replace native task or inventory evidence.

Use structured events compatible with trace/span concepts. An OpenTelemetry
export can follow when useful; a tracing backend is not a prerequisite for the
panel or evidence service. Avoid sending every state frame to a model. Default
agent diagnosis should fit in one bounded response, with references for detail.

## Delivery order and validation

1. **Unify read-only explanation.** Start with peeling and pantry work. Expose
   current ownership, hierarchy, phase, guards, blockers and budgets from actual
   state. Mark fields unavailable until instrumentation exists. Add matching
   panel and compact agent representations before editable graphs.
2. **Retain decisions and evidence.** Add scoped IDs, causal transitions,
   bounded trace/evidence retrieval and scenario capture. Link timeline
   selection to map and camera checkpoints; keep recorded/live views distinct.
3. **Author and replay.** Versioned procedure packages, validation, recorded
   decision comparison, separate host simulation, bounded live canary and
   rollback. A changed action makes later recorded world outcomes unknown;
   replay does not certify a new live kill or deposit.
4. **Improve with feedback.** Humans and agent operators use the same
   diagnose/capture/edit/validate/compare tools. Pin running goals to versions;
   require explicit promotion for revisions. Distill stable procedures into
   native routines only after equivalent safety/evidence checks pass.

Initial inspection scenarios should cover a delayed second pursuer, unknown
ownership, blocked escape, event waiting, missing food in one chest with another
candidate remaining, owner interruption, offline/stale observations and restart
with uncertain intent. Assert reasons and evidence links against runtime state,
not merely screenshot text. Use existing panel fixture/screenshot tooling for
desktop/mobile presentation. Natural live trials separately establish actual
motion, safe escape, observed kills and completed returns/deposits.

No model escape hatch may broaden permissions, reset exhausted budgets or replay
uncertain effects. The explainable operator workflow must preserve those
properties while making a legitimate new plan easy to propose and test.
