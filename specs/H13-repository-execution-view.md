---
specview:
  status: done
---

# H13 - Repository Execution View

## Goal

Make a repository page answer the next question after host discovery:

> Where is execution happening inside this repository right now?

H12 establishes the host-level repository activity index. H13 adds a read-only repository execution projection without changing the top-level rule that a Project is a concrete Git repository.

## User flow

```text
Host
  -> Repository
      -> active execution
      -> worktrees
      -> specification projection
```

Opening a repository must show:

- canonical repository path;
- `origin` remote URL when one exists, otherwise `No remote`;
- currently detected Codex execution contexts;
- exact process cwd;
- the Git worktree that contains that cwd;
- logical process grouping so helper/app-server processes sharing one cwd do not render as separate agents;
- all Git worktrees returned by `git worktree list --porcelain`;
- branch or detached state;
- short HEAD revision;
- dirty change count;
- current specification convention and existing work-item projection.

## Authority

H13 remains read-only.

```text
Git
  repository / worktree / branch / HEAD / dirty state / remote

runtime observation
  active agent process / cwd

repo-native artifacts
  specification convention / Intent
```

Specview does not create worktrees, change branches, add remotes, or modify repository files.

## Repository identity

The host page continues to show one row per canonical Git repository.

Linked worktrees must not become duplicate top-level projects.

```text
repository
  ├── main worktree
  ├── linked worktree A
  └── linked worktree B
```

The repository page may show all of those worktrees as children.

## Execution context

The current Codex observer may expose multiple OS processes for one logical interactive execution, for example a CLI process plus an app-server helper.

H13 groups observed Codex processes by:

```text
agent + cwd + repository
```

and renders one execution context with a process count.

This is a POC heuristic. H14 will introduce an Execution Adapter Contract where each agent can provide a stronger logical session identity.

## Git discovery

Repository execution metadata is derived only from Git commands rooted at the known repository:

```text
git remote get-url origin
git worktree list --porcelain
git status --porcelain
```

No recursive filesystem repository crawl is introduced.

## Specification state

The existing specification projection remains available below execution state.

If no convention is recognized, the repository page must say so without creating files:

```text
No recognized specification pattern.
Specview is observing this repository read-only.
```

## Acceptance criteria

- [x] Clicking a repository opens a repository execution page.
- [x] A repository with no remote shows `No remote`.
- [x] A repository with `origin` shows that exact configured remote value.
- [x] `git worktree list --porcelain` is projected into repository child rows.
- [x] Worktree branch, HEAD, and dirty count are visible.
- [x] Active Codex cwd is visible.
- [x] Multiple Codex helper processes sharing a cwd are grouped into one logical row.
- [x] Linked worktrees remain one top-level repository on the host page.
- [x] Existing Specview / GitHub Spec Kit / OpenSpec work-item projection still renders.
- [x] No repository or specification files are written by observation.
- [x] gofmt, module verification, go vet, race tests, build, and release cross-build pass.

## Verification

Automated verification passed on the H13 implementation:

```text
gofmt                         PASS
module verification            PASS
go vet ./...                   PASS
go test -race ./...            PASS
go build ./cmd/specview        PASS
release cross-build            PASS
```

The Git projection test covers an exact `origin` value and dirty-state counting. A real macOS host acceptance run confirmed:

```text
repository: wms
worktrees:  main + project-login
execution:  Codex active in main
processes:  2 OS processes -> 1 logical execution row
remote:     No remote
specs:      No recognized specification pattern
```

## Out of scope

- Claude Code and OpenCode adapters;
- durable logical execution-session identity;
- worktree creation/deletion;
- GitHub PR/CI projection;
- SQLite migration;
- Acceptance policy;
- lifecycle stages beyond the current specification projection.
