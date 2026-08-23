# ADR-012 - v0.0.1 Compatibility Debt Boundaries

## Status

Accepted for the v0.0.1 release-stabilization phase.

## Context

H12-H22 expanded Specview from the original single-repository Markdown dashboard into a Host-level, read-only control plane with normalized Execution, Evidence, Acceptance, MCP, Host identity, and federation.

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

The frozen read-only MCP contract already exposes a logical session `id` plus a separate `process_ids` diagnostic array. HostSnapshot v1 consumes the logical session projection and does not export PID as session identity.

### Migration target

After v0.0.1, a dedicated catalog/index migration may introduce a new historical execution schema based on stable logical execution-session identity and optional diagnostic process membership.

That change must be explicit because the persisted catalog and SQLite index are versioned data structures. It must not be smuggled into an unrelated feature slice.

## Decision 2 - Heartbeat persistence is an implementation concern, not a browser event contract

The Host observer currently updates in-memory `LastSeenAt` values during execution polling. The legacy JSON catalog may persist heartbeat-only updates more frequently than the rest of the material model requires.

The UI and SQLite layers already suppress heartbeat-only material changes through structural fingerprints. Therefore repeated Host catalog persistence is not part of the observable UI contract.

H23 must evaluate normal dogfooding disk behavior before the release is cut.

If heartbeat writes are materially noisy, H23 may add write coalescing/throttling without changing:

- execution discovery cadence;
- logical session semantics;
- Host material fingerprint semantics;
- SQLite authority boundaries;
- MCP/federation contracts.

If normal disk behavior is negligible at v0.0.1 scale, the persistence optimization may be deferred. The decision must be recorded with evidence rather than assumed.

## Decision 3 - Repository `server` fields are legacy v1 compatibility fields

The v1 `.specview.yaml` parser still accepts:

```yaml
server:
  host: 127.0.0.1
  port: 7331
```

The current Host observer binds its local UI independently, and federation Host/peer state is stored outside repositories. This makes Host networking conceptually Host-scoped rather than repository-scoped.

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
- Host-scoped configuration is not accidentally expanded inside repositories.

### Negative

- the first release intentionally carries some internal compatibility debt;
- the JSON catalog remains less elegant than the live Execution model;
- heartbeat persistence still requires a measured release decision;
- `.specview.yaml` v1 contains fields that are not the long-term Host configuration design.

## Release gate

These items are not automatically release blockers. H23 becomes blocked only if acceptance testing shows a correctness, safety, privacy, portability, or material performance defect caused by one of them.
