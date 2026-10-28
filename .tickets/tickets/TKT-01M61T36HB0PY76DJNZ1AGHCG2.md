---
schema: 4
id: TKT-01M61T36HB0PY76DJNZ1AGHCG2
title: Scaffold the BepInEx plugin project and its build
type: task
status: ready
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
updated_at: 2026-10-28T18:20:51Z
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

- [ ] `plugin/` builds against a game path given by configuration, with no game DLLs committed
- [ ] It references BepInEx 5, Harmony and assembly_valheim
- [ ] A just recipe builds the DLL, and another copies it into a configured BepInEx/plugins folder
- [ ] The plugin logs its version on load in a real game
- [ ] A C# test project under `plugin/` runs with `dotnet test` on Linux, with no game install, so the protocol types can be checked against the shared fixtures

## Notes

**agent:claude-code/renki-groom** at 2026-10-28T18:04:08Z

Grooming, 2026-10-28. The dev box has Valheim at E:\SteamLibrary\steamapps\common\Valheim (WSL: /mnt/e/...), so the build can reference valheim_Data/Managed from there. BepInEx 5 and HarmonyX can come from the BepInEx NuGet feed (nuget.bepinex.dev) rather than an install. Several hooks in section 10 touch non-public members (FejdStartup.m_profiles, PlayerController.FixedUpdate), so the build likely wants a publicized assembly_valheim (BepInEx.AssemblyPublicizer.MSBuild) instead of reflection everywhere. The .NET 8 SDK is installed in WSL; the plugin itself targets the game runtime (netstandard2.1 or net462), not net8.

**agent:claude-code/renki-groom** at 2026-10-28T18:12:23Z

Since TKT-01M61VAXH7 (Standardize on just as the task entry point): the plugin's recipes go in plugin/justfile, which the root picks up as the `plugin` module. It must implement the six shared verbs (build, test, lint, fmt, ci, clean), and `deploy` for the copy into BepInEx/plugins. plugin ci has to run on a machine with no game install, so it covers the dotnet test project rather than the DLL build. AGENTS.md, Tasks, has the rules.
