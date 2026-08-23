---
specview:
  status: done
---

# H25 - Host Catalog Heartbeat Coalescing

## Goal

Reduce steady-state Host catalog disk churn after v0.0.1 without changing live observation cadence, lifecycle durability, or public control-plane/federation contracts.

```text
2s Execution scan cadence
        ↓
in-memory heartbeat truth
        ↓
30s heartbeat persistence window
        ↓
immediate material/lifecycle persistence
```

## Scope

H25 is a persistence-only migration of the legacy Host catalog described by ADR-013 and ADR-014.

It does not introduce a new product plane or public protocol.

## Required semantics

- execution discovery remains on the existing cadence;
- active repository/session `LastSeenAt` advances in memory on every observation;
- first observation of a repository persists immediately;
- new session persists immediately;
- session end persists immediately;
- convention/detection-state changes persist immediately;
- heartbeat-only observations inside the 30-second window do not rewrite `catalog.json`;
- heartbeat-only observations at/after the 30-second boundary persist the latest in-memory timestamps;
- material persistence also flushes any pending heartbeat timestamps;
- failed material persistence remains dirty and retries immediately on the next observation/flush;
- graceful runtime shutdown flushes pending catalog state;
- hard-crash heartbeat loss is bounded to one persistence window;
- persisted schema remains catalog version 1;
- reopening a catalog does not persist throttle metadata and may write the first new heartbeat immediately.

## Compatibility

H25 leaves unchanged:

- MCP contracts and binary smoke fixtures;
- HostSnapshot v1 and federation fixtures;
- H23 federation runtime/status behavior;
- Web/SSE material fingerprints;
- SQLite indexing authority;
- Host/repository/session logical identity;
- repository configuration format.

## Acceptance

- [x] H24 heartbeat baseline test is replaced by coalescing tests.
- [x] heartbeat-only observation inside 30s changes memory but not file bytes.
- [x] heartbeat-only observation at 30s persists the newest timestamps.
- [x] new session bypasses heartbeat throttle and persists immediately.
- [x] session end bypasses heartbeat throttle and persists immediately.
- [x] failed material save remains dirty and retries without waiting for the heartbeat window.
- [x] `Catalog.Flush` persists pending state and is idempotent when clean.
- [x] `Runtime.Run` flushes pending heartbeat state on graceful cancellation.
- [x] no catalog schema-version change.
- [x] existing hoststate tests pass under race detection.
- [x] MCP and all federation built-binary smokes pass unchanged.
- [x] Chromium semantic E2E passes unchanged.
- [x] release archive cross-build passes.
- [x] total production coverage remains above the repository gate.

## Verification baseline

Functional head `d2efd5400599ccbaf1ce03ff953fc326116caeab` passed GitHub Actions CI #1044:

```text
total production statement coverage: 64.2%
internal/hoststate coverage:          60.4%
```

The same run passed gofmt, module verification, vet, race tests, build, MCP binary smoke, all federation binary smokes, Chromium semantic E2E, and release archive cross-build.

## Expected write-rate change

For one continuously active Host with no lifecycle changes:

```text
v0.0.1 baseline: up to 1,800 catalog snapshots/hour
H25 target:       about 120 heartbeat snapshots/hour
```

Lifecycle writes are intentionally additional and immediate.

## Definition of done

H25 is done: heartbeat persistence is bounded and explicitly tested, lifecycle durability remains immediate with retry on persistence failure, graceful shutdown flushes pending state, all public contracts remain unchanged, and the functional head passed the full CI matrix.
