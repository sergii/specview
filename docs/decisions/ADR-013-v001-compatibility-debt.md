# ADR-013 - v0.0.1 Compatibility Debt Boundaries

## Status

Accepted for the v0.0.1 release-stabilization phase.

## Context

H12-H23 expanded Specview from the original single-repository Markdown dashboard into a Host-level, read-only control plane with normalized Execution, Evidence, Acceptance, MCP, Host identity, federation, and a derived federation polling/status runtime.

Several early POC implementation details remain in internal compatibility layers. They should not be allowed to redefine the public product model, but changing them immediately before the first release can create more risk than keeping them explicit and bounded.

This ADR records the boundary for three known items.

## Decision 1 - Historical catalog sessions are not logical ExecutionSession identity

The live execution domain is authoritative:

```text
ExecutionAdapter
      ↓
ExecutionSession
      ├── logical ID
      ├── repository/worktree/cwd
      └── ProcessIDs diagnostics
```

The Host catalog currently persists historical session rows shaped around `agent + PID`.

That catalog representation is an internal compatibility/history format. It must not become:

- MCP logical execution identity;
- federation logical execution identity;
- cross-Host correlation identity;
- a requirement for future execution adapters.

The frozen read-only MCP contract exposes a logical session `id` plus a separate `process_ids` diagnostic array. HostSnapshot v1 consumes the logical session projection and does not export PID as session identity.

### Migration target

After v0.0.1, a dedicated catalog/index migration may introduce a new historical execution schema based on stable logical execution-session identity and optional diagnostic process membership.

That change must be explicit because the persisted catalog and SQLite index are versioned data structures. It must not be smuggled into an unrelated feature slice.

## Decision 2 - Heartbeat persistence is deferred after explicit characterization

The Host observer updates in-memory `LastSeenAt` values during execution polling. The legacy JSON catalog currently persists a heartbeat-only observation as a new atomic catalog snapshot.

H24 adds `TestCatalogHeartbeatPersistenceBaseline` to make that behavior explicit rather than assumed. With the current two-second Host execution scan interval, a continuously observed active Host can therefore perform up to:

```text
30 catalog snapshots / minute
1,800 catalog snapshots / hour
```

The write rate is per catalog refresh, not multiplied by every browser client. The catalog is a small local JSON compatibility/history file written through a temporary file plus atomic rename. SQLite, Web material fingerprints, and H23 federation material fingerprints already suppress heartbeat/transport-only changes, so this persistence does not create repeated SQLite rewrites, browser fragment refreshes, or federation status changes.

### v0.0.1 release decision

Defer write coalescing until after v0.0.1.

Reasoning:

- the behavior is covered by an explicit baseline test;
- it does not change logical Execution semantics;
- it does not amplify into browser, SQLite, or federation material-update traffic;
- no correctness, safety, privacy, or release-gate failure is caused by it at current POC scale;
- changing persistence cadence immediately before the first release would alter crash/restart history semantics without a migration-specific acceptance slice.

The current behavior is not considered desirable long-term. It is accepted as bounded implementation debt for the first POC release.

### Migration target

After v0.0.1, add an explicit Host-catalog persistence slice that can coalesce or throttle heartbeat-only snapshots while preserving:

- immediate persistence of repository/session lifecycle changes;
- useful crash/restart history semantics;
- execution discovery cadence;
- logical session semantics;
- Host and federation material-fingerprint semantics;
- SQLite authority boundaries;
- MCP/federation contracts.

The migration should replace the H24 baseline test with tests for the chosen coalescing semantics.

## Decision 3 - Repository `server` fields are legacy v1 compatibility fields

The v1 `.specview.yaml` parser still accepts:

```yaml
server:
  host: 127.0.0.1
  port: 7331
```

The current Host observer binds its local UI independently, and federation Host/peer/runtime state is stored outside repositories. This makes Host networking conceptually Host-scoped rather than repository-scoped.

For v0.0.1:

- keep the v1 fields for compatibility;
- do not add new Host settings under the repository `server` section;
- do not imply that repository configuration owns federation or future Host networking;
- document the section as a compatibility artifact.

### Migration target

After v0.0.1, Host-level settings should move to Host-level configuration or explicit CLI/environment configuration.

Removal of repository `server` fields requires either:

1. a versioned repository configuration contract, or
2. a backwards-compatible deprecation reader that can consume v1 while no longer generating the legacy fields.

## Consequences

### Positive

- the domain model stays cleaner than the early POC persistence details;
- v0.0.1 avoids unnecessary schema churn immediately before release;
- future migrations have explicit targets;
- public MCP/federation contracts remain logical-session based;
- Host-scoped configuration is not accidentally expanded inside repositories;
- heartbeat persistence debt is quantified and regression-characterized rather than hidden.

### Negative

- the first release intentionally carries some internal compatibility debt;
- the JSON catalog remains less elegant than the live Execution model;
- a continuously active Host may still rewrite the small JSON catalog every two seconds;
- `.specview.yaml` v1 contains fields that are not the long-term Host configuration design.

## Release gate

These items are not release blockers for v0.0.1 unless later installed-product acceptance exposes a correctness, safety, privacy, portability, or material performance defect caused by one of them.
