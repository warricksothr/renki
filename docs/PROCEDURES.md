# Authorable procedures and durable decisions

Research recommendation, 2026-10-30. This document proposes extensions to the
implemented [goal host](GOALS.md); it does not describe an available script
upload API or a generic durable behavior composer.

## Recommendation and current implementation

Keep the restricted Starlark interpreter already embedded in `internal/goal`.
Use named phases and guarded transitions to describe procedures, with one
synchronous `step(context, state, outcome)` decision at a time. Go owns durable
execution, candidate selection, evidence, permissions and budgets. C# owns
motion, timing-sensitive interactions and immediate safety. Users and models
can author the same versioned procedure format; a future visual editor can
compile to it.

For more granular C# responses and choreography, prefer parameterized native
routines plus a bounded declarative local sequence first. The native-side
comparison, dance example and explicit interpreter adoption experiment are in
[Local native policies](NATIVE-POLICIES.md). This is a separate task-level
extension under the same body ownership and safety authority, not another
long-horizon autonomy loop.

The concrete proposed interfaces, operating-agent workflow, feedback records and
implementation sequence are in [Procedure tools and feedback](PROCEDURE-TOOLS.md).
All interfaces in that companion are proposals. Existing goal and audit APIs
remain the available inspection surface; no new runtime is required.

The existing implementation is a useful first slice, but has concrete gaps:

- `internal/goal/policy.go` accepts a deliberately narrow decision-tree subset:
  conditions, local assignments, JSON literals, literal indexing, `len`, and
  dictionary `get`. It rejects loops, arithmetic, helpers and arbitrary calls.
- `internal/goal/runtime.go` handles `combat_trial` specifically. It constructs
  a combat runner, calls `Next` once and requires an attack. The child is not
  retained as a general multistep runner. Registering another template alone
  cannot make gathering or food resupply durable.
- `internal/goal/store.go` persists policy/state/budgets and refuses to replay
  uncertain dispatches. Reconnect continuation remains explicit.
- `internal/app/goals.go` exposes start, resume and stop. Scripts are bundled;
  there are no draft, upload, revision, replay or publication endpoints.
- `scripts/vm/resupply_trial.py` already demonstrates bounded chest search,
  updated contents and exact transfer proofs. It is a test supervisor, not a
  production persistent food goal.

## Options considered

| Approach | What it provides | Decision |
| --- | --- | --- |
| Restricted Starlark | Readable decisions, explicit JSON state, existing host and replay coverage | Extend the current runtime. |
| Lua | Mature embedding, coroutines, allocator and instruction hooks | No present requirement justifies a second interpreter; coroutine stacks would still need durable recovery. |
| Behavior trees | Sequence/fallback patterns and reactive guards | Support as authoring patterns with explicit running, cancellation and bounded retry semantics. |
| Statecharts | Named phases, guarded transitions, interruption/history | Use as the conceptual model; store the active phase in JSON. |
| HTN planning | Decomposes objectives into prerequisites and supported tasks | Consider later; needs maintained precondition/effect models. |
| Visual graph editor | Accessible editing of branches and recovery | Compile a restricted graph to the same Starlark contract; preserve node IDs and source maps. |

Starlark-Go's resource guidance says instruction limits alone do not bound
memory or expensive individual operations. The existing AST restrictions serve
that purpose. Keep them initially. Broader untrusted syntax should first move
to an evaluator worker with OS memory/CPU limits; adding a timeout inside the
controller is insufficient.

## Procedure package and host contract

A proposed package contains immutable source and hash, versioned objective and
state schemas, declared behavior capabilities, required permissions, a host
success-predicate identifier and replay scenarios. Pin each running goal to an
exact package version. Dependencies on named subprocedures must also be pinned.
The host must recognize every capability and success predicate; scripts cannot
supply executable validators or declare themselves successful.

`step(context, state, outcome)` stays synchronous and side-effect free. The host
provides only compact, relevant inputs:

- Current readiness and world/connection scope; observation age and provenance.
- Verified objective progress and a ranked next candidate or named alternatives.
- Remaining budgets and a bounded summary of distinct attempts.
- Structured outcomes with references to retained task receipts and traces.

Return a supported behavior request, bounded wait, needs-plan result or completion
request, plus explicit JSON state. Reuse the existing directive shape. Proposed
context fields such as `next_candidate` and `remaining_candidates` are data,
not new callable interpreter builtins. The current syntax cannot iterate over
an arbitrary list or increment a cursor. Let Go own the candidate cursor and
attempt ledger instead of expanding the language to solve that bookkeeping.
Host-ranked candidates can account for distance, freshness, access and prior
outcomes. Scripts choose between allowed actions; they cannot invent a target
or expand scope by modifying a host-provided parameter dictionary.

Generalize child execution incrementally with explicit reconstructible
host-controlled phases. There is no generic child checkpoint protocol today.
Define and test a stable checkpoint schema for each admitted adapter before
allowing a goal to use it; ordinary in-memory runners do not become durable
merely because the goal record is saved. Every native dispatch needs a saved
intent, validated parameters, scoped identity and a matched outcome receipt.
Account for every native task
inside a child, including retries, rather than charging once per child.

Distinguish running, bounded waiting, suspended, uncertain, needs-plan,
cancelled and succeeded; add partial progress only where its meaning is defined.
An owner stop or replacement order cancels the entire continuation. Existing
permissions, body ownership and native safety checks apply on every dispatch.

## Search outcomes and retries

An empty chest is an expected negative observation about one candidate, not
failure of the entire objective. Preserve the scope of that observation:
"requested food absent in chest A at observation revision X."

The existing combat failure counter compares the last reason. Reusing that
counter unchanged would treat three different empty chests as one repeated
failure. Track repetition by action, target and relevant observation fingerprint.
Keep global request, task, time, travel and model budgets separate. Timestamps
alone must not make the same unsuccessful action eligible again. Revisit a
candidate after meaningful changed evidence or an explicitly budgeted refresh.

Food resupply is the recommended first general procedure:

1. Check fresh carried supplies against the host objective predicate.
2. Rank allowed chests: fresh useful contents, remembered food hints, then other
   nearby authorized chests. Resolve current identities before use.
3. Approach and inspect through supported bounded interaction. Today's ordinary
   food probe already returns a contents snapshot when the item is absent.
4. Replace hoped-for contents with the fresh snapshot. Withdraw a bounded amount
   if available; otherwise record this candidate and continue to another chest.
5. Require a matched transfer receipt and fresh inventory/consumption evidence.
   Continue until enough supplies are proved or the overall budget is exhausted.
6. Report a specific shortfall if no allowed candidate remains. An inaccessible
   chest, uncertain identity and inspected empty chest are distinct outcomes.

Reconciled chest contents remain hints with their original source/time. A wrong
hint should cost bounded extra inspections. It must never prove current stock,
justify a fabricated transfer, or make a script claim the objective succeeded.

Gathering can use the same machinery for supplies, a carried tool, travel to an
approved worksite, harvesting, collection, deposit and return. Unsupported
crafting/repair remains a named unmet prerequisite. Exploration can substitute
frontier candidates and route evidence, reserve return resources and require
both new coverage and observed return; see [EXPLORATION.md](EXPLORATION.md).

## Recovery after interruption

Keep the current explicit resume policy while generalizing execution. An intent
without a receipt after restart is uncertain. Transfers and consumption are not
assumed idempotent. Reconcile against retained task receipts and fresh scoped
observations only when they prove the result; otherwise preserve uncertainty
and request a new plan. Do not blindly replay an action or infer that it failed.

Goals retain spent budgets through restart, explicit resume and policy revision.
A revised script must inherit original authority, objective scope, deadline,
cancellation and uncertainty. Revision cannot serve as a new retry allowance.
Checkpoint write failure stops progression before the next side effect.

## User and model authoring

Provide typed interfaces for capability discovery, draft creation, validation,
dry-run evaluation, scenario replay, publication, revision comparison and trace
retrieval. Give model authors the exact restricted grammar and valid examples;
ordinary Python examples encourage unsupported constructs. Separate draft
authoring, publication and operation permissions. Authorship grants no authority
to run a procedure, and publication cannot enable native combat or gathering.

A bounded planner can propose a revision when deterministic alternatives are
exhausted or failures expose a missing recovery. Its input should contain the
objective, allowed capabilities, relevant observations, distinct attempted
recoveries and remaining budgets. Its output is a draft or supported alternative.
Validate the source, schemas and capabilities, replay relevant scenarios, then
activate through the authorized publication policy. Retain the prior version
for rollback. A model cannot override refusals or invent success evidence.
Planner request/token/time budgets are durable and survive revision. Ask a human
when new authority or an unavailable resource is actually required.

Make the deterministic system useful to a coding agent before adding an
automatic planner. Capability discovery must say what is implemented, allowed
and ready; diagnosis must identify the decision and evidence behind a refusal;
draft/edit/validate/replay/compare must work offline without body authority.
Use immutable feedback records with separate observed facts, inferred causes
and proposed remedies. A model explanation is not new game evidence. The
companion design specifies bounded trace pages, structured validation errors,
content-pinned scenarios and publication/rollback boundaries.

## Evaluation and promotion into routines

Replay stale hints, empty boxes, partial transfers, new chest identities,
concurrent inventory consumption, inaccessible candidates, restart uncertainty,
checkpoint failures, cancelled goals and owner takeover. Record the package
hash, input fingerprint, state transition, chosen action, remaining budgets and
proof reference. Summaries should explain the relevant failed step without
loading an entire perception/audit stream into model context.

Recorded replay verifies policy decisions for recorded inputs. It cannot prove
what a different physical action would have caused. Keep native core checks and
bounded live trials for movement, collection, combat and other world effects.

Promote a reusable, stable composition into Go once varied successful/adverse
scenarios establish its contract. Move timing-sensitive world control into C#.
Keep the same evidence predicates and replay fixtures, and compare scripted and
compiled decisions over those scenarios. Promotion is a reviewed engineering
change, not an automatic translation after one successful run. Retain useful
scripts as higher-level compositions of the new routines.

## Delivery order

1. Preserve M2/M3 implementation priority. Add compact diagnostics and scenario
   capture around existing combat/food-supervisor behavior without enabling
   authoring, automatic revision or a new food goal.
2. Generalize the host with typed goal kinds, a capability registry, structured
   search outcomes and a durable candidate ledger. Admit one explicitly
   checkpointed food adapter at a time; preserve combat behavior. This supports
   M2 prerequisites but does not satisfy harvesting or M3 combat acceptance.
3. Add versioned packages and agent draft/edit/validate/replay/compare tooling,
   then authorized publication, bounded live canaries and rollback. Keep the
   automatic model planner disabled until these deterministic tools work.
4. Add bounded model proposals using those same tools and durable budgets.
5. Add a visual editor compiled to the same representation when authoring needs
   justify it. Add planning/decomposition only with explicit capability models.
6. Promote proven patterns into native routines while retaining replay contracts.

Draft follow-up tickets, in dependency order:

- TKT-01M6701B73BXT58GXFS0FYNZK8 — Generalize durable goals for multistep food search.
- TKT-01M6701PBVRTHD0SGRE45F42Q5 — Add versioned procedure authoring and replay.
- TKT-01M6701ZJS66ZY78FVG10TSWMC — Bound model-assisted procedure revisions and recovery.
- TKT-01M670QPEAJNPZ952E3R5VWWVF — Evaluate bounded native policies and choreography.
  This separate experiment waits on M2/M3 and may conclude that no additional
  interpreter is warranted.

## Primary sources

- [Starlark specification](https://github.com/bazelbuild/starlark/blob/master/spec.md)
  and [Starlark-Go resource bounds](https://github.com/google/starlark-go#bounding-time-memory-and-other-resources).
- [Lua 5.4 coroutine API](https://www.lua.org/manual/5.4/manual.html#4.7),
  [allocator](https://www.lua.org/manual/5.4/manual.html#lua_Alloc) and
  [instruction hooks](https://www.lua.org/manual/5.4/manual.html#lua_sethook).
- [BehaviorTree.CPP sequence implementation](https://github.com/BehaviorTree/BehaviorTree.CPP/blob/master/include/behaviortree_cpp/controls/sequence_node.h).
- [W3C SCXML statechart specification](https://www.w3.org/TR/scxml/).
- [GTPyhop hierarchical task/goal planning](https://github.com/dananau/GTPyhop).
- [Temporal activity idempotency](https://docs.temporal.io/activity-definition#idempotency),
  for the same external-side-effect retry ambiguity.
- [Voyager](https://github.com/MineDojo/Voyager) and
  [Code as Policies](https://code-as-policies.github.io/): precedents for
  model-authored executable skills composed from supported APIs and improved
  through feedback. Generated code and self-evaluation do not replace Renki's
  host evidence requirements.
