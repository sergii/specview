---
specview:
  status: in_progress
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

## Acceptance criteria

- [ ] `ExecutionAdapter` is a first-class interface.
- [ ] `ExecutionSession` separates logical execution identity from OS process IDs.
- [ ] `ExecutionRegistry` aggregates one or more adapters.
- [ ] Host runtime uses the registry rather than constructing a Codex scanner directly.
- [ ] Repository execution view consumes normalized sessions and contains no direct Codex discovery call.
- [ ] `specview doctor` reaches Codex diagnostics through the adapter contract.
- [ ] Codex helper processes sharing repository + cwd normalize into one logical session.
- [ ] Multiple adapters can coexist without changing the catalog or web domain model.
- [ ] Failure in one adapter does not discard sessions from a healthy adapter.
- [ ] H13 repository/worktree UI semantics remain unchanged.
- [ ] No execution adapter writes repository files.
- [ ] gofmt, module verification, go vet, race tests, build, and release cross-build pass.

## Out of scope

- Claude Code implementation;
- OpenCode implementation;
- native cross-restart agent session IDs;
- durable execution-session persistence migration;
- agent control/start/stop commands;
- GitHub PR/CI projection;
- Acceptance policy.
