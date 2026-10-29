---
schema: 4
id: TKT-01M61T36B839WT7KEZ14TDMB8B
title: "M1: bot joins, follows, stops, reports status and chats"
type: epic
status: ready
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - controller
assignees: []
milestone: M1
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-28T17:49:28Z
updated_at: 2026-10-29T00:18:48Z
created_by:
  id: agent:claude-code/valheim-setup
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

The first milestone: a second Valheim client on the Windows VM, driven by
the C# plugin and the Go controller, that joins the server unattended, follows
an owner, stops, reports its status and holds a conversation in chat.

Perception, inventory, gathering and combat are M2 and later. The protocol is
docs/BRIDGE-PROTOCOL.md; the tickets below cite its sections.

## Acceptance criteria

- [x] The bot joins the server from the launch script with nobody at the VM
- [x] It follows an owner across 200 m of mixed terrain and recovers from getting stuck
- [x] `follow`, `stop` and `status` work from chat, from owners only
- [x] Chat addressed to the bot is answered in character through terva
- [x] Killing the controller stops the bot within 10 seconds
- [x] The plugin refuses any task outside the M1 whitelist

## Notes

**agent:claude-code/renki-groom** at 2026-10-28T23:24:20Z

Evidence from 2026-10-28 on the bot VM. (1) RenkiStart joins the server with nobody at the VM, every time. (3) follow, stop, wait here, status, set home and go home worked from chat for the owner Nimah; the non-owner jarl was refused at 17:42 with nothing sent. (6) The plugin's TaskRunner refuses anything outside capabilities and welcome.allowedTasks (TaskRunner tests); the controller's session refuses the same before sending. Still to see: a 200 m follow with a stuck recovery during the follow (recoveries were seen on the way home), the dead-man stop when the controller dies mid-follow, and the terva voice, which is not configured on the VM.

**agent:claude-code/renki-groom** at 2026-10-28T23:56:48Z

Voice verified in the game on 2026-10-28 through the cpa provider (gemma-4-12B-heretic-v3-mtp). At 18:47:41 Nimah asked about the weather and got an in-character answer 15 s later (the first call is slow while the model warms up). At 18:48:23 Nimah asked the time of day and got an answer in 1 s. A third question at 18:49:16 went unanswered because the controller restarted at 18:49:26 for the deploy of 3bffa7b; it was not a fault in the voice. Still open: the follow over 200 m with stuck recovery (check d) and killing the controller (check c).

**agent:claude-code/renki-groom** at 2026-10-28T23:59:24Z

Killing the controller verified in the game on 2026-10-28.

- At 18:58:16.8 I killed controller.exe on the VM while the bot was running after Nimah (task t-1, follow).
- The plugin logged "bridge: disconnected; retrying in 16 s" and kept following on its last order. It ran about 28 m, from (12.3, 32.0, 35.7) to (38.9, 32.3, 25.5).
- It then logged "no controller message for 10 s; stopping". After that the log has no walker or follow lines, only "retrying in 30 s": it stayed stopped, and nothing resumed on its own.
- At 18:58:51 I restarted with RenkiStart. At 18:59:04.8 the plugin reconnected, and the controller sent move_to home (why "after an outage") and "Something cut me off. Heading home."

The 10 s count starts at the plugin's last message from the controller, which came no later than the kill, so the stop came within 10 s of the kill.

**agent:claude-code/renki-groom** at 2026-10-29T00:16:18Z

Stuck recovery verified in the game on 2026-10-28; see the note on TKT-01M61T36N6 for the detector fix. Criterion 2 stays unticked on purpose. The long runs (about 300 m, with stuck recovery and escapes) were move_to from 'come to me'. The longest follow was the one at 18:58, about 95 m. Follow uses the same walker and detector, but nobody has done a 200 m follow yet.

**agent:claude-code/renki-groom** at 2026-10-29T00:18:48Z

Criterion 2 verified, superseding the note above that left it unticked. The follow at 19:01:22–19:03:46 on 2026-10-28 (t-3) went from home (-36.0, 11.8) to the slope at (-6.5, 315.2): 393 m walked, heights 28–39, and two stuck recoveries that got it free (19:01:44 and 19:03:31). It ended when a creature killed it at 19:03:46.
