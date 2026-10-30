# Procedure tools and feedback: proposed contract

Design for review, 2026-10-30. This extends [PROCEDURES.md](PROCEDURES.md).
Every `procedures.*` tool and `/api/procedures` route below is **proposed**.
Examples describe a future installed food adapter, not capabilities available
today. Local tactical authoring is assessed separately in
[NATIVE-POLICIES.md](NATIVE-POLICIES.md).

## What an operating agent needs

Start with four questions: what was the objective, what did the host actually
observe, why did it choose this action, and what different allowed action can
be tested? A useful diagnosis fits in one response. It should expose a failed
decision and its inputs before asking an agent to retrieve native traces.

The minimum usable loop is discover, diagnose, capture, edit, validate, replay,
compare and propose publication. None of these authoring operations needs body
control. Starting a live canary does. Build this loop for human-driven coding
agents first; automatic model calls are a later consumer of the same service.

Today `GET /api/goals` and `POST /api/goals` expose bundled combat goals and
start/resume/stop. `GET /api/task?id=...&at=...` scopes task IDs to a controller
run, while `/api/timeline` pages a day's audit using time and an equal-time
skip count. These remain existing APIs. Goal history is bounded to 16 outcomes;
audit retention defaults to 14 days and can be disabled. Neither is a durable
scenario archive. Never present missing historical evidence as an empty result
that proves nothing happened.

## Best practices applied

| Primary source | Relevant practice | Renki application |
| --- | --- | --- |
| [OpenTelemetry trace API](https://opentelemetry.io/docs/specs/otel/trace/api/) | Named operations, timestamped events, attributes and causal links | Correlate goal, decision, child step and run-scoped native task; retain causes instead of only text messages. A new telemetry backend is optional. |
| [W3C PROV overview](https://www.w3.org/TR/prov-overview/) | Distinguish entities, activities and responsible agents | Separate an observation, a reconciliation, a policy decision and a model suggestion; retain derivation and attribution. No RDF implementation is required. |
| [HTTP If-Match](https://www.rfc-editor.org/rfc/rfc9110.html#section-13.1.1) | Conditional mutation prevents lost updates | Every draft edit, channel change and goal revision uses a strong revision precondition. |
| [RFC 9457](https://www.rfc-editor.org/rfc/rfc9457.html) | Machine-readable problem details with extension fields | Give stable error codes, source locations, relevant evidence and an actionable correction. |
| [Temporal activity idempotency](https://docs.temporal.io/activity-definition#idempotency) | External side effects and retries can be ambiguous | Do not replay uncertain transfers or food consumption; retry-safe API submission is different from retry-safe game execution. |
| [Google SRE canarying](https://sre.google/workbook/canarying-releases/) | Use representative metrics and compare with a control | Compare by scenario/readiness cohorts, constrain exposure and stop on safety/evidence regressions. A single bot's sequential trials are not a randomized controlled experiment. |

These are design applications, not claims that Renki already implements the
standards or that a tracing library can establish game truth.

## One package and one authority boundary

An immutable package identifies source, manifest, objective/state schemas,
required capability contracts, host predicate and pinned dependency versions.
Store its digest over a documented canonical manifest plus source bytes and
referenced content digests. Display names such as `food-search` are aliases;
execution pins an immutable `versionId`. Include interpreter build, grammar
profile and host capability/predicate versions in validation and replay runs.

Separate package requirements from the authorization attached to a goal.
Publication makes a package selectable. Starting a goal supplies an authorized
world, area, objective, native permissions and budgets. A package cannot grant
any of them. Each native dispatch rechecks current permissions, body ownership
and safety; an earlier validation receipt is not a permission lease.

Keep the current restricted `step(context, state, outcome)` and directive
shape. The sample source below uses only currently allowed syntax; the context
fields and adapter names are proposed. The host computes success and supplies
the next allowed action, including its validated parameters. It updates its
candidate ledger from outcomes before invoking the policy again.

```python
def step(context, state, outcome):
    if context["objectiveSatisfied"]:
        return {"action": "done", "reason": "host has supply proof", "state": {"phase": "complete"}}
    if context["blocked"]:
        return {"action": "needs_plan", "reason": context["blockedReason"], "state": state}
    candidate = context.get("nextAction")
    if candidate == None:
        return {"action": "needs_plan", "reason": "allowed search candidates exhausted", "state": {"phase": "shortfall"}}
    return {"action": "run", "template": candidate["template"], "params": candidate["params"], "reason": "inspect or withdraw from the next allowed candidate", "state": {"phase": "search"}}
```

The host's food adapter may use a bounded ordinary-food withdrawal probe that
returns actual contents even when the requested item is missing. Do not pretend
there is already a dedicated chest-inspection task. Whether to add a read-only
inspection primitive is a separate protocol/native design decision.

## Tool and transport conventions

Expose typed tools over one controller service; HTTP handlers and a future
agent connector must call the same implementation. Proposed names map below.
Read operations use GET; offline computations use POST with no body authority;
mutations use authenticated POST/PUT. Preserve panel authentication and
`X-Renki-Act: 1` for writes. Agent integrations authenticate independently of
source authorship; they must not scrape browser credentials.

| Tool | Proposed HTTP route | Effect |
| --- | --- | --- |
| `procedures.capabilities` | `GET /api/procedures/capabilities` | Discover schemas, grammar, adapters, permissions and readiness. |
| `procedures.diagnose` | `GET /api/procedures/goals/{id}/diagnosis` | Compact evidence-linked diagnosis. |
| `procedures.trace` | `GET /api/procedures/goals/{id}/trace` | Bounded structured trace pages. |
| `procedures.evidence` | `GET /api/procedures/evidence/{id}` | Retrieve one authorized, bounded evidence object. |
| `procedures.draft` / `procedures.edit` | `POST /api/procedures/drafts`; `PUT /api/procedures/drafts/{id}` | Create or conditionally replace a draft revision. |
| `procedures.validate` | `POST /api/procedures/validations` | Parse, schema, capability and compatibility checks; no dispatch. |
| `procedures.capture` | `POST /api/procedures/scenarios` | Preserve a bounded scenario from retained evidence; no new sensing. |
| `procedures.replay` | `POST /api/procedures/replays` | Execute the pure policy and supported host model offline. |
| `procedures.compare` | `POST /api/procedures/comparisons` | Compare pinned versions on identical pinned inputs. |
| `procedures.feedback` | `POST /api/procedures/feedback` | Append attributed evidence-linked feedback; no execution. |
| `procedures.publish` | `POST /api/procedures/publications` | Publish an immutable validated package. |
| `procedures.canary` | `POST /api/procedures/canaries` | Operate explicitly authorized, bounded live goals. |
| `procedures.rollback` | `POST /api/procedures/rollbacks` | Change a channel's version under a revision precondition. |

All results include `schemaVersion` and `requestId`; IDs below are illustrative
opaque IDs. Mutation responses return object/revision IDs; read the object when
its complete content is needed. Immutable validation/replay jobs are fetched
through `GET /api/procedures/jobs/{id}`; creation returns HTTP 202 and a job ID
when not completed synchronously. A disconnected job client does not start a
second job. Configure bounded job concurrency, CPU, memory, storage and TTL.

Return strong `ETag` headers on mutable resources. Require `If-Match` for edits,
channel changes and goal revision activation; missing preconditions return 428,
stale ones 412. All POST submissions require a principal-scoped
`Idempotency-Key`, retained for a documented TTL (initial proposal: 24 hours).
The same key/body returns the original result; changed body returns 409. After
expiry, look up the prior operation before resubmitting an operation with live
effects. This deduplicates API requests, not native actions after a crash.

Reads default to compact fields and never inline complete perception streams,
policy source or inventory tables unless requested. Initial proposed bounds:
32 KiB compact responses, 64 trace events per page, 256 KiB expanded evidence,
16 KiB source/state as today, and eight evidence references per diagnostic.
Advertise actual limits through capabilities; return counts and truncation,
not a silently incomplete JSON value. Larger evidence is retrieved in bounded
pages through scoped IDs, never arbitrary paths or URLs supplied by scripts.

## Capability discovery

`GET /api/procedures/capabilities?goalId=g17` returns a goal-scoped view:

```json
{
  "schemaVersion": 1,
  "requestId": "req_01",
  "catalogVersion": "cat_7",
  "grammar": {"profile": "renki-decision-tree-v1", "sourceMaxBytes": 16384, "loops": false, "dynamicIndexing": false, "arithmetic": false},
  "adapters": [{"id": "food_probe.v1", "implemented": true, "durablePhases": true, "permitted": true, "ready": true, "paramsSchemaRef": "schema_food_probe_1", "requiredPermissions": ["withdraw", "movement"], "successPredicate": "food_supply.v1"}],
  "goal": {"id": "g17", "authorityRef": "auth_17", "world": "DizzysHellheim", "canRevise": false},
  "tools": {"draft": true, "publish": false, "canary": false},
  "limits": {"tracePageEvents": 64, "compactBytes": 32768, "expandedEvidenceBytes": 262144}
}
```

Return unavailable adapters with `implemented:false`, or explain why a known
adapter is not permitted/ready using stable reason codes and required evidence.
Do not conflate missing implementation, missing authority and missing food.
Schema references resolve through bounded reads under the same service. Actual
permission names must map to existing controller/native gates; aliases in this
proposal do not add permissions. A current implementation would list only
combat as a durable goal adapter, not the future food entry shown here.

## Diagnosis, traces and evidence

`GET /api/procedures/goals/g17/diagnosis?view=compact`:

```json
{
  "schemaVersion": 1,
  "requestId": "req_02",
  "goalId": "g17",
  "goalRevision": "gr_9",
  "versionId": "pv_food_01",
  "status": "needs_plan",
  "objective": {"predicate": "food_supply.v1", "requiredCount": 4, "verifiedCount": 0},
  "diagnosis": {"code": "candidate_negative_terminated_search", "confidence": "inferred", "summary": "Chest A lacked the hinted Raspberry; the policy stopped while one allowed chest remained.", "decisionId": "d12", "source": {"line": 6, "phase": "search"}},
  "facts": [{"code": "requested_item_absent", "targetRef": "chest_A", "evidenceRef": "ev_receipt_12"}, {"code": "candidate_remaining", "count": 1, "evidenceRef": "ev_context_12"}],
  "remaining": {"nativeTasks": 2, "waitS": 40, "deadlineAt": "2026-10-30T19:00:00Z"},
  "uncertainIntents": [],
  "nextTools": [{"tool": "procedures.trace", "goalId": "g17", "aroundDecision": "d12", "limit": 8}, {"tool": "procedures.capture", "goalId": "g17", "decisionId": "d12"}],
  "coverage": {"decisionsComplete": true, "nativeEvidenceComplete": true, "missingRefs": []}
}
```

The diagnosis is a hypothesis derived from separately retrievable facts. A
deterministic classifier can establish that a policy returned needs-plan with
remaining candidates; a model's explanation of its intent remains an inference.
Return `unknown` when evidence cannot distinguish a wrong policy, wrong host
context or missing receipt. Include the authoritative refusal layer and
relevant predicate values; avoid generic advice such as "retry with more food."

`GET /api/procedures/goals/g17/trace?aroundDecision=d12&limit=8`:

```json
{
  "schemaVersion": 1,
  "requestId": "req_03",
  "snapshotId": "ts_44",
  "events": [{"seq": 12, "decisionId": "d12", "at": "2026-10-30T18:50:00Z", "controllerRunId": "run_44", "connectionId": "conn_2", "taskId": "t-6", "kind": "policy_decision", "phase": "search", "inputRef": "ev_context_12", "stateBeforeRef": "ev_state_11", "directive": {"action": "needs_plan", "reason": "hinted food missing"}, "stateAfterRef": "ev_state_12", "causeRefs": ["ev_receipt_12"], "versionId": "pv_food_01"}],
  "nextCursor": "cursor_44_13",
  "hasMore": true,
  "truncated": false,
  "missingEvidenceCount": 0
}
```

Continuation uses only `cursor` and an optional smaller limit. A cursor binds
principal, goal, filters, stable ordering and a high-water sequence; new events
do not shift a paginated snapshot. Order by persisted sequence, not wall clock
alone. Return 410 `trace_cursor_expired` if the snapshot is gone, with a fresh
bounded query suggestion. A full forward trace query and `aroundDecision` are
distinct modes. Evidence IDs include controller-run/session scope internally:
`t-6` alone is never a globally unique join key.

`GET /api/procedures/evidence/ev_receipt_12?view=compact`:

```json
{
  "schemaVersion": 1,
  "requestId": "req_04",
  "id": "ev_receipt_12",
  "kind": "container_receipt",
  "availability": "retained",
  "provenance": {"source": "native_task_receipt", "controllerRunId": "run_44", "connectionId": "conn_2", "taskId": "t-6", "world": "DizzysHellheim", "observedAt": "2026-10-30T18:49:59Z", "entity": "chest_A", "derivedFrom": []},
  "data": {"direction": "withdraw", "requested": {"Raspberry": 4}, "moved": {}, "contentsRevision": "cs_A_3", "contentsSummary": {"Raspberry": 0, "Wood": 15}},
  "coverage": {"complete": true, "omittedItemKinds": 0},
  "contentDigest": "sha256:2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
}
```

Digests in wire examples illustrate the full SHA-256 shape, not fixture bytes.
A receipt proves that operation's observation and moved counts, not all future
stock. A remembered/reconciled record retains original observation time,
`observedEntity`, current candidate entity and reconciliation evidence. It must
not be relabeled fresh because its location was matched. A missing/truncated
scan is not absence evidence. Refer to the exact snapshot/revision used in
ranking, not a mutable "latest chest" reference.

Evidence reads recheck authorization. Redact credentials/chat/private owner
data consistently before model exposure; preserve a redaction indicator and
digest of the retained permitted representation. Expired objects return 410
`evidence_expired` with metadata/retention dates where available, not a false
empty inventory. Source records and derived summaries are immutable. Corrections
append superseding feedback with links to the old record.

## Draft, edit and static validation

`POST /api/procedures/drafts` with `Idempotency-Key: draft-g17-1`:

```json
{"schemaVersion":1,"fromVersionId":"pv_food_01","purpose":"Continue search after a fresh negative chest observation","evidenceRefs":["ev_receipt_12","ev_context_12"]}
```

```json
{"schemaVersion":1,"requestId":"req_05","draftId":"draft_18","revision":"dr_1","baseVersionId":"pv_food_01","status":"draft"}
```

Read the draft with `GET /api/procedures/drafts/draft_18?fields=manifest,source`.
Edit with `PUT /api/procedures/drafts/draft_18`, `If-Match: "dr_1"` and a fresh
idempotency key. Initial APIs should conditionally replace complete bounded
source and manifest, avoiding ambiguous line patches. The manifest is supplied
in full; the shorter example changes only source and edit rationale:

```json
{"schemaVersion":1,"source":"def step(context, state, outcome):\n    if context[\"objectiveSatisfied\"]:\n        return {\"action\": \"done\", \"state\": state}\n    candidate = context.get(\"nextAction\")\n    if candidate == None:\n        return {\"action\": \"needs_plan\", \"reason\": \"allowed candidates exhausted\", \"state\": state}\n    return {\"action\": \"run\", \"template\": candidate[\"template\"], \"params\": candidate[\"params\"], \"state\": {\"phase\": \"search\"}}\n","rationale":"A fresh negative updates the host candidate ledger; the next allowed candidate is still useful."}
```

```json
{"schemaVersion":1,"requestId":"req_06","draftId":"draft_18","revision":"dr_2","changedFields":["source","rationale"],"validationStatus":"not_run"}
```

Field absence preserves a draft field; null is allowed only where the published
schema explicitly permits it. Unknown fields are rejected, following the
repository's strict JSON decoding. Every edit invalidates older validation and
replay eligibility for publication. Draft ACLs include provenance and base
version access; cloning a package does not launder its permissions.

`POST /api/procedures/validations`:

```json
{"schemaVersion":1,"draftId":"draft_18","draftRevision":"dr_2","catalogVersion":"cat_7","profile":"renki-decision-tree-v1"}
```

```json
{"schemaVersion":1,"requestId":"req_07","validationId":"val_21","draftRevision":"dr_2","valid":true,"checks":{"syntax":"pass","manifest":"pass","capabilities":"pass","stateCompatibility":"pass"},"warnings":[{"code":"live_evidence_required","message":"Replay cannot establish chest B transfer success."}],"diagnostics":[]}
```

Check AST rules and bounded parsing, exact entry point, schema sizes/depth,
known adapter/predicate versions, permissions declared, dependency cycles and
checkpoint compatibility. Apply bounded nesting before parsing hostile deep
sources; evaluator step limits begin too late to protect an unbounded parser.
Some directive/parameter checks depend on inputs and must run again at replay
and dispatch. Static validation never guarantees success or exhaustively proves
that every possible branch returns a valid directive.

An invalid source returns HTTP 422 `application/problem+json`:

```json
{"type":"urn:renki:problem:policy-syntax","title":"Unsupported policy syntax","status":422,"detail":"Dynamic indexing is not allowed in renki-decision-tree-v1.","instance":"/api/procedures/validations/val_22","code":"policy_dynamic_index","requestId":"req_08","diagnostics":[{"path":"source","line":4,"column":26,"rule":"literal-index","actual":"candidates[state[\"index\"]]","suggestion":"Use the host-ranked context[\"nextAction\"] or an advertised named alternative."}]}
```

Use stable codes for `capability_unimplemented`, `permission_denied`,
`checkpoint_schema_incompatible`, `pending_intent_uncertain`,
`evidence_expired`, `scenario_incomplete`, `counterfactual_divergence`,
`budget_exhausted` and `revision_conflict`; do not parse English to branch.
Runtime needs-plan outcomes are successful diagnostic reads, not HTTP failures.

## Scenario capture, replay and comparison

`POST /api/procedures/scenarios` captures retained data without operating the bot:

```json
{"schemaVersion":1,"goalId":"g17","decisionIds":["d11","d12"],"include":["policy_inputs","checkpoints","receipts","candidate_ledger"],"redactionProfile":"agent_safe_v1","maxBytes":262144,"labels":["wrong_food_hint"]}
```

```json
{"schemaVersion":1,"requestId":"req_09","scenarioId":"sc_hint_1","revision":"sr_1","kind":"recorded","complete":true,"decisionCount":2,"evidenceRefs":["ev_receipt_12","ev_context_12"],"limitations":["No recorded chest B withdrawal follows decision d12."],"retention":{"pinned":true,"expiresAt":null}}
```

Capture the source inputs, exact state/outcome, goal spec/authority boundary,
host-derived context, candidate ledger, observation revisions, seed if used,
package/catalog/interpreter versions and expectation origin. Replay must use a
virtual clock. Bounded capture pins its permitted evidence under a storage quota;
if pinning fails, fail the capture or mark it incomplete. No silent reference to
an audit file scheduled for pruning. Exports default to redacted minimal data;
imports are untrusted and receive schema/size checks. Manual expectations and
synthetic outcomes carry author/derivation labels, distinct from observed facts.

`POST /api/procedures/replays`:

```json
{"schemaVersion":1,"subject":{"draftId":"draft_18","revision":"dr_2"},"scenarioIds":["sc_hint_1"],"mode":"recorded_decisions","maxDecisions":64,"maxWallMs":1000}
```

```json
{"schemaVersion":1,"requestId":"req_10","replayId":"rp_31","status":"completed","checks":{"schema":"pass","authority":"pass","expected_negative_continues":"pass"},"firstDivergence":{"decisionId":"d12","baselineAction":"needs_plan","candidateAction":"run","candidateTarget":"chest_B"},"outcome":{"decisionAssertionsPassed":true,"liveSuccess":"unknown","reason":"No receipt exists for the proposed chest B action."},"coverage":{"recordedDecisions":2,"continuedWorldSteps":0},"traceRef":"ev_replay_31"}
```

Support three explicit modes:

1. `recorded_decisions`: feed identical recorded input snapshots to pure policy
   decisions. Verify each decision independently. Stop causal continuation at
   the first different world action. Later logged inputs may still be evaluated
   as independent probes, clearly marked unreachable under this edited trace.
2. `host_simulation`: run the deterministic host with virtual time and explicitly
   stubbed native outcomes. Test ledger advancement, budgets and proof rejection.
   Results are simulated; stubbed receipts cannot enter live goal evidence.
3. `live_canary`: a separate authorized operation, never a replay mode that can
   accidentally send actions. It exercises real native effects under limits.

The API enum for replay therefore accepts the first two modes only. Compiling a
different action does not make its observed result available. Code coverage,
expected branches and native physics are different kinds of evidence.

`POST /api/procedures/comparisons` runs both versions against identical pinned
scenario inputs/catalog/interpreter and declared assertions:

```json
{"schemaVersion":1,"baseline":{"versionId":"pv_food_01"},"candidate":{"draftId":"draft_18","revision":"dr_2"},"scenarioIds":["sc_hint_1","sc_empty_all_1","sc_restart_1"],"mode":"recorded_decisions"}
```

```json
{"schemaVersion":1,"requestId":"req_11","comparisonId":"cmp_8","sourceDiffRef":"ev_diff_8","summary":{"scenarios":3,"improvedDecisionAssertions":1,"regressedDecisionAssertions":0,"unchanged":2,"unknownWorldOutcomes":1},"changes":[{"scenarioId":"sc_hint_1","decisionId":"d12","before":"needs_plan","after":"run food_probe.v1 chest_B","assertion":"continue_after_negative","result":"improved"}],"eligibility":{"candidatePublication":true,"stablePromotion":false},"missingStableRequirements":["bounded_live_resupply_evidence"]}
```

Return source/manifest/authority-requirement diffs separately from behavior diffs.
Show first divergence and changed target/parameters/wait/budget use; textual
line count is insufficient. Block publication if any required evidence is
missing, stale for the package, or from a different host/catalog version without
an explicit compatibility result. A catalog change does not automatically
invalidate every fixture; record the compatible contracts actually used.

## Publish, canary and rollback

Proposed permissions are `procedure.read`, `procedure.author`,
`procedure.publish`, `procedure.operate` and `procedure.admin`. These are future
service permissions, not current role names. Native permissions remain final.
Default model principals receive read/author only; an owner may explicitly grant
bounded publication/operation authority. Read-only location access never grants
inventory inspection or travel. Capability discovery reports the intersection.

`POST /api/procedures/publications`, with authorizing principal and a channel
`If-Match`, publishes a candidate without making it the default:

```json
{"schemaVersion":1,"draftId":"draft_18","draftRevision":"dr_2","validationId":"val_21","comparisonId":"cmp_8","channel":"food-search/candidate","expectedChannelRevision":"ch_2"}
```

```json
{"schemaVersion":1,"requestId":"req_12","versionId":"pv_food_02","channel":"food-search/candidate","channelRevision":"ch_3","defaultForNewGoals":false,"liveUse":"canary_only"}
```

Candidate publication requires validation/replay gates; stable-channel promotion
additionally requires declared live canary evidence. Publish the exact immutable
candidate first, canary that version, then promote the same version without
changing its bytes. Report both eligibility classes explicitly. Published
version bytes cannot be edited in place.

`POST /api/procedures/canaries` is a separate live request:

```json
{"schemaVersion":1,"versionId":"pv_food_02","authorizationRef":"auth_canary_4","goalSpec":{"kind":"food_supply","world":"DizzysHellheim","areaRef":"home_area_2","requiredCount":4},"limits":{"maxGoals":1,"maxNativeTasks":4,"maxElapsedS":120},"stopOn":["damage","permission_change","uncertain_intent","evidence_violation"]}
```

```json
{"schemaVersion":1,"requestId":"req_13","canaryId":"can_4","goalId":"g18","status":"queued","pinnedVersionId":"pv_food_02","automaticPromotion":false}
```

After a passing canary, `POST /api/procedures/publications` can conditionally
promote an existing candidate instead of publishing a draft:

```json
{"schemaVersion":1,"versionId":"pv_food_02","canaryIds":["can_4"],"channel":"food-search/stable","expectedChannelRevision":"ch_6"}
```

```json
{"schemaVersion":1,"requestId":"req_13b","versionId":"pv_food_02","channelRevision":"ch_7","defaultForNewGoals":true,"activeGoalsChanged":0}
```

The request is a strict union: draft publication requires draft/validation/
comparison fields; promotion requires an existing version and canary evidence.
Both require publication permission and the channel's `If-Match` precondition.

The authorization reference is a server-issued scoped grant tied to the
authenticated principal, world, area, task types, limits and expiry. It cannot
be invented by source or provided as an arbitrary bearer string. Verify current
native/session gates at start and every dispatch. Do not change an existing
goal's scope or replenish its budget by calling it a canary. Existing resumed
goals retain their original limits; a genuinely new trial is a new authorized
goal with separately reported exposure, not evidence of recovery of the old one.

Canaries need an available safe bot and cleanup/supervision through the existing
body owner. No simultaneous competing baseline task can control the same body.
Sequential baseline/candidate trials record different conditions and cannot
establish causal improvement alone. Failure to find food is a legitimate
bounded shortfall; bypassed guards or false success are release blockers.

`POST /api/procedures/rollbacks` with `If-Match: "ch_7"`:

```json
{"schemaVersion":1,"channel":"food-search/stable","toVersionId":"pv_food_01","expectedChannelRevision":"ch_7","reason":"Canary exceeded unchanged-target retry bound","activeGoals":"leave_pinned"}
```

```json
{"schemaVersion":1,"requestId":"req_14","channelRevision":"ch_8","defaultVersionId":"pv_food_01","activeGoalsChanged":0,"worldEffectsReversed":false}
```

Rollback changes selection for future goals; it does not undo a transfer or
rewind world state. An operator may separately stop affected active goals.
Live replacement requires a quiescent, receipt-complete checkpoint, compatible
state and explicit goal revision authorization; otherwise start a separately
authorized goal after resolving uncertainty. Never hot-swap an in-flight step,
revive a cancelled goal or silently migrate to a different predicate.

## Durable feedback and bounded improvement

`POST /api/procedures/feedback` accepts an append-only annotation:

```json
{"schemaVersion":1,"goalId":"g17","versionId":"pv_food_01","decisionId":"d12","kind":"diagnostic_hypothesis","claim":"Policy terminated on a candidate-negative outcome","evidenceRefs":["ev_receipt_12","ev_context_12"],"proposedRemedy":"Continue to the next host-authorized candidate","supersedesId":null}
```

```json
{"schemaVersion":1,"requestId":"req_15","feedbackId":"fb_6","origin":"agent","confidence":"unverified_hypothesis","affectsAuthority":false,"affectsSuccessProof":false}
```

The service derives actor/origin from authentication, rather than trusting an
agent-supplied `host` or `owner` label. Appending feedback requires author
permission and access to referenced evidence; only the host can issue verified
outcome records. Preference changes affecting authority use their existing
authorized settings/order path, not a feedback annotation.

Persist feedback linked to immutable package/scenario/decision/evidence IDs.
Recommended fields: feedback ID, author/principal, origin (`host`, `owner`,
`model`, `replay`, `live_trial`), observed result, failure category, scope,
evidence refs, confidence/uncertainty, proposed cause/remedy, supersedes ID and
timestamps. An owner preference such as "leave my Honey" is a policy/authority
constraint, not a stock observation. A model-generated cause cannot supersede
a native receipt. Host evidence determines success; users can annotate quality.

Measure per version and objective/adapter/environment cohort:

- Host-verified successes / all authorized started goals; include all terminal
  categories, cancellation and unresolved goals separately with censoring.
- Expected negatives resolved, distinct candidates inspected, repeated unchanged
  action rate, attempts/tasks/travel/time per verified unit, partial progress.
- Needs-plan categories: actual resource shortfall, missing capability, stale
  sensing, authority refusal, checkpoint failure and unknown action outcome.
- False-success attempts, authority/evidence violations, damage, owner takeover
  latency and budget overruns. Release gates use zero tolerated violations in
  the defined test corpus; passing that corpus is not proof of universal safety.
- Authoring cost: diagnosis bytes/tool calls, validation failures, replay time,
  model requests/tokens and accepted versus rolled-back proposals.

Keep numerators, denominators, exposure, uncertainty counts and observation
windows, rather than only a success score. Separate simulator results from live
metrics, and expected search negatives from execution faults. Never reward
declaring needs-plan early or selecting only easy cases to improve a percentage.
Retain raw evidence long enough for the release decision under an explicit
quota; aggregate metrics are not substitutes for its proof bundle.

The future autonomous loop runs only under a configured scoped grant:

1. Host exhausts valid deterministic alternatives or classifies a repeated
   failure. It emits one fingerprinted needs-plan event, not one request per tick.
2. Retrieve compact diagnosis/capabilities; expand only referenced evidence
   needed to distinguish hypotheses. Treat world text, chat, item names and
   traces as untrusted data, never instructions to widen permissions.
3. Propose one draft edit with an evidence-linked rationale. Reuse the same
   validation/replay tools an interactive coding agent uses.
4. Reject invalid/regressing changes; at most a configured number of edits and
   model calls. Initial experiment: two calls/two drafts, 30 seconds total and
   a configured token ceiling, persisted before submission. This is a proposed
   tuning value, not a current setting or extra authority after goal expiry.
5. Publish/run only if the principal and release policy permit it. Preserve
   previous versions; promote after declared canary gates, otherwise stop with
   the specific remaining shortfall. A third automatic replan is not implicit.

Persist an improvement episode ID, parent goal, input fingerprint, requests,
reserved/consumed budgets, jobs, versions and final disposition. Count usage
across new drafts and restarts. When a remote request's completion/usage is
unknown, retain its conservative reservation and reconcile by request ID; do
not reset the allowance. Native safety still runs if planner service is down.

## Wrong food hint: realistic agent workflow

1. Diagnose `g17`: chest A was remembered to contain Raspberry; its exact new
   receipt shows none, while chest B remains authorized. Inspect `d12`, not the
   whole audit day. Verify the host offered B. If it did not, this is a host
   candidate-generation defect that a policy edit cannot fix; record a code
   issue and keep evidence instead of fabricating `nextAction` in the script.
2. Capture `sc_hint_1` before retention expires. Clone `pv_food_01`, replace the
   branch that treats all `item_missing` results as terminal, and validate the
   complete draft against the advertised restricted grammar/catalog.
3. Replay the recorded decision: the edited policy now selects B. Report that
   branch improvement and unknown transfer outcome. Add a separately labeled
   host-simulation scenario: A is empty, B has four Honey, a matched withdrawal
   moves four and a newer inventory confirms four carried. Also test B empty,
   partial move, permission revoked, cancelled goal and interrupted receipt.
4. Compare against the baseline corpus. The invariant is that A's negative
   advances the ledger once, B is considered, unchanged negatives do not loop,
   and completion needs host evidence. Do not make a fixture expect Honey in
   the real game merely because the synthetic scenario contains it.
5. Present the source/behavior diff and missing live evidence. A principal with
   publication/operation authority can create one bounded canary goal. Inspect
   actual receipts and inventory; concurrent eating requires matched consumption
   evidence and an objective definition that explicitly counts it, never an
   unexplained missing inventory delta. The simple four-carried-items objective
   above succeeds only if the fresh carried inventory meets its requirement.
6. Publish stable only after live gates. Roll back channel selection on
   regression, retain the feedback/scenario bundle, and leave old interrupted
   goals pinned until their uncertainty is actually resolved.

## Versioning and implementation recommendations

Version transport envelopes, package format, grammar profile, host capability,
objective predicate, checkpoint state and scenario format independently. Additive
optional response fields can be compatible; changed outcome/proof semantics
require a new contract version. Reject unknown required contracts and fields on
writes. Old packages keep old interpretable state or fail with an actionable
compatibility error; migration is an explicit bounded host function, not script
code run during file loading. Never alter existing proof meaning retroactively.

Keep the current `goals.json` v1 reader unchanged until a migration exists.
Start with separate bounded versioned files/directories for procedure packages,
feedback and pinned scenarios. Atomic writes and copy-on-write updates should
follow `internal/goal/store.go`. Consider a database only when measured query,
concurrency or quota needs justify it; existing atlas SQLite use does not imply
a shared transaction with the game. Design recovery for partial publication of
artifacts and records (write immutable artifacts first, then atomic references;
garbage-collect unreferenced content under a grace period).

Suggested implementation seams, not files added by this research:

| Existing seam | Incremental implementation |
| --- | --- |
| `internal/goal/policy.go` | Typed diagnostics with rule/location; keep AST restrictions; make evaluator fingerprint explicit. |
| `internal/goal/runtime.go` / `store.go` | Structured outcomes, durable candidate ledger, decision IDs and evidence links; explicit per-adapter reconstructible phases. Start with one food probe adapter, not arbitrary `behaviour.Runner` serialization. |
| `internal/behaviour/definition.go` / `engine.go` | Preserve body ownership, cancellation, `FailureAware` handling and per-native-task accounting across an admitted adapter. |
| `internal/audit` / `internal/session` | Run-scoped receipt IDs, bounded evidence resolver, trace sequence/index and explicit retention gaps. Keep existing audit consumers working. |
| `internal/app/goals.go` / panel handler wiring | Shared authenticated service for diagnosis and offline authoring; enforce preconditions and service permissions independently of native gates. |
| Proposed `internal/procedure` | Immutable package/draft catalog, validation/replay/comparison jobs, feedback and publication records. This hosts tooling, not another autonomy loop. |
| `scripts/vm/resupply_trial.py` | Source for recorded/synthetic fixtures and a bounded live acceptance supervisor; do not leave production candidate decisions in a VM Python supervisor. |

Deliver read-only diagnosis/capture first. Then implement deterministic food
phases and useful replay assertions alongside M2/M3 work. Agent draft/edit tools
follow, then controlled publication/canary/rollback. Automatic model revision,
visual authoring and HTN decomposition follow only demonstrated need. This
research does not enable any of them or replace existing milestone acceptance.

Add implementation recipes in the appropriate just module: for example,
`just check controller procedure-replay <scenario-id>` and a focused plugin
policy benchmark when it exists. These names are proposed recipes, not runnable
commands today. Existing focused validation is
`just check controller test-packages ./internal/goal ./internal/app ./internal/behaviour`;
the repository gate remains `just check ci`, with committed review through
`just check review` before pushing.

Promotion to a Go composition or C# routine requires a stable parameter and
evidence contract, representative successful/adverse/uncertain cases, no known
authority regressions, measured reuse or performance benefit, and equivalent
decisions or documented deliberate differences over the corpus. Keep a pinned
script/reference implementation for comparison and rollback. Promoting physical
control additionally requires native core tests, tick/allocation measurements
and bounded live evidence; a policy-only replay cannot certify it.
