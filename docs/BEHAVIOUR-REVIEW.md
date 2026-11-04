# Behavior architecture review

2026-11-04 static subagent review, requested by the owner during exploration,
night-invitation and food-preference work. This is an assessment, not proof of
runtime behavior or a mandate to rewrite the system.

Keep the existing Template/Runner, durable goal adapters, session automation
leases, owner control revisions and native bounded safety checks. A behavior-tree
rewrite would not solve the most pressing issues.

### Decisions need the same status as actions

The audit currently retains frames and lifecycle logs better than the reasons
for choices. `internal/inspection/inspection.go` acknowledges that decision inputs
and causal transitions are not archived. `Engine` filters candidates without a
shared decision result. Food and presence also run outside its body-work lane.

Introduce a versioned decision/transition envelope with policy version,
controller run, connection/world UID, observation sequences/times, relevant
inputs and guards, selected/rejected candidates, reason, phase change and task
ID. Record changes/actions rather than every identical tick. The inspector
should project those same records instead of maintaining a separate explanation.

This batch starts that direction with pure food Decision results, candidate
reasons, durable goal decisions, maintenance/goal audit logs and task correlation.
It also logs bedtime reservation attempts with day/night and native scope.
These are local improvements; a shared cross-routine audit schema remains work.

### Inspection must not advance history

`Engine.List()` calls `view()`, which updates pile/activity trackers, then
`Loot.Inspect()` calls `Loot.Observe()`, which can update settle history. Reads
do not dispatch work, but they can affect observation history. Move tracker
updates into scheduler observation ingestion and publish immutable cached
snapshots. Assert that repeated status reads cannot change later decisions.
This batch does not fix the existing List/Inspect behavior.

### Share workload and observation context

A gather/escort episode contains movement, storage, preparation, waits and gaps
without a native task. A passive run can borrow an owner follow. Nearby inferred
player activity is not proof of Tas's authorized work. Use current lease-backed
runner/goal context, then controller pending work and native tasks; do not infer
workload by calling the mutating List/Inspect paths.

This batch adds a side-effect-free ownership getter for food, retains working
context across runner gaps/borrowed follows, shares meal selection with food
goals, and uses real StateObservation arrival times plus DispatchFor for keeper
instant tasks. A common immutable observation/workload envelope across all
routines remains a useful follow-up.

### Make authoring and verification easier

The small Runner.Next contract works, but optional lifecycle/readiness/episode/
handoff interfaces are scattered. Document common durable-step and concurrent-
instant patterns. Prefer typed phases and explicit transitions. Extract adapters
only after two routines demonstrate the same lifecycle needs; avoid another DSL.

Build a deterministic scenario harness with recorded observations, clock
advances, native receipts, owner takeover, reconnect, restart and failed saves.
Assert semantic decisions and invariants, including observer purity and no replay,
rather than exact log wording. The existing pure checks, injected clocks, bridge
fixtures and persistence-failure tests are a strong foundation.

### Suggested sequence

1. Finish the local food/freshness/scoped-dispatch improvements in this batch.
2. Publish cached observer snapshots and a shared immutable workload context.
3. Standardize durable decision/transition records and inspector projections.
4. Add the scenario harness, then extract demonstrated common runner adapters.
