# ADR-012: Federation runtime and multi-host projection

Status: Accepted

## Context

H20 defined immutable `HostSnapshot` v1 and deterministic cross-host aggregation. H21 transported snapshots over a narrow read-only HTTP boundary. H22 added explicit peer configuration, credential references, last-known remote observations, and deterministic freshness states.

Specview now needs to keep configured peers reasonably current while the host dashboard is running and expose one read-only view across the local Host and known remote Hosts.

## Decision

### Polling is derived runtime behavior

The federation polling runtime is not a synchronization authority. It periodically:

1. re-reads the Host-level peer registry from disk;
2. refreshes each currently configured peer through the unchanged H22 `Refresher`;
3. stores success/failure through the unchanged H22 observation store;
4. invokes an optional material-change callback.

The runtime does not modify `HostSnapshot` v1, repository correlation rules, source `observed_at`, peer identity, or remote state.

### Registry is re-opened every cycle

A long-running `specview serve` process must observe `specview federation peer add/remove` performed by another process without restart. Therefore the runtime re-opens the versioned peer registry on each cycle instead of treating an in-memory registry as authoritative forever.

### Initial cadence

H23 uses one conservative fixed poll interval for all peers. Poll cadence is runtime policy, not part of peer v1 persistence. Per-peer scheduling can be introduced later without changing peer or snapshot contracts.

### Multi-host projection

A new read-only projection service combines:

- one freshly built local `HostSnapshot`;
- every configured peer's projected freshness metadata;
- the last valid remote `HostSnapshot`, when available;
- the existing H20 aggregator over snapshots that actually exist.

Remote `unreachable` does not mean empty or inactive. If a last-known snapshot exists, it remains in the aggregate and is explicitly source-attributed as unreachable/stale metadata outside the snapshot itself.

`never_retrieved` peers appear in Host status but contribute no snapshot to repository aggregation.

### Failure isolation

One peer failure must not prevent other peers or the local Host from appearing. Registry decode failure is surfaced as a projection/runtime error because the configured peer set is then not trustworthy.

### No credential leakage

The runtime delegates credentials and error redaction to H22. Multi-host projections expose credential reference metadata only when explicitly needed; they never expose resolved secret values.

## Consequences

- `specview serve` can keep peer observations current without a separate distributed system.
- CLI peer changes are picked up without server restart.
- The same projection can back CLI, MCP, and Web surfaces later.
- Last-known remote facts remain visible during outages and are never silently reclassified as current local facts.
- H20-H22 language-neutral contracts remain unchanged.

## Out of scope

- peer discovery;
- push synchronization;
- remote execution or mutation;
- distributed locks or shared databases;
- per-peer poll schedules;
- WebSocket federation;
- changing repository correlation semantics.
