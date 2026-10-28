---
schema: 4
id: TKT-01M61T36HB0PY76DJNZ1AGHCG2
title: Scaffold the BepInEx plugin project and its build
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - plugin
  - tooling
assignees: []
milestone: M1
parent: TKT-01M61T36B839WT7KEZ14TDMB8B
origin: null
dependencies: []
blocks_on: none
references: []
moved_to: null
claim: null
archive: null
created_at: 2026-10-28T17:49:28Z
updated_at: 2026-10-28T23:24:05Z
created_by:
  id: agent:claude-code/valheim-setup
  name: Claude Code
updated_by:
  id: agent:claude-code/renki-groom
  name: Claude Code
extensions: {}
---

## Description

A C# BepInEx 5 plugin project under `plugin/`. Valheim's managed
assemblies can't be redistributed, so the build references them from a
configured game install rather than from the repository.

## Acceptance criteria

- [x] `plugin/` builds against a game path given by configuration, with no game DLLs committed
- [x] It references BepInEx 5, Harmony and assembly_valheim
- [x] A just recipe builds the DLL, and another copies it into a configured BepInEx/plugins folder
- [x] The plugin logs its version on load in a real game
- [x] A C# test project under `plugin/` runs with `dotnet test` on Linux, with no game install, so the protocol types can be checked against the shared fixtures

## Implementation plan

plugin/ holds three projects. Renki.Core (netstandard2.1, no Unity) has the protocol types, the bridge client, the task runner, the safety rules and the movement maths. Renki.Plugin (netstandard2.1) is the BepInEx 5 plugin, the thin layer that connects Core to the game. Renki.Tests (net8.0, xunit) tests Core against internal/protocol/testdata. JSON is Newtonsoft 13.0.2, which the game already ships: Core takes it from NuGet for tests, and the plugin is deployed without it. Game assemblies are referenced from $(ValheimDir)/valheim_Data/Managed, with ValheimDir coming from VALHEIM_DIR or plugin/game.local.props (gitignored), and assembly_valheim is publicized with BepInEx.AssemblyPublicizer.MSBuild. BepInEx.Core 5.4.21 comes from nuget.bepinex.dev. plugin/justfile implements the six shared verbs and deploy. build and ci always cover Core and the tests; the plugin DLL builds only when the game path is configured, and says so when it skips.

## Notes

**agent:claude-code/renki-groom** at 2026-10-28T18:04:08Z

Grooming, 2026-10-28. The dev box has Valheim at E:\SteamLibrary\steamapps\common\Valheim (WSL: /mnt/e/...), so the build can reference valheim_Data/Managed from there. BepInEx 5 and HarmonyX can come from the BepInEx NuGet feed (nuget.bepinex.dev) rather than an install. Several hooks in section 10 touch non-public members (FejdStartup.m_profiles, PlayerController.FixedUpdate), so the build likely wants a publicized assembly_valheim (BepInEx.AssemblyPublicizer.MSBuild) instead of reflection everywhere. The .NET 8 SDK is installed in WSL; the plugin itself targets the game runtime (netstandard2.1 or net462), not net8.

**agent:claude-code/renki-groom** at 2026-10-28T18:12:23Z

Since TKT-01M61VAXH7 (Standardize on just as the task entry point): the plugin's recipes go in plugin/justfile, which the root picks up as the `plugin` module. It must implement the six shared verbs (build, test, lint, fmt, ci, clean), and `deploy` for the copy into BepInEx/plugins. plugin ci has to run on a machine with no game install, so it covers the dotnet test project rather than the DLL build. AGENTS.md, Tasks, has the rules.

**agent:claude-code/renki-groom** at 2026-10-28T18:50:41Z

Built against the game (assembly_valheim 0.221.x on the dev box) and unit-tested off-game; not yet run inside Valheim. In review = check it in game; docs/TESTING.md has the steps.

**agent:claude-code/renki-groom** at 2026-10-28T23:24:04Z

Seen on the bot VM: BepInEx logs 'renki 0.1.0 loaded; dialling ws://127.0.0.1:8765/ws/' at every start.

## Summary

The plugin project builds against the game's assemblies from configuration, tests run on Linux without the game, just plugin build/deploy work, and it loads in the real game.
