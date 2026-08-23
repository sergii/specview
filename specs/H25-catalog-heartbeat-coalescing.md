---
specview:
  status: in_progress
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
- graceful runtime shutdown flushes pending heartbeat state;
- hard-crash heartbeat loss is bounded to one persistence window;
- persisted schema remains catalog version 1;
- reopening a catalog does not persist throttle metadata and may write the first new heartbeat immediately.

## Compatibility

H25 must leave unchanged:

- MCP contracts and binary smoke fixtures;
- HostSnapshot v1 and federation fixtures;
- H23 federation runtime/status behavior;
- Web/SSE material fingerprints;
- SQLite indexing authority;
- Host/repository/session logical identity;
- repository configuration format.

## Acceptance

- [ ] H24 heartbeat baseline test is replaced by coalescing tests.
- [ ] heartbeat-only observation inside 30s changes memory but not file bytes.
- [ ] heartbeat-only observation at 30s persists the newest timestamps.
- [ ] new session bypasses heartbeat throttle and persists immediately.
- [ ] session end bypasses heartbeat throttle and persists immediately.
- [ ] `Catalog.Flush` persists pending heartbeat state and is idempotent when clean.
- [ ] `Runtime.Run` flushes pending heartbeat state on graceful cancellation.
- [ ] no catalog schema-version change.
- [ ] existing hoststate tests pass under race detection.
- [ ] MCP and all federation built-binary smokes pass unchanged.
- [ ] Chromium semantic E2E passes unchanged.
- [ ] release archive cross-build passes.
- [ ] total production coverage remains above the repository gate.

## Expected write-rate change

For one continuously active Host with no lifecycle changes:

```text
v0.0.1 baseline: up to 1,800 catalog snapshots/hour
H25 target:       about 120 heartbeat snapshots/hour
```

Lifecycle writes are intentionally additional and immediate.

## Definition of done

H25 is done when heartbeat persistence is bounded and explicitly tested, lifecycle durability remains immediate, graceful shutdown flushes pending state, all public contracts remain unchanged, and exact-head CI is fully green.
