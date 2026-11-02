---
schema: 4
id: TKT-01M6EFGJZKWV5XZTND3CH55E7W
title: "CI: fail when Renki.Plugin is skipped, and show skips in check"
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - tooling
  - plugin
assignees: []
milestone: M4
parent: TKT-01M6DCF2ABDDNG3QF369NDPBT5
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-11-02T15:54:40Z
updated_at: 2026-11-02T15:54:40Z
created_by:
  id: agent:claude-code/renki-plan-m4
  name: ""
updated_by:
  id: agent:claude-code/renki-plan-m4
  name: ""
extensions: {}
---

## Description

Goal 4.1 says `just ci` compiles Renki.Plugin or fails. Today `plugin/justfile`'s `build` recipe skips the plugin DLL with one stderr line when neither `VALHEIM_DIR` nor `plugin/game.local.props` is set, and `scripts/check.sh` reports PASS without showing that line. A compile error in the roughly 9k lines of `plugin/Renki.Plugin` first shows up at `just package` or `just vm install`. Split out of TKT-01M6DCFST1 (CI: run just ci automatically, require the plugin, real lint), item 2.

### What to build

- `RENKI_REQUIRE_PLUGIN=1` turns the skip in `plugin/justfile` (`has_game`, `skip_note`, `build`) into a failure.
- Set it wherever a plugin DLL is shipped: the root `package` recipe, which `just vm install` and `just vm deploy` call.
- `scripts/check.sh` prints any line containing "skipped" beside its PASS line, covered by `scripts/check-test.sh`.
- CI compiles the plugin. The game assemblies cannot be committed or redistributed (`plugin/Directory.Build.props`), so this needs an owner decision between: reference-only stub assemblies generated from the public API surface; a self-hosted runner on a machine with the game; or leaving CI's plugin build skipped and requiring it only on deploy. Record the decision here before building it.

### Files

- `plugin/justfile` (`has_game`, `skip_note`, `build`, `ci`)
- `plugin/Directory.Build.props`, `plugin/Renki.Plugin/Renki.Plugin.csproj` (only for the stub option)
- `justfile` (`package`)
- `scripts/check.sh`, `scripts/check-test.sh`

### Verification

- `just check check-test` covers the skipped-line output.
- With `VALHEIM_DIR` unset, `RENKI_REQUIRE_PLUGIN=1 just plugin build` exits non-zero, and `just plugin build` without it still skips.
- `just check ci` passes.

## Acceptance criteria

- [ ] RENKI_REQUIRE_PLUGIN=1 makes a skipped Renki.Plugin build fail, and the package recipe sets it
- [ ] scripts/check.sh prints skipped lines beside PASS, covered by check-test
- [ ] How CI compiles Renki.Plugin is decided by the owner and recorded here, and CI does it or says why not
