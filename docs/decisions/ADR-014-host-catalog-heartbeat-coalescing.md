# ADR-014 - Host Catalog Heartbeat Coalescing

## Status

Accepted for H25.

## Context

ADR-013 intentionally deferred Host catalog heartbeat write coalescing until after v0.0.1. The v0.0.1 catalog updates in-memory `LastSeenAt` values on every execution observation and persists every heartbeat-only change as a new atomic JSON snapshot.

With the current two-second execution scan cadence, one continuously active Host can therefore write up to roughly 1,800 catalog snapshots per hour even when no repository/session lifecycle fact changes.

The live Execution model, Web material fingerprinting, SQLite index behavior, MCP, and federation already avoid treating heartbeat timestamps as material product changes. The remaining churn is isolated to the legacy Host catalog persistence layer.

## Decision

H25 keeps execution discovery cadence unchanged and coalesces only heartbeat-only catalog persistence.

The persistence rules are:

1. In-memory `Repository.LastSeenAt` and active `Session.LastSeenAt` still advance on every observation.
2. Material lifecycle changes persist immediately:
   - new repository;
   - new active session;
   - active session ending;
   - repository convention or detection-error change.
3. Heartbeat-only changes are persisted at most once per 30-second coalescing interval.
4. A successful material save also persists any pending heartbeat timestamps and clears heartbeat dirtiness.
5. Graceful Host runtime shutdown flushes any pending heartbeat state.
6. Reopening an existing catalog starts with no in-process throttle history; the first subsequent heartbeat may persist immediately. This intentionally favors freshness after restart over preserving the previous process's throttle window.
7. The persisted catalog schema remains version 1. H25 changes persistence cadence, not data shape.

## Crash semantics

A hard process crash may lose at most one coalescing window of heartbeat-only timestamp freshness. It must not lose a lifecycle transition that `Observe` successfully returned after an immediate material save.

With a 30-second interval and continuous activity, heartbeat-only writes are bounded to approximately 120 snapshots/hour, a roughly 15x reduction from the v0.0.1 baseline of 1,800/hour, excluding immediate lifecycle writes.

## API semantics

`Catalog.Observe` continues to report whether the in-memory catalog changed. Callers must not infer that `changed == true` means a disk write occurred.

This preserves the existing observation API while making persistence an internal concern.

`Catalog.Flush` is added as an explicit durability boundary for graceful shutdown and tests. A no-op flush is valid when no heartbeat state is pending.

## Preserved boundaries

H25 must not change:

- execution scan cadence;
- logical `ExecutionSession` identity;
- Host identity or repository identity;
- MCP wire contracts;
- HostSnapshot v1 or federation correlation rules;
- Web/SSE material-change semantics;
- SQLite authority boundaries;
- repository `.specview.yaml` configuration semantics.

## Consequences

### Positive

- dramatically fewer small catalog rewrites during steady-state activity;
- no new catalog schema or migration burden;
- lifecycle durability remains immediate;
- graceful shutdown retains recent heartbeat history;
- crash-loss semantics are explicit and bounded.

### Negative

- persisted heartbeat timestamps may lag in-memory truth by up to 30 seconds during steady state;
- a hard crash can lose the latest heartbeat-only window;
- `Observe`'s `changed` result no longer implies persistence, which must remain documented and tested.
