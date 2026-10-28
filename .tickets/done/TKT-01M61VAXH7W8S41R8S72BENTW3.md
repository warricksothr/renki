---
schema: 4
id: TKT-01M61VAXH7W8S41R8S72BENTW3
title: Standardize on just as the task entry point, with routing
type: chore
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - tooling
  - docs
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
created_at: 2026-10-28T18:11:09Z
updated_at: 2026-10-28T18:12:23Z
created_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

renki has two components in two toolchains (Go controller, C# plugin), plus
tooling around them: tickets, and the vuoro review before a push. People, agents
and CI should all reach every one of those tasks the same way, so that a recipe
is the single place a task is defined and a green `just ci` means a green CI.

The root justfile does the routing. Its verbs (build, test, lint, fmt, ci,
clean) fan out to every component that exists. Each component keeps its own
recipes in a just module (`just controller build`, `just plugin build`), which
runs in that component's directory. Tooling that spans the repository (review,
tickets) lives at the root.

## Acceptance criteria

- [x] The root justfile routes build, test, lint, fmt, ci and clean to every component module present
- [x] Controller recipes live in a `controller` module; a `plugin` module is picked up when plugin/justfile exists
- [x] `just review` runs vuoro on origin/main...HEAD, and `just tickets-check` runs git ticket check --strict
- [x] AGENTS.md states the convention: tasks are recipes, root verbs route, components implement the shared verbs

## Implementation plan

Move the Go recipes from the root justfile into controller.just at the root, loaded as `mod controller`, so they run beside go.mod. Load the plugin as `mod? plugin`, so it stays optional until plugin/justfile exists. Root verbs loop over the components present and call `just <component> <verb>`, so adding a component means one line in the components list. Add root recipes review (vuoro, defaulting to origin/main...HEAD and taking flags after) and tickets-check, and make ci run every component's ci, then tickets-check. AGENTS.md gets a Tasks section with the convention. Tick the vuoro ticket's recipe and AGENTS.md criteria from here.

## Summary

The root justfile is routing plus repo-wide tooling. build, test, lint, fmt, clean and ci go through a private `route` recipe that runs the verb in each component listed in `components` and present in `just --summary`. An absent component is skipped with a note, and a component present but missing the verb fails. The Go recipes moved unchanged into controller.just (`mod controller`). The plugin is `mod? plugin`, so plugin/justfile joins as soon as it exists. Root tooling: `review` (vuoro on origin/main...HEAD, flags passed through, exit status passed through) and `tickets-check`; ci is every component's ci, then tickets-check. AGENTS.md has a Tasks section with the convention. Checked: just ci green; just build from cmd/; a stub plugin/justfile is routed to, and its missing `test` fails `just test`.
