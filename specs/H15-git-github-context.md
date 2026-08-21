---
specview:
  status: in_progress
---

# H15 - Git and GitHub Context

## Goal

Add portable source-control context to the repository execution page without coupling the core domain to one operating system or one forge provider.

```text
RepositoryContext
├── GitContext
│   ├── origin remote
│   └── worktrees[]
│       ├── branch / HEAD / dirty state
│       ├── upstream
│       ├── ahead / behind
│       └── last commit
│
└── ProviderContext
    └── GitHubAdapter
        ├── open pull request
        └── check rollup
```

The architectural rule is:

> core/domain stays portable; OS-specific process discovery is an execution-adapter mechanic; forge-specific remote context is a provider-adapter mechanic.

## Authority

Local Git is authoritative for:

- repository remote configuration;
- worktree paths;
- branch / detached state;
- HEAD revision;
- dirty state;
- configured upstream;
- ahead / behind relative to the locally known upstream ref;
- last local commit subject.

GitHub is an optional remote projection for:

- open pull requests for branches currently represented by observed worktrees;
- PR head/base state;
- GitHub check rollup.

GitHub context must never override or invalidate local Git context.

## No implicit fetch

Specview remains read-only and must not run `git fetch` as part of observation.

Ahead / behind therefore means:

> divergence from the locally known upstream tracking ref.

If the tracking ref is stale, Specview shows the local truth it can prove rather than silently performing network I/O.

## Provider adapter

H15 introduces a provider-neutral boundary:

```text
ProviderAdapter
├── Supports(remote)
└── Inspect(GitContext) -> ProviderContext
```

The first concrete provider is GitHub through the locally installed `gh` CLI.

Using `gh` keeps authentication outside Specview. Specview does not persist GitHub tokens or credentials.

Supported GitHub origin forms in this slice include:

```text
git@github.com:owner/repo.git
https://github.com/owner/repo.git
ssh://git@github.com/owner/repo.git
```

## Failure semantics

Remote provider failure is degradable.

```text
Git inspection succeeds
GitHub unavailable
      ↓
repository page still renders Git context
```

Examples:

- no GitHub remote -> provider section says no GitHub origin;
- `gh` not installed -> Git context remains visible and GitHub reports unavailable;
- `gh` unauthenticated / remote API failure -> Git context remains visible and provider error is observable.

A provider failure must not return a repository-level error when local Git inspection succeeded.

## Remote refresh

The repository page can reload frequently due to execution SSE events. Remote provider context is therefore cached for 30 seconds by the source-control service.

Local Git state is re-read for each repository projection; GitHub is not queried every two seconds.

## Checks are context, not Evidence

GitHub check rollup in H15 is a source-control/provider projection only.

It does not automatically become normalized Evidence and does not satisfy Acceptance policy.

A later bridge may explicitly translate provider CI facts into the H11 Evidence contract with revision identity.

## UI

The repository page keeps the H13 structure and enriches it:

```text
ACTIVE NOW
  logical execution sessions

WORKTREES
  branch | path | HEAD | upstream + ahead/behind | dirty + active agent

GITHUB
  PR | title / head -> base | state | checks

SPECIFICATION
  existing Intent projection
```

## Acceptance criteria

- [ ] `RepositoryContext` separates local Git from optional provider context.
- [ ] local Git inspection is portable and contains no Darwin/Linux process mechanics.
- [ ] worktrees show upstream and ahead/behind without implicit `git fetch`.
- [ ] worktrees preserve branch, HEAD, dirty count, and last commit context.
- [ ] `ProviderAdapter` is provider-neutral.
- [ ] GitHub is implemented as the first provider adapter.
- [ ] GitHub adapter uses local `gh` authentication and does not persist credentials.
- [ ] GitHub remote parsing supports SSH/scp and HTTPS forms used by GitHub.
- [ ] only open PRs for branches represented by observed worktrees are projected.
- [ ] GitHub check rollup distinguishes passed, failed, pending, and skipped states.
- [ ] unknown check states are never interpreted as passed.
- [ ] provider failure does not discard local Git context.
- [ ] provider context is cached so SSE refresh does not repeatedly query GitHub.
- [ ] H13 active execution/worktree mapping remains intact.
- [ ] GitHub checks remain context and are not silently promoted into Evidence.
- [ ] gofmt, module verification, go vet, race tests, build, and release cross-build pass.

## Out of scope

- implicit or scheduled `git fetch`;
- GitLab / Bitbucket / Forgejo provider implementations;
- GitHub write operations, merge, review, or branch mutation;
- GitHub webhook ingestion;
- translating GitHub checks into H11 Evidence;
- Acceptance policy;
- SQLite persistence of provider snapshots;
- multi-host federation.
