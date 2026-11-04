---
schema: 4
id: TKT-01M6KZZ5F5C3YYJWX7RV2C3TMS
title: Choose food by activity and reserve better meals for work
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - controller
  - safety
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-04T19:18:27Z
updated_at: 2026-11-04T20:09:35Z
created_by:
  id: agent:codex/explore-night
  name: ""
updated_by:
  id: agent:codex/explore-night
  name: ""
extensions: {}
---

## Description

The owner requests activity-aware meal priorities: eat tier1/2 meals while idle and reserve higher meals for escort or heavy work. Add deterministic shared selection, configurable tier overrides for modded foods, conservative emergency feeding and durable choice diagnostics. Preserve native CanEat, body ownership, food-goal reservations and receipt/backoff rules. Cover carried maintenance and autonomous personal/household food eating; explicit escort preparation remains work.

## Acceptance criteria

- [x] Idle maintenance conserves later-tier meals while escort, travel and labor can use them; native eligibility and food safety remain authoritative.
- [x] Food tiers and idle preference are configurable, with documented unknown/modded food and emergency fallback behavior.
- [x] Shared pure decisions explain activity, candidates and selection; meaningful tests cover maintenance and food goals, and full CI passes.
- [x] Deploy and verify settings and actual carried food decisions without claiming unavailable food trials.

## Implementation plan

Use one pure explained meal selector with known vanilla biome tiers and validated per-prefab overrides. Idle chooses eligible Meadows/Black Forest food and delays optional refresh; work/preparation may choose reserved meals and emergency hunger/injury can override. Share it between carried maintenance and goal eating. Use a read-only workload getter over scheduler ownership plus current controller/native tasks, real arrival freshness, scoped instant dispatch and post-action receipts. Persist goal decisions and log maintenance decision changes with scope/observation/task correlation; expose current choice in panel status and inventory. Test selectors, ownership gaps, stale/reconnect scenes, settings persistence and goal integration; run just check ci, commit/review, deploy and verify live without consuming unavailable food or inventing a tier for mods.

## Notes

**agent:codex/explore-night** at 2026-11-04T19:33:39Z

Owner accepted Meadows/Black Forest tier1/2 idle defaults and reservation of later/unclassified modded meals, with emergency override. Implemented shared pure selector with version/mode/activity/candidate reasons, configurable overrides, observed stamina for work ranking, lower-tier idle choice and late refresh. Keeper uses actual StateObservation timestamps and scoped instant DispatchFor while preserving post-action receipt/backoff. Food goals share selection and persist/log their decision; escort preparation, outdoor food trips and known events are explicit work contexts. Read-only scheduler ownership getter retains active runner/lent-follow workload without calling mutating List/Inspect; light idle chores retain their context. Panel Inventory shows mode/choice and runtime settings expose policy. Focused selector/goal/settings/bridge checks passed after fixing new fixture setup; native protocol round-trip exposed a missing optional-zero serialization rule, corrected and FixtureTests passed. Final fullCI is running. Static review recorded in docs/BEHAVIOUR-REVIEW.md; broader observer-purity/audit work remains draft.

**agent:codex/explore-night** at 2026-11-04T20:09:34Z

Food policy deployed08590de and retained on f254e95. Live settings food.activityAware=true,food.idleMaxTier=2 persist. Actual idle decision classified carried BoarJerky as tier1 and explained refresh_not_needed while native eligibility was true but the active meal still had368s left. No later-tier alternative was available for a real premium-versus-cheap consumption trial; do not claim one. Native eligibility and injury history remain intact. .checks/panel-batch-installed-{status,settings,inventory}.json retain actual observations. Final home status shows one active BoarJerky,two carried,full current health38.5/38.5,injurySeq0; max-health decline is meal decay. FullCI203s and committed review clean; Tas/private gameplay online and lease released.

## Summary

Implemented on PR179, deployed plugin/controller f254e95; full just check ci and committed review passed. Food policy deployed08590de and retained on f254e95. Live settings food.activityAware=true,food.idleMaxTier=2 persist. Actual idle decision classified carried BoarJerky as tier1 and explained refresh_not_needed while native eligibility was true but the active meal still had368s left. No later-tier alternative was available for a real premium-versus-cheap consumption trial; do not claim one. Native eligibility and injury history remain intact. .checks/panel-batch-installed-{status,settings,inventory}.json retain actual observations. Final home status shows one active BoarJerky,two carried,full current health38.5/38.5,injurySeq0; max-health decline is meal decay. FullCI203s and committed review clean; Tas/private gameplay online and lease released.
