---
schema: 4
id: TKT-01M638V1VYY93J6343W7CJAD2X
title: "Stuck recovery: don't strafe off a ledge"
type: bug
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - movement
  - safety
assignees: []
milestone: M3
parent: TKT-01M62N2557R401S3E6JECSAPN4
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-29T07:26:24Z
updated_at: 2026-11-02T15:57:05Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-plan-m3
  name: ""
extensions: {}
---

## Description

Seen on the VM, 2026-10-29 01:40: sent home, the bot stood stuck on the deck west of the house at (-42.13, 36.09, 8.93). The second recovery, Strafe, walked it sideways off the deck edge, and it fell 2.5 m to the ground at (-45.22, 33.62, 9.68), under the house's level. From there the way home ran under the raised floor.

Strafe and the push after a climb move the bot sideways or forwards without looking where it will land. Before such a step it should check for ground within a step's drop (a ray down a little ahead of the bot), and choose the other side, or skip the strafe, when there is none.

### Why it matters for M3

Goal 3.1 in docs/ROADMAP.md needs one retreat that starts on the same raised base. A recovery that drops the bot off the deck during a flee or the walk home puts it under the floor, where the 2026-10-29 and 2026-10-31 runs got stuck.

### What the code does now

`Walker` in `plugin/Renki.Plugin/BodyTasks.cs` picks recoveries from `Steering` (`plugin/Renki.Core/Movement/Steering.cs`: Jump, Strafe, Repath, then Climb, Strafe, Repath). `Recovery.Strafe` sets `strafeUntil` and flips `strafeSign`, and the move direction becomes the cross product with up plus a little forward. Only when `dryGround` is set does the move get a `DryTravel.ClearanceDistance` probe; ordinary walks (follow, plain `move_to`) strafe unchecked. The climb push is in `Walker.Climb` with `plugin/Renki.Core/Movement/Climber.cs`.

### Approach

Put the decision in Renki.Core so it is testable: given support samples on both sides (heights below the feet at about 0.6 m and 1.2 m along each strafe direction), return left, right or skip. Treat a drop over 1 m, or no hit, as no ground. The plugin supplies the raycasts. Apply the same check to the climb's forward push. Emit a `walk` trace when a side is skipped for a drop.

### Files

- `plugin/Renki.Plugin/BodyTasks.cs` (`Walker`, the `Recovery.Strafe` case and `Climb`)
- `plugin/Renki.Core/Movement/Steering.cs` or a new Core class beside it, and `plugin/Renki.Core/Movement/Climber.cs`
- Tests in `plugin/Renki.Tests/CoreTests.cs` (existing Steering tests) or a new `StrafeLandingTests.cs`

### Verification

- `just check plugin test-core Steering` (or the new test class), with the captured deck positions above as a case
- `just check ci`

No live check is required by this ticket; the raised-base retreat run under goal 3.1 exercises it.

## Acceptance criteria

- [ ] Strafe and the climb's push do not step where the ground drops more than about a metre
- [ ] A trace says when a recovery skipped a side because of a drop
- [ ] A Core test with the captured deck positions shows the strafe picks the side with ground or skips; just check ci passes
