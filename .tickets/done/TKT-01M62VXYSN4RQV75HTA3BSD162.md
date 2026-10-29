---
schema: 4
id: TKT-01M62VXYSN4RQV75HTA3BSD162
title: "Controller: behaviour engine: templates, instances, triggers, repeat"
type: task
status: done
status_reason: null
priority: high
due_on: null
labels:
  - controller
assignees: []
milestone: M2
parent: TKT-01M62VXYQB6ZDBQE4E6ERGN83C
origin: null
dependencies:
  - TKT-01M62VXYR69PKZGJFEWQ1759GV
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T03:40:48Z
updated_at: 2026-10-29T20:09:34Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:codex/renki-resume
  name: ""
extensions: {}
---

## Description

Runs behaviour instances. Each instance names a template and fills in:

- **params**, checked against the template's schema;
- **trigger**: manual, an event (inventory full, a drop appeared, stock
  low, idle at home, nightfall) or a timer;
- **when**: conditions that must hold to start or carry on: what the bot
  carries, free slots, what the chests hold (only if seen recently), time
  of day, an owner nearby, where the bot is;
- **repeat**: once, on every trigger with a cooldown, every N minutes, or
  while or until a condition holds, with a maximum number of runs;
- **priority**: one number;
- **enabled**.

A run is one template doing its work: no loops or variables in the
instance. Anything more complex is a new template in Go, which keeps every
instance checkable before it is saved.

### Running

- One body task at a time. A higher-priority instance that becomes ready
  takes over, and the one it interrupted resumes or starts again.
- Survival (flee, eat, the after-death rules) and direct orders from an
  owner always come first. While following, instances stay within a leash
  of the leader.
- Start, stop, enable, disable, clone and edit work on any instance.
  Instances live in the settings store.
- Every trigger, start, step, finish and failure is written to the audit
  log, with the instance's name.

### First templates

`loot` and `stow` (their own tickets). The existing home rules can move to
templates later. They do not have to move now.

## Acceptance criteria

- [x] Instances are checked against their template's schema before they are saved
- [x] Triggers, conditions, repeat and priority decide which instance runs, one at a time
- [x] A higher-priority instance takes over, and an owner's order always does
- [x] Start, stop, enable, disable, clone and edit work on any instance and survive a restart
- [x] Each run's start, steps and end are in the audit log with the instance name

## Implementation plan

Read the existing settings instance store, session dispatcher, chat/home/retreat dispatch paths and protocol snapshots. Add a session automation lease serialized with ordinary dispatch: one engine run may send successive steps, while any normal body order, stop, logout, reconnect, death or world change invalidates its authority before the next step. Hold ownership over the run, not just a streamed state frame. Do not cancel a newer owner/survival task when stopping stale automation.

Build a controller behaviour package with a registered Go-template interface, strict definition/parameter validation before persistence, deterministic priority selection, event/timer/manual triggers, snapshot conditions (inventory slots/items, weight, encumbered/stranded, nearby owner/place, time and recently observed chest contents), cooldown/repeat/run limits, bounded pending triggers and named audit lifecycle records. Higher-priority ready instances restart interrupted lower-priority runs. Expose engine methods and authenticated panel API for list/save/start/stop/enable/disable/clone/delete; detailed chat/UI remain their own tickets. Include one bounded move_to template to exercise the engine without changing inventory or enabling combat. Leave loot/stow to their existing tickets.

Use fake time for scheduler/condition tests and the real bridge for ownership/preemption races. Verify direct orders and flee win before a new state frame, stop cannot cancel a newer task, invalid definitions never save, stale chest observations fail conditions, and settings survive restart without replaying an unfinished manual run. Run just check controller test-packages for affected packages, just check fmt, just check ci and committed just check review. Land, deploy the controller only, and verify version/health; live game route/combat checks stay deferred.

## Notes

**agent:claude-code/renki-groom** at 2026-10-29T04:12:05Z

2026-10-28: conditions need the carried weight against weightMax (and the encumbered and stranded flags), not only free slots.

**agent:codex/renki-resume** at 2026-10-29T20:08:47Z

Implemented the M2 controller engine on t3/behaviour-engine. Session automation ownership now spans task steps and serializes wire order with normal dispatch. Normal body orders (including stop/flee), logout, reconnect, death and world changes invalidate old authority; queued work also tracks normal-order revisions. Acquisition pins the observed connection/world so a stale planning snapshot cannot start in a new world. A stale run never stops a newer owner task.

The behaviour package validates template definitions/parameters before persistence and on load; schedules manual/event/timer triggers with conditions, repeat/cooldown/run limits and deterministic priorities; stops on changed conditions; restarts lower-priority interrupted instances; bounds incoming event processing; and emits named trigger/start/step/finish/failure/interruption logs. The initial bounded move_to Go template exercises the engine. Authenticated /api/behaviours controls support save/list/start/stop/enable/disable/clone/delete, using the panel CSRF header. Saved definitions/enabled flags survive restart, while unfinished manual work and queued triggers do not replay. docs/BEHAVIOURS.md documents the API and policies.

Verification: just check controller test-packages ./internal/session ./internal/behaviour ./internal/app ./internal/panel and just check ci passed. The full gate includes controller race tests, Linux/Windows builds, plugin build and all 158 C# tests including interop. Real-bridge regressions exercise concurrent owner-stop wire order, immediate flee preemption before a new state, stale-handle stop refusal, queued-work cancellation, stale-world acquisition, priority/restart, conditions, event coalescing, cooldown/run limits, failed steps/deadlines, persistence, authentication/CSRF and named lifecycle logs. A fixture initially omitted flee from its advertised capabilities; the fixture was corrected to test the actual allowed task, without changing production permissions.

Scope: the engine initially waits for an idle body and never interrupts a running follow; controlled loot detours with a leader leash belong to the hauling templates. Loot/stow, the stock_low emitter and dedicated chat/page controls remain their separate tickets. No definitions are installed by default. Live navigation comparisons and M3's controlled Greyling retreat remain deferred at the owner's request. Committed review, landing and a controller-only deployment with disabled-definition API verification follow this implementation commit.

## Summary

Implemented on t3/behaviour-engine: validated named Go templates, manual/event/timer scheduling, inventory/weight/flags/time/owner/place/recent-stock conditions, cooldown and repeat/run limits, deterministic priorities, higher-priority restart, bounded events, and authenticated persisted definition controls. Session ownership guards stop stale continuations and stale stops from overriding owner/survival orders across state lag, reconnect, death, logout and world changes. Named lifecycle audit records cover runs and steps.

The first move_to template is bounded; loot/stow, stock-low generation and page/chat controls remain separate tickets. The engine waits for an idle body; hauling templates still need controlled follow detours and a leader leash. Saved policies/enabled flags persist; interrupted manual work does not replay on restart. No default job is enabled.

just check ci passed (including controller race tests, both controller builds, plugin build and 158 C# tests), focused controller checks passed and just check review returned no findings. Behaviour controls, lifecycle and ownership were verified with real bridge connections and authenticated HTTP tests. docs/BEHAVIOURS.md describes the runtime/API contracts. Controller-only deployment and disabled-definition API verification follow landing; paired navigation and Greyling checks remain deferred.
