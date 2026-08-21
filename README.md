# Specview

Specview is a local-first execution control plane for repo-native, spec-driven software development.

It observes what is running on a host, maps activity to concrete Git repositories and worktrees, detects supported specification conventions, and projects Intent, Execution, and Evidence without making the filesystem stop being the source of truth.

## Current vertical slice

```text
HOST
  ↓
REPOSITORY
  ↓
WORKTREE / EXECUTION SESSION
  ↓
INTENT | EXECUTION | EVIDENCE
```

The current implementation includes:

- a host-level Today / Yesterday / Earlier repository activity index;
- passive Codex process discovery on macOS and Linux;
- canonical Git repository mapping so linked worktrees do not duplicate top-level projects;
- a repository execution page with active sessions, worktrees, branch, HEAD, dirty state, and remote;
- a normalized Execution Adapter Contract so runtime and UI are not Codex-specific;
- read-only detection for Specview, GitHub Spec Kit, OpenSpec, Kiro, and BMAD conventions;
- normalized Intent adapters for Specview, GitHub Spec Kit, and OpenSpec;
- revision-scoped normalized Evidence;
- SSE refresh for live host activity;
- structured development/production logging;
- `bin/install` and `bin/doctor` developer workflows.

## Run locally

The canonical development update/run command is:

```bash
bin/install
```

It fetches and fast-forwards the current branch, checks formatting and modules, runs vet and race tests, builds a fresh binary, and starts Specview.

Then open:

```text
http://127.0.0.1:7331
```

For execution discovery diagnostics:

```bash
bin/doctor
```

## Logging

Development defaults to colorized structured console logs through `log/slog` + `tint`.

Production can use JSON without changing application log calls:

```bash
SPECVIEW_ENV=production ./bin/specview
```

See `docs/observability/logging.md` for the logging contract.

## Specification conventions

Strong read-only signatures currently detected:

```text
.specview.yaml  -> Specview
.specify/       -> GitHub Spec Kit
openspec/       -> OpenSpec
.kiro/specs/    -> Kiro
_bmad-output/   -> BMAD
```

Plain `specs/` is intentionally ambiguous and does not auto-select an adapter.

## Execution adapters

Execution observation is normalized behind:

```text
ExecutionAdapter
      ↓
ExecutionRegistry
   ↙         ↘
Host Runtime  Repository View
```

`CodexExecutionAdapter` is the first implementation. Multiple OS processes that represent one Codex execution context are normalized into one logical execution session while process IDs remain diagnostic details.

Claude Code and OpenCode adapters are intentionally not implemented yet.

## Intent, Execution, Evidence

```text
INTENT                    EXECUTION                 EVIDENCE
what should be true?      what is happening now?   why should we trust it?
```

Files describe durable Intent. Runtime observation describes Execution. Verifiers produce revision-scoped Evidence. UI state is a projection over those facts, not a new source of truth.

## Development commands

```bash
bin/install
bin/dev install
bin/dev check
bin/dev build
bin/doctor
```

`bin/install` is the short alias for the most common `bin/dev install` workflow.
