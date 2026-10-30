---
schema: 4
id: TKT-01M62N12M69CCYJ2STYPT31KJ5
title: "Controller and plugin: eat to recover health"
type: task
status: in-progress
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - controller
assignees: []
milestone: M2
parent: TKT-01M62MZN7QMTJ3ST0KRZH15DXZ
origin: null
dependencies:
  - TKT-01M62N129K1RGB8VSJZ3SDAC7N
blocks_on: none
references: []
moved_to: null
claim:
  actor: agent:codex/renki-resume
  branch: t3/food-recovery
  worktree: /home/sothr/.t3/worktrees/renki/t3-b9d91179
  commit: 3e3402d2d486e9a904ceeae2c278929145db785d
  session: null
  claimed_at: 2026-10-29T23:21:07Z
  expires_at: null
archive: null
created_at: 2026-10-29T01:40:10Z
updated_at: 2026-10-30T00:08:26Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:codex/renki-resume
  name: ""
extensions: {}
---

## Description

The game heals only from food, and the bot never eats: after the fights on 2026-10-28 it sat at 11/25 health for good. use_item (7.2) lets it eat; the controller keeps food slots filled from what it carries, or asks the owner for food when it has none.

## Acceptance criteria

- [x] use_item eats a food by prefab
- [x] The bot keeps its food slots filled from its inventory
- [ ] It asks an owner for food when it has none and its health is low

## Implementation plan

Report optional food status in state: active meals and remaining time, plus unique usable carried foods with health/regen and game CanEat eligibility. Add a controller food keeper that fills empty slots before refreshing eligible meals through instant use_item, respects capabilities/current permissions, waits for post-action observations, backs off failures, avoids stale/disconnected/menu/dead/sleep/attack/flee states and never replaces a body task. Ask configured owners once per bounded cooldown only when injured with no active or usable carried food. Keep consumption entirely in the normal game path; add protocol fixtures and controller policy/lifecycle tests, then observe live automatic recovery alongside a from-home harvesting loop.

## Notes

**agent:codex/renki-resume** at 2026-10-29T23:22:33Z

Food-only use_item is implemented with matching Go/C# argument validation, supported tasks, welcome/Enabled permission, inventory-GUI UseItem semantics and actual one-item/active-food verification. It preserves game refusals and reports missing/non-food items without consumption. Offline argument/runner regression coverage is added; just check ci passed before a small related correction: actual damage now latches harvesting cancellation so food regeneration above initial health cannot conceal a hit. Rechecking the final source. Live food and recovery checks remain pending. Food upkeep and owner requests are not implemented; all acceptance criteria remain open until live evidence.

**agent:codex/renki-resume** at 2026-10-29T23:33:13Z

Landed primitive in PR #32, main 11c0e8d; final just check ci (193 plugin tests) and committed just check review passed. just check vm install verified package and installed plugin hashes while offline. On rejoin, t-1 returned safely home, 2.7 HP preserved. Food chest 1:30600 at [-37.09,36.09,3.42] held CookedDeerMeat11, NeckTailGrilled9, CookedMeat4 plus fish. Withdrew two of each cooked food, then t-8/t-9/t-10 each consumed exactly one and confirmed active food. Immediate duplicate t-11 refused without consumption; seeds t-12 and absent Raspberry t-13 refused invalid_target. Pack retains one spare of each, AxeFlint1, BeechSeeds15 and Wood2. Chest totals remain 9/7/2 cooked foods and original fish. Tas regenerated naturally from 2.7 to full (113.4/113.4 as food maximum slowly decayed), without fabricated healing or damage. Evidence .checks/audit-window-MUTQbj.jsonl and .checks/audit-window-8SwOKK.jsonl. AC1 is earned. Automatic upkeep/owner requests and live fourth-food refusal remain open.

**agent:codex/renki-resume** at 2026-10-29T23:58:14Z

Implemented optional game-reported active/carried food status, exposed it through panel status, and wired automatic upkeep through normal use_item. Empty slots take precedence over refresh; current game eligibility governs consumption. One action waits for new state; failure cooldown, stale state/connection/world guards, permissions and sleep/attack/flee guards are covered. Owner say requests are bounded to five minutes and require low health with no active or usable carried food. just check ci passed (20261029T235737Z-Wj8non.log). Live automatic consumption and naturally occurring foodless low-health request remain unverified; AC2/3 stay open until evidence.

**agent:codex/renki-resume** at 2026-10-30T00:03:46Z

PR #36 merged as dc981d1; just check vm deploy passed with installed hashes. On rejoin at home, automatic t-1/t-2/t-3 consumed exactly the carried CookedDeerMeat, CookedMeat and NeckTailGrilled, with active food confirmation and zero of each left carried. Health regenerated normally from 25 to full 113.7/113.7. T-4 withdrew one spare of each from food chest 1:30600; current food observation correctly marks all three ineligible, with no duplicate consumption. Pack retains AxeFlint1, seeds15 and original Wood2. Evidence .checks/audit-window-G5p4Sh.jsonl and .checks/task-f0u_bp2u.json. AC2 earned. Owner-request policy passes deterministic tests but no natural injured/foodless live case was staged; AC3 remains open.

**agent:codex/renki-resume** at 2026-10-30T00:08:26Z

Restart check exposed active meal prefab names as localization tokens because deserialized Food.m_item lacks m_dropPrefab. Report the game Food.m_name, its persisted prefab identifier, so empty-slot-before-refresh priority remains consistent across restart. The game eligibility already prevented duplicate consumption. Normal food maintenance and recovery stay verified; owner request live case remains open.
