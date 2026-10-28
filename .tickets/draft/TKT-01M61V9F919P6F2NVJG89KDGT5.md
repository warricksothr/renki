---
schema: 4
id: TKT-01M61V9F919P6F2NVJG89KDGT5
title: Gate pushes on a local vuoro review
type: chore
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - tooling
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
created_at: 2026-10-28T18:10:22Z
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

renki is developed in the open on github.com/warricksothr/renki, with no
human code reviewer behind most changes. Run vuoro's headless review on the
workstation before a branch is pushed, as a gate alongside whatever reviews
the pull request on GitHub.

### What vuoro offers today

`vuoro review [-C dir] [-preset name | -profile name [-model m]] [-json]
[-timeout d] [range]` reviews a local commit range in its own clone and
worktree, prints the findings as Markdown (or one JSON envelope), and never
posts. The exit status is 0 clean, 1 findings, 2 usage or empty range, 3 failed
or timed out, and 130 interrupted. vuoro's docs/plan.md section 6.7 and decision
D35 are the spec.

### Constraints that shape the integration

- **Commits only.** Uncommitted work is not reviewed, and a dirty tree only
  warns. The recipe runs after committing and before pushing.
- **No default range here.** Without a range it uses the upstream, then
  `origin/HEAD`, then `origin/main`. Agent branches in this repo have no
  upstream, so the recipe passes `origin/main...HEAD` explicitly.
- **Presets are the person's, not the repo's.** Reviewers come from vuoro's
  own config.json (`presets`, `profiles`; built-ins are code, security and
  documentation). Per-repository review instructions are on vuoro's v1.0
  list, not built yet ("a repository never picks the reviewers"). Until then,
  renki cannot ship its own checklist (the protocol doc as the contract, the
  section 8 safety rules, the shared fixtures).
- **Reviewers run on Claude Code** with user and project settings skipped,
  so authentication has to come through the environment vuoro is launched in.
- The workstation has no vuoro config.json in WSL yet, so a run gets one
  code reviewer on the default model.

## Acceptance criteria

- [x] A `just review` recipe runs `vuoro review` on `origin/main...HEAD` (overridable), passing through -preset and -json
- [x] AGENTS.md tells agents to run `just review` before pushing a branch, and how to treat exit 1 (fix, or answer each finding in the PR description)
- [ ] A vuoro preset for renki is defined in the workstation config, and its name is recorded here
- [ ] One real run against a renki branch is recorded in a note: findings, time and cost

## Notes

**agent:claude-code/renki-groom** at 2026-10-28T18:12:23Z

The recipe and the AGENTS.md rule landed with TKT-01M61VAXH7 (Standardize on just as the task entry point): `just review [range] [vuoro flags]`, defaulting to origin/main...HEAD. Still open: a renki preset in the workstation's vuoro config, and one real run recorded here.
