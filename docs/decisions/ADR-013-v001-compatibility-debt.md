# ADR-013 - v0.0.1 Compatibility Debt Boundaries

## Status

Resolved post-v0.0.1. Retained as the historical release-boundary record.

## Resolution

All three migration targets recorded before v0.0.1 were retired through dedicated post-release slices:

1. **Historical execution identity — H26 / ADR-015.** Host catalog v2 and derived SQLite schema v2 persist logical `ExecutionSession` identity with `process_ids` as diagnostics. Catalog v1 remains readable through a deterministic migration path.
2. **Heartbeat persistence — H25 / ADR-014.** Heartbeat-only catalog persistence is coalesced to a 30-second window while lifecycle/material changes remain immediately durable and graceful shutdown flushes pending state.
3. **Repository `server` fields — H27 / ADR-016.** Repository config v2 removes Host networking from the canonical writer. Existing valid v1 files remain readable and are not rewritten; v2 rejects a repository `server` section.

The public MCP, federation, Evidence, Acceptance and HostSnapshot authority boundaries remained unchanged across these migrations.

## Context

H12-H23 expanded Specview from the original single-repository Markdown dashboard into a Host-level, read-only control plane with normalized Execution, Evidence, Acceptance, MCP, Host identity, federation, and a derived federation polling/status runtime.

Several early POC implementation details remained in internal compatibility layers. They were not allowed to redefine the public product model, but changing them immediately before the first release would have created more risk than keeping them explicit and bounded.

This ADR recorded the boundary for three known items before v0.0.1.

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

At the v0.0.1 boundary, the Host catalog persisted historical session rows shaped around `agent + PID`.

That representation was classified as an internal compatibility/history format. It was never allowed to become:

- MCP logical execution identity;
- federation logical execution identity;
- cross-Host correlation identity;
- a requirement for future execution adapters.

The frozen read-only MCP contract exposed a logical session `id` plus a separate `process_ids` diagnostic array. HostSnapshot v1 consumed the logical session projection and did not export PID as session identity.

### Migration target

A dedicated catalog/index migration was required to introduce a historical execution schema based on stable logical execution-session identity and optional diagnostic process membership.

**Resolved:** H26 / ADR-015 introduced catalog v2 and SQLite schema v2 with deterministic v1 history migration.

## Decision 2 - Heartbeat persistence was deferred after explicit characterization

The Host observer updates in-memory `LastSeenAt` values during execution polling. At the v0.0.1 boundary, the legacy JSON catalog persisted a heartbeat-only observation as a new atomic catalog snapshot.

H24 added `TestCatalogHeartbeatPersistenceBaseline` to make that behavior explicit rather than assumed. With the two-second Host execution scan interval, a continuously observed active Host could therefore perform up to:

```text
30 catalog snapshots / minute
1,800 catalog snapshots / hour
```

The write rate was per catalog refresh, not multiplied by every browser client. SQLite, Web material fingerprints, and federation material fingerprints already suppressed heartbeat/transport-only changes, so this persistence did not create repeated SQLite rewrites, browser fragment refreshes, or federation status changes.

### v0.0.1 release decision

Write coalescing was deferred until after v0.0.1 because the behavior was characterized, did not change logical Execution semantics, and was not a correctness or release blocker at POC scale.

### Migration target

The post-release migration had to preserve:

- immediate persistence of repository/session lifecycle changes;
- useful crash/restart history semantics;
- execution discovery cadence;
- logical session semantics;
- Host and federation material-fingerprint semantics;
- SQLite authority boundaries;
- MCP/federation contracts.

**Resolved:** H25 / ADR-014 coalesces heartbeat-only persistence to a 30-second window, retries failed material persistence immediately, and flushes pending state on graceful shutdown.

## Decision 3 - Repository `server` fields were legacy v1 compatibility fields

The v1 `.specview.yaml` parser accepts:

```yaml
server:
  host: 127.0.0.1
  port: 7331
```

The Host observer binds its local UI independently, and federation Host/peer/runtime state is stored outside repositories. This makes Host networking Host-scoped rather than repository-scoped.

For v0.0.1 the decision was:

- keep the v1 fields for compatibility;
- do not add new Host settings under the repository `server` section;
- do not imply that repository configuration owns federation or future Host networking;
- document the section as a compatibility artifact.

### Migration target

Removal from the canonical writer required either a versioned repository configuration contract or a backwards-compatible deprecation reader.

**Resolved:** H27 / ADR-016 introduces repository config v2. `specview init` writes v2 without `server`; valid v1 remains readable and unchanged; v2 with `server` fails closed.

## Consequences

### Positive

- the domain model remains cleaner than the early POC persistence details;
- v0.0.1 avoided unnecessary schema churn immediately before release;
- each debt item received a dedicated migration with explicit acceptance criteria;
- public MCP/federation contracts remained logical-session based;
- Host-scoped networking is no longer generated inside repository Intent configuration;
- heartbeat persistence was quantified before optimization and then reduced without changing discovery cadence.

### Compatibility retained intentionally

- the catalog v1 reader remains so v0.0.1 history can migrate deterministically;
- the repository config v1 reader remains so existing repositories do not require forced rewrites;
- these readers are compatibility surfaces, not the canonical writer formats.

## Release gate history

None of these items blocked v0.0.1. They were explicitly classified before release and subsequently resolved through H25-H27 rather than being mixed into unrelated feature work.
