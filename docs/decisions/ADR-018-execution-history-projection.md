# ADR-018 - Execution History Projection

## Status

Accepted.

## Context

Specview already persists logical `ExecutionSession` history in Host catalog v2. The Host dashboard groups repositories by recent activity, and MCP exposes only active sessions. Completed sessions therefore exist as valid local history but are not directly readable through a dedicated human or MCP surface.

## Decision

Add one read-only execution-history projection over the existing Host catalog. The projection is derived and introduces no new persistence or authority.

A history record preserves:

- repository ID, name, and local root;
- logical session ID and identity kind;
- adapter and agent;
- cwd and worktree root when known;
- diagnostic process IDs;
- started, last-seen, and ended timestamps;
- active state.

History is ordered by most recent activity (`last_seen_at`) descending with deterministic tie-breaking by repository ID and session ID.

The Web surface exposes `/history`. MCP adds `get_execution_history` with no arguments and returns the same normalized projection contract.

## Authority

The Host catalog remains the only persistence source for historical local execution observations. H29 does not:

- create another history database;
- infer missing sessions;
- regroup ended legacy PID fragments beyond the H26 migration rules;
- turn PID into logical identity;
- mutate repositories or agents;
- federate historical Host catalog rows across machines.

Federation remains snapshot-oriented and unchanged.

## Consequences

Completed work becomes inspectable without reading `catalog.json`, and agent clients can query the same history humans see. Because the projection is derived from catalog v2, a future implementation replacement can reproduce it from language-neutral fixtures without preserving Go-specific internals.
