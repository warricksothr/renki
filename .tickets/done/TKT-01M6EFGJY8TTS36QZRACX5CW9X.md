---
schema: 4
id: TKT-01M6EFGJY8TTS36QZRACX5CW9X
title: "CI: run just ci on every pull request"
type: task
status: done
status_reason: null
priority: high
due_on: null
labels:
  - tooling
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
updated_at: 2026-11-02T20:14:07Z
created_by:
  id: agent:claude-code/renki-plan-m4
  name: ""
updated_by:
  id: agent:claude-code/renki-m4-ci
  name: ""
extensions: {}
---

## Description

Goal 4.1 in docs/ROADMAP.md says `just ci` runs on every pull request. Nothing runs it today: there is no `.github/` directory, and the root `justfile` still says "There is no CI yet". Every other M4 agent relies on this gate, so this is the first M4 ticket to land. Split out of TKT-01M6DCFST1 (CI: run just ci automatically, require the plugin, real lint), item 1.

### What to build

- A workflow that runs `just ci` and nothing else on every pull request and on pushes to main, on Linux.
- Toolchains it needs, read from the repository rather than guessed: Go from `go.mod` (`go 1.27.0`), the .NET 8 SDK (`plugin/Renki.Tests/Renki.Tests.csproj` targets `net8.0`), Node 20 or newer (`panel/package.json` engines), Playwright Chromium through `just panel ui-install`, Python 3 for the `vm::test-*`, `broadcast-test` and `streaming::test` recipes, `just`, and the `git-ticket` binary for `just tickets-check`.
- Cache Go modules, NuGet packages and `panel/node_modules` so a run stays short.
- Remove the "There is no CI yet" comment from the root `justfile`, and make README prerequisites match what the workflow installs.

The Renki.Plugin build stays skipped in CI until TKT-01M6DCFST1's required-plugin child decides how CI compiles it. This ticket must not hide that skip; the required-plugin child makes it a failure.

### Files

- `.github/workflows/ci.yml` (new)
- `justfile` (the comment at the top, and `ci`)
- `AGENTS.md` ("`just ci` is the gate" paragraph) only if the wording changes
- `README.md` prerequisites

### Verification

- `just check ci` passes locally before the workflow is added.
- The workflow's first run on the pull request that adds it is green, and its log shows every component's `ci` and every `vm::test-*` recipe ran.
- A throwaway commit on a scratch branch that plants a Go compile error turns the run red. Link both runs in a note (this is goal 4.1's evidence).

## Acceptance criteria

- [x] A workflow runs just ci on every pull request and on pushes to main
- [x] The workflow installs Go, .NET 8, Node, Playwright Chromium, Python 3, just and git-ticket, and its first run is green
- [x] A planted Go compile error on a scratch branch turns the run red, and both runs are linked in a note
- [x] The root justfile no longer says there is no CI

## Implementation plan

.github/workflows/ci.yml runs on ubuntu-24.04 for every pull_request and for pushes to main. It sets up the toolchains, then runs `just ci` as its only gate step.

- Go: setup-go with go-version-file go.mod, with module and build caches.
- .NET: setup-dotnet with global-json-file plugin/global.json, which is new and pins SDK 8.0 (rollForward latestFeature). The runner image also has SDK 10, whose `dotnet format` disagrees with SDK 8's. NuGet is cached through actions/cache on ~/.nuget/packages, keyed on the plugin csproj files and nuget.config.
- Node 22 through setup-node, with the npm cache keyed on panel/package-lock.json. node_modules itself is not cached because `npm ci` deletes it.
- Python 3.12 through setup-python. The scripts use only the standard library.
- just through extractions/setup-just. git-ticket through `go install github.com/terva-sh/git-ticket/cmd/git-ticket@v0.24.0`.
- Playwright Chromium through `just panel ui-install`, with ~/.cache/ms-playwright cached, and its OS libraries through `just panel ui-install-system`. Playwright elevates with sudo itself.
- On failure, the run uploads .checks/panel-ui as an artifact.

The Renki.Plugin DLL skip stays visible: on GitHub Actions, plugin/justfile prints it as a `::warning title=Renki.Plugin not built::` annotation. scripts/check-test.sh uses grep, not rg, because the runner has no ripgrep. The root justfile header and README prerequisites now describe CI.

## Notes

**agent:claude-code/renki-m4-ci** at 2026-11-02T19:51:40Z

The first CI runs on PR #164 found three things that a workstation run of `just ci` hides:

1. Run 41509791593 failed the plugin's `dotnet format` lint, because the runner picked SDK 10 when no global.json was present. Fixed by plugin/global.json.
2. Run 41511564549, attempt 1, failed internal/memory TestPerceptionMemoryExcludesPreStateMenuAndReplacedConnections under -race. Filed as draft TKT-01M6EWJ9EE (Flaky gate: memory test races a replaced connection under -race) and not fixed here. TKT-01M6ESNXKF's app flake did not appear.
3. Attempt 2 failed check-test with `rg: command not found`. Fixed by switching scripts/check-test.sh to grep.

**agent:claude-code/renki-m4-ci** at 2026-11-02T20:14:07Z

Goal 4.1 evidence.

- Green: https://github.com/warricksothr/renki/actions/runs/41515661244 on PR #164 (final workflow; also https://github.com/warricksothr/renki/actions/runs/41514135257 before the cache/upload-artifact bump). The log shows `-- panel ci`, `-- controller ci`, `-- plugin ci`, all 16 `vm::test-*` recipes, `streaming::test`, `broadcast-test`, `check-test` and `tickets-check`; the plugin DLL skip appears as a "Renki.Plugin not built" warning annotation.
- Red: https://github.com/warricksothr/renki/actions/runs/41516639527 on scratch draft PR #166 (closed unmerged, branch deleted), which planted `var plantedCompileError int = "not an int"` in cmd/controller; `just controller lint`'s go vet failed the run.

Not built on the runner: the Renki.Plugin DLL (needs Valheim's assemblies). Left to TKT-01M6DCFST1's required-plugin child.

## Summary

Landed in PR #164: .github/workflows/ci.yml runs just ci on every pull request and push to main (ubuntu-24.04), green in run 41515661244 and red on a planted Go error in run 41516639527. Side fixes the runner exposed: plugin/global.json pins SDK 8 (the image's SDK 10 formatter disagrees), check-test uses grep instead of rg and its negated assertion now fails properly. Renki.Plugin DLL still skipped on the runner, shown as a warning annotation; TKT-01M6DCFST1 makes it required. New flake filed as draft TKT-01M6EWJ9EE.
