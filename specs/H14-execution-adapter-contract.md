---
specview:
  status: done
---

# H14 - Execution Adapter Contract

## Goal

Remove Codex-specific discovery from the host runtime and repository projection.

Specview must observe execution through a normalized adapter contract so additional agents can be added without changing the Host, Repository, Worktree, or web projection models.

```text
ExecutionAdapter
      ↓
ExecutionRegistry
   ↙         ↘
Host Runtime  Repository View
```

## Normalized session

An adapter returns logical execution sessions:

```text
ExecutionSession
├── adapter
├── id
├── agent
├── cwd
├── repository_root
├── worktree_root
└── process_ids[]
```

`process_ids` are diagnostic/runtime evidence, not the logical identity of a session.

For the current Codex POC, logical identity is derived from:

```text
adapter + repository + cwd
```

so a CLI process and helper/app-server process sharing the same execution context collapse into one session.

A future adapter may replace that heuristic with an agent-native session identifier without changing consumers.

## Adapter contract

Each execution adapter must provide:

- a stable adapter name;
- current logical sessions;
- adapter-specific diagnostics for `specview doctor`.

The registry aggregates adapters and exposes a normalized execution source to:

- the host polling runtime;
- repository execution projection;
- diagnostics.

## Codex adapter

H14 introduces the first concrete adapter:

```text
CodexExecutionAdapter
```

It owns:

- process matching;
- cwd discovery;
- Git repository resolution;
- process-to-logical-session grouping;
- Codex diagnostics.

Darwin and Linux process mechanics remain platform-specific implementation details below the adapter boundary.

## Failure semantics

One broken execution adapter must not erase healthy observations from other adapters.

The registry logs adapter failures independently. If at least one adapter produced sessions, healthy sessions remain observable. If every adapter fails and no sessions can be produced, the registry returns an error.

## Repository projection

The repository page consumes the same normalized execution source as the host runtime.

It must not call Codex-specific discovery directly.

Worktree mapping remains read-only and Git-authoritative.

## Path identity

Filesystem identity is normalized independently of display spelling.

Specview resolves symlink-equivalent paths when comparing repository roots, worktree roots, and execution session identity. This handles cases such as macOS `/var/...` and `/private/var/...` without rewriting the path shown to the user.

The same shared path-identity logic is used above the Darwin/Linux scanner boundary.

## Acceptance criteria

- [x] `ExecutionAdapter` is a first-class interface.
- [x] `ExecutionSession` separates logical execution identity from OS process IDs.
- [x] `ExecutionRegistry` aggregates one or more adapters.
- [x] Host runtime uses the registry rather than constructing a Codex scanner directly.
- [x] Repository execution view consumes normalized sessions and contains no direct Codex discovery call.
- [x] `specview doctor` reaches Codex diagnostics through the adapter contract.
- [x] Codex helper processes sharing repository + cwd normalize into one logical session.
- [x] Multiple adapters can coexist without changing the catalog or web domain model.
- [x] Failure in one adapter does not discard sessions from a healthy adapter.
- [x] H13 repository/worktree UI semantics remain unchanged by design.
- [x] No execution adapter writes repository files.
- [x] gofmt, module verification, go vet, race tests, build, and release cross-build pass.

## Verification

Real macOS acceptance on 2026-08-22 passed:

```text
bin/dev check
  formatting             PASS
  module verification    PASS
  go vet                 PASS
  go test -race ./...    PASS

./scripts/build-release.sh dev-local
  linux/amd64             PASS
  linux/arm64             PASS
  darwin/amd64            PASS
  darwin/arm64            PASS

bin/doctor
  codex adapter           PASS
  matched processes       2
  cwd resolution          PASS
  Git repository          PASS
```

Unit coverage includes:

- multi-adapter aggregation;
- adapter failure isolation;
- all-adapters-failed error semantics;
- process-to-logical-session grouping;
- generic future-agent projection into a repository worktree;
- Linux diagnostics preserving cwd/repository context required by the normalized session contract;
- symlink-equivalent filesystem path identity.

GitHub Actions run `32532744655` for commit `c974649` reported failure before exposing workflow step metadata (`steps: null`). No project gate result was available from that run, so the reproducible local full gate above is used for H14 acceptance while the CI harness issue remains separate.

## Out of scope

- Claude Code implementation;
- OpenCode implementation;
- native cross-restart agent session IDs;
- durable execution-session persistence migration;
- agent control/start/stop commands;
- Windows process discovery;
- GitHub PR/CI projection;
- Acceptance policy.
