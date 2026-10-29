---
schema: 4
id: TKT-01M6348M7PNSNXWCW877A7FTDH
title: "Perception and panel: draw the walls of buildings near the bot"
type: task
status: done
status_reason: null
priority: high
due_on: null
labels:
  - plugin
  - protocol
  - controller
assignees: []
milestone: null
parent: TKT-01M628X106RD9FFR8YBXV7K5HT
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T06:06:26Z
updated_at: 2026-10-29T06:41:43Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

Nimah, 2026-10-29: the map knows where beds, doors and fires are but not
the walls, so the base is a scatter of markers with no rooms. Perception
leaves walls and floors out on purpose (6.2): there are hundreds of them
and nothing the bot does needs each one.

A wall does not move, so the plugin works out once per piece whether it
is a wall (tall, thin and long, from the piece's own colliders in its own
frame) and where its foot runs, and sends the walls within the
perception radius as segments in a new `layout` message (6.9), only when
that set changes. The occupancy grid spike (TKT-01M62ACH) can use the
same data for paths later.

## Acceptance criteria

- [x] The plugin sends the walls near the bot as segments in a layout message when they change, with a protocol fixture
- [x] The controller keeps the newest layout, the audit log records it, and a new panel page gets it
- [x] The panel map draws the walls under the markers, so rooms, doors and chests read in place
- [ ] A scan with layout stays within the perception budget

## Notes

**agent:claude-code/renki-groom** at 2026-10-29T06:37:00Z

Deployed and seen on the VM at home: 72 walls in range, and the panel draws the house outline with the gap at the door (checked against the real layout frame in a local preview).

AC 4 (budget) is left unticked. In steady state the scan averages 1.1–1.25 ms with layout on (1.0 before), and the worst over 30 s is 2.5–4.8 ms (2.4 before). The first scan after login was already ~40 ms (JIT and first classification) and is now ~56 ms: measuring is about 0.1 ms per piece the first time, capped at 200 pieces per scan, so a base fills in over a few scans. If that spike matters, lower MeasurePerScan.

Accepted review finding: walls below the bot are drawn at full strength, so an upstairs bot sees the ground floor's walls over its own doorways. A two-sided band would also fade every building downhill (the old shelter sits 3.7 m below the house), and the bot is on the ground floor far more often, so only upper storeys fade.

Half walls (1 m) and floors are not walls by design (WallShape: at least 1.2 m tall, twice as long as thick, at most ~1 m thick).

## Summary

Plugin sends wall foot lines in a layout message (6.9) on change; the audit log keeps the newest; the map draws them, fading upper storeys. Seen in use on the VM, 2026-10-29. AC 4 left unticked: steady scans stay around 1.1-1.25 ms, but the first scan after login rose from about 40 to 56 ms (see the note).
