---
schema: 4
id: TKT-01M6CP0J0A6TVZCHAPVMVB0SGQ
title: Execute natural get-in-bed requests as sleep commands
type: bug
status: done
status_reason: null
priority: high
due_on: null
labels:
  - controller
  - chat
assignees: []
milestone: M2
parent: TKT-01M62PRDSBCRCPH5S0TGGCMAEY
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-01T23:09:46Z
updated_at: 2026-11-01T23:14:49Z
created_by:
  id: agent:codex/renki-milestones
  name: ""
updated_by:
  id: agent:codex/renki-milestones
  name: ""
extensions: {}
---

## Description

Nimah reported that Tas could not fulfil a go-to-bed request and asked to add it to the current combat/escort/food/harvesting test batch. Audit on 2026-11-01 at 23:06:44Z captured authorized shouted text "tas get in bed please" from Nimah. At 23:06:47Z Tas sent say task t-7 promising he was heading to bed, but no bed interaction was dispatched. The normalizer already removes please; the grammar lacks "get in bed", so the request fell through to conversational generation. A prior Castile Rackham request was from a non-operator and must not gain command permission through this fix.

Add natural bed-entry aliases and a dispatch-level regression using the existing ownership/bed selection path. Strengthen the existing alias-loop test, which currently accepts an interaction from an earlier iteration and can conceal a missing alias. Keep the parent ticket's nighttime bed entry, claims/spawn, approach routing and honest native refusal checks in the current live test batch. Do not take another player's bed or report claiming as actual sleep.

Evidence: .checks/protection-policy/bed-audit-path.txt references retained audit-window-zqaAsg.jsonl.

## Acceptance criteria

- [x] An authorized get-in-bed request dispatches interact with sleep=true to a suitable bed
- [x] Guests cannot issue these requests and unknown or occupied beds remain excluded
- [x] Each tested sleep phrase must dispatch a new task; earlier successful phrases cannot mask a parsing failure

## Implementation plan

Route get-in-bed/get-into-bed and possessive variants through the existing explicit sleep command. Keep authorization, bed ownership, native reach and sleep/claim verification unchanged. Strengthen the existing sleep alias test to require one fresh task per phrase, including please normalization and guest rejection. Run focused chat checks, the full just check ci gate, commit/review, deploy only the controller so the game stays online, and record actual nighttime bed entry as a pending parent-ticket batch check.

## Notes

**agent:codex/renki-milestones** at 2026-11-01T23:13:57Z

Added get in bed/get into bed/go to your bed and possessive variants to the existing sleep command. Existing please/punctuation normalization now recognizes the exact reported phrase. Strengthened the dispatch test to require a new body task for every alias, rather than allowing an earlier interaction to satisfy later phrases, and exercised the new wording against guest authorization. Unknown/occupied bed exclusions and sleep=true remain covered.

just check controller test-packages ./internal/chat passed (.checks/20261101T231013Z-ow6SJv.log). just check ci passed in173s (.checks/20261101T231022Z-bKfNXs.log), including controller race checks, native plugin/policy/interop checks, UI scenarios and live-tool simulations. Current audit shows an explicit controller logout command l-8 at23:07:01Z after the bed request; Tas is currently connected at menu. Preserve that state during controller-only deployment. Nighttime bed approach/actual in_bed and escort interruption remain unchecked on the parent ticket and in docs/TESTING.md.

**agent:codex/renki-milestones** at 2026-11-01T23:14:49Z

Implementation committed at880cbc3. just check review HEAD~1..HEAD passed with no findings (.checks/20261101T231357Z-lVTSRV.log). just check vm deploy-controller passed (.checks/20261101T231408Z-Fucupf.log). Fresh API status reports controller version880cbc3, connected=true, phase=menu and no body task: deployment preserved the explicit logged-out state without restarting or joining the game. Evidence .checks/protection-policy/bed-after-deploy.json. The command-parser bug is fixed; current-layout nighttime bed entry/escort interruption remains explicitly pending on parent TKT-01M62PRDSBCRCPH5S0TGGCMAEY.

## Summary

Fixed and deployed at 880cbc3; full CI/review passed. Nighttime bed-entry check remains on the parent batch ticket.
