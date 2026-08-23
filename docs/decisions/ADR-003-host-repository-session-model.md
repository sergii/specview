# ADR-003: Host, Repository, and Session are first-class runtime entities

- Status: Accepted
- Date: 2026-08-21

## Context

Specview originally booted inside one configured repository and rendered that repository directly at `/`.

That is too low-level for a persistent host daemon. A developer may work on several repositories on the same laptop or dev box during one day, then return tomorrow and need to reconstruct what was active.

## Decision

Specview introduces three runtime entities above Intent artifacts:

```text
HOST
  |
  +-- REPOSITORY
        |
        +-- SESSION
        |
        +-- INTENT
        +-- EXECUTION
        +-- EVIDENCE
```

A Host is one machine running Specview. A Repository is the minimum top-level project entity. A Session records observed work by an execution actor inside a repository.

Linked Git worktrees belong to the same repository identity and may become child execution entities later rather than separate host-dashboard projects.

The default UI groups repositories by most recent observed session into Today, Yesterday, and Earlier. The host slice does not recursively index every Git repository on disk: a repository enters the catalog when Specview observes a supported execution process running from it.

Specification detection is read-only. Strong signatures are `.specview.yaml`, `.specify/`, `openspec/`, `.kiro/specs/`, and `_bmad-output/`. A plain `specs/` directory remains ambiguous.

Observed repository/session history is runtime state outside repositories. The first slice uses a host-level JSON state store behind a replaceable boundary. The planned SQLite projection may replace it without changing scanner, UI, or domain contracts.

## Consequences

- `specview` can start from any directory and no longer requires `.specview.yaml` to boot.
- `.specview.yaml` remains an optional repository-level override.
- tomorrow's dashboard can show repositories observed yesterday;
- specification frameworks remain adapters below Repository;
- future worktree, process, GitHub, and evidence views have a natural parent.

## Non-goals

Full-disk repository discovery, worktree UI, company/workspace folders, repository search, multi-host federation, Claude/OpenCode discovery, and SQLite implementation are not part of this slice.
